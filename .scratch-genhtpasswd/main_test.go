package scratch

import (
	"os"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestGenerate(t *testing.T) {
	h, err := bcrypt.GenerateFromPassword([]byte("playwright-scan"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("/work/repo/.scratch-genhtpasswd/out.txt", h, 0644); err != nil {
		t.Fatal(err)
	}
}
