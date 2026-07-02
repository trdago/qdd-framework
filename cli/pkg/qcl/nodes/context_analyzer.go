package nodes

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	
	"github.com/qdd-framework/qdd/pkg/qcl/models"
)

type ContextAnalyzer struct{}

func NewContextAnalyzer() *ContextAnalyzer {
	return &ContextAnalyzer{}
}

func (c *ContextAnalyzer) Process(session *models.CognitiveSession) error {
	fmt.Println("🤖 [QCL] ContextAnalyzer: Evaluando contexto del proyecto...")
	
	qddDir := filepath.Join(".", ".qdd")
	configPath := filepath.Join(qddDir, "config.yaml")
	
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err // Should not happen if Gatekeeper already passed
	}

	// We can reuse a local struct here or import the one from qcl
	type ConfigFile struct {
		Project      string   `yaml:"project"`
		Languages    []string `yaml:"languages"`
		Databases    []string `yaml:"databases"`
		Architecture string   `yaml:"architecture"`
	}

	var config ConfigFile
	if err := yaml.Unmarshal(content, &config); err != nil {
		return err
	}

	session.Context = &models.ProjectContext{
		HasState: true,
		// We can add fields to ProjectContext later to hold languages, db, arch
	}

	fmt.Printf("   -> Contexto verificado (Lenguaje: %v, DB: %v, Arch: %s)\n", config.Languages, config.Databases, config.Architecture)
	
	return nil
}
