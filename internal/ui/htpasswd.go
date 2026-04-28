package ui

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// VerifyHtpasswdLine reports whether the given (plain) password
// matches the credential `entry` in htpasswd format. Bcrypt entries
// (`$2y$`, `$2a$`, `$2b$`) are verified via crypto/bcrypt; every
// other algorithm (`$apr1$` MD5, `{SHA}` SHA-1, `$5$` / `$6$`
// crypt) is unsupported — the loader rejects them at startup and
// this function returns false defensively.
//
// We deliberately ship only one algorithm so the comparison is a
// single battle-tested code path. Operators with legacy htpasswd
// files generate a fresh bcrypt entry with `htpasswd -B -c file
// user`.
func VerifyHtpasswdLine(entry, plain string) bool {
	switch {
	case strings.HasPrefix(entry, "$2a$"),
		strings.HasPrefix(entry, "$2b$"),
		strings.HasPrefix(entry, "$2y$"):
		return bcrypt.CompareHashAndPassword([]byte(entry), []byte(plain)) == nil
	}
	return false
}

// LoadHtpasswd reads path and returns a map of user → entry. It
// rejects any entry whose hash uses an unsupported algorithm so an
// operator who copies an old apr1 file gets a startup error
// instead of a silent always-deny.
func LoadHtpasswd(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("htpasswd: open %s: %w", path, err)
	}
	defer f.Close()
	out := map[string]string{}
	sc := bufio.NewScanner(f)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.IndexByte(line, ':')
		if i <= 0 {
			return nil, fmt.Errorf("htpasswd: line %d: missing colon", lineNo)
		}
		user, entry := line[:i], line[i+1:]
		if err := assertSupportedAlgo(entry); err != nil {
			return nil, fmt.Errorf("htpasswd: line %d: %w", lineNo, err)
		}
		out[user] = entry
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("htpasswd: read: %w", err)
	}
	return out, nil
}

func assertSupportedAlgo(entry string) error {
	switch {
	case strings.HasPrefix(entry, "$2a$"),
		strings.HasPrefix(entry, "$2b$"),
		strings.HasPrefix(entry, "$2y$"):
		return nil
	case strings.HasPrefix(entry, "$apr1$"):
		return errors.New("MD5-based htpasswd entry ($apr1$) is unsupported; use bcrypt (htpasswd -B)")
	case strings.HasPrefix(entry, "{SHA}"):
		return errors.New("SHA-1 ({SHA}) htpasswd entry is unsupported; use bcrypt (htpasswd -B)")
	case strings.HasPrefix(entry, "$5$"):
		return errors.New("SHA-256 crypt is unsupported; use bcrypt (htpasswd -B)")
	case strings.HasPrefix(entry, "$6$"):
		return errors.New("SHA-512 crypt is unsupported; use bcrypt (htpasswd -B)")
	}
	return errors.New("unsupported htpasswd algorithm; use bcrypt (htpasswd -B)")
}
