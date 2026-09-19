package wand

import (
	_ "embed"
	"fmt"
	"strings"
	"sync"

	"go.yaml.in/yaml/v2"
)

//go:embed registry_redaction.yaml
var registryRedactionYAML []byte

// RegistryRedaction is one entry from registry_redaction.yaml: a TLD
// whose registry redacts registrant data for every domain under it.
type RegistryRedaction struct {
	TLD      string `yaml:"tld"`
	Registry string `yaml:"registry"`
}

type registryRedactionFile struct {
	Redactions []RegistryRedaction `yaml:"redactions"`
}

var (
	registryRedactionOnce sync.Once
	registryRedactionList []RegistryRedaction
	registryRedactionErr  error
)

func loadRegistryRedaction() ([]RegistryRedaction, error) {
	registryRedactionOnce.Do(func() {
		var f registryRedactionFile
		if err := yaml.UnmarshalStrict(registryRedactionYAML, &f); err != nil {
			registryRedactionErr = fmt.Errorf("wand: parse registry_redaction.yaml: %w", err)
			return
		}
		registryRedactionList = f.Redactions
	})
	return registryRedactionList, registryRedactionErr
}

// RegistryRedactionEntries returns the loaded list (for tests).
func RegistryRedactionEntries() []RegistryRedaction {
	list, _ := loadRegistryRedaction()
	return list
}

// RegistryRedactionFor looks up the registry that redacts registrant
// data for every domain under tld (matched case-insensitively,
// leading dot optional). Returns the registry name and true when the
// TLD is on the list.
func RegistryRedactionFor(tld string) (string, bool) {
	list, err := loadRegistryRedaction()
	if err != nil {
		return "", false
	}
	tld = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(tld), "."))
	if tld == "" {
		return "", false
	}
	for _, r := range list {
		if strings.EqualFold(r.TLD, tld) {
			return r.Registry, true
		}
	}
	return "", false
}
