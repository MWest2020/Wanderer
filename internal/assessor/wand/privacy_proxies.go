package wand

import (
	_ "embed"
	"fmt"
	"strings"
	"sync"

	"go.yaml.in/yaml/v2"
)

//go:embed privacy_proxies.yaml
var privacyProxiesYAML []byte

// PrivacyProxy is one entry from privacy_proxies.yaml: a named
// commercial service that hides the real registrant behind its own
// identity.
type PrivacyProxy struct {
	Name string `yaml:"name"`
}

type privacyProxiesFile struct {
	Proxies []PrivacyProxy `yaml:"proxies"`
}

var (
	privacyProxiesOnce sync.Once
	privacyProxiesList []PrivacyProxy
	privacyProxiesErr  error
)

func loadPrivacyProxies() ([]PrivacyProxy, error) {
	privacyProxiesOnce.Do(func() {
		var f privacyProxiesFile
		if err := yaml.UnmarshalStrict(privacyProxiesYAML, &f); err != nil {
			privacyProxiesErr = fmt.Errorf("wand: parse privacy_proxies.yaml: %w", err)
			return
		}
		privacyProxiesList = f.Proxies
	})
	return privacyProxiesList, privacyProxiesErr
}

// PrivacyProxyEntries returns the loaded list (for tests).
func PrivacyProxyEntries() []PrivacyProxy {
	list, _ := loadPrivacyProxies()
	return list
}

// MatchPrivacyProxy reports whether name matches a known commercial
// privacy-proxy service (case-insensitive substring match), returning
// the matched entry's canonical name. A miss is not a failure — see
// the package doc comment on privacy_proxies.yaml.
func MatchPrivacyProxy(name string) (string, bool) {
	list, err := loadPrivacyProxies()
	if err != nil || name == "" {
		return "", false
	}
	low := strings.ToLower(name)
	for _, p := range list {
		if strings.Contains(low, strings.ToLower(p.Name)) {
			return p.Name, true
		}
	}
	return "", false
}
