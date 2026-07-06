package audit

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type QDDPolicies struct {
	OWASP        bool `yaml:"owasp" json:"owasp"`
	CleanCode    bool `yaml:"clean_code" json:"clean_code"`
	ZeroElse     bool `yaml:"zero_else" json:"zero_else"`
	BeyondLimits bool `yaml:"beyond_limits" json:"beyond_limits"`
	Traceability   bool `yaml:"traceability" json:"traceability"`
	Enterprise     bool `yaml:"enterprise" json:"enterprise"`
	AllowExecution bool `yaml:"allow_execution" json:"allow_execution"`
}

// DefaultPolicies returns the default strict QDD configuration
func DefaultPolicies() QDDPolicies {
	return QDDPolicies{
		OWASP:          true,
		CleanCode:      true,
		ZeroElse:       true,
		BeyondLimits:   true,
		Traceability:   true,
		Enterprise:     true,
		AllowExecution: true,
	}
}

// LoadPolicies loads the policies from .qdd/policies.yaml, or returns default if not found
func LoadPolicies(cwd string) QDDPolicies {
	policiesPath := filepath.Join(cwd, ".qdd", "policies.yaml")
	data, err := os.ReadFile(policiesPath)
	if err != nil {
		return DefaultPolicies()
	}

	var p QDDPolicies
	err = yaml.Unmarshal(data, &p)
	if err != nil {
		return DefaultPolicies()
	}
	return p
}

// SavePolicies saves the given policies to .qdd/policies.yaml
func SavePolicies(cwd string, p QDDPolicies) error {
	qddDir := filepath.Join(cwd, ".qdd")
	if _, err := os.Stat(qddDir); os.IsNotExist(err) {
		os.MkdirAll(qddDir, 0755)
	}

	policiesPath := filepath.Join(qddDir, "policies.yaml")
	data, err := yaml.Marshal(p)
	if err != nil {
		return err
	}

	return os.WriteFile(policiesPath, data, 0644)
}
