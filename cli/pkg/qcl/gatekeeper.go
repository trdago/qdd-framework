package qcl

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type ConfigFile struct {
	Project      string   `yaml:"project"`
	Languages             []string `yaml:"languages"`
	Databases             []string `yaml:"databases"`
	Architecture          string   `yaml:"architecture"`
	AutoUICertification   bool     `yaml:"auto_ui_certification"`
}

func CheckMinimumAlignment() error {
	configPath, err := checkQDDDir()
	if err != nil {
		return err
	}

	config, err := readConfigFile(configPath)
	if err != nil {
		return err
	}

	return validateConfigAlignment(config)
}

func checkQDDDir() (string, error) {
	qddDir := filepath.Join(".", ".qdd")
	if _, err := os.Stat(qddDir); os.IsNotExist(err) {
		return "", errors.New("No existe la carpeta .qdd/. Alerta: Debes ejecutar `qdd init`")
	}
	return filepath.Join(qddDir, "config.yaml"), nil
}

func readConfigFile(configPath string) (*ConfigFile, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, errors.New("No se pudo leer .qdd/config.yaml")
	}

	var config ConfigFile
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, errors.New("El archivo .qdd/config.yaml tiene un formato inválido")
	}
	return &config, nil
}

func validateConfigAlignment(config *ConfigFile) error {
	if len(config.Languages) == 0 {
		return errors.New("Contexto insuficiente. QDD se niega a operar a ciegas. Por favor declara el lenguaje (languages) en config.yaml, o pide a tu asistente de IA conectado por MCP que ejecute la tool `qdd_learn`")
	}

	if config.Architecture == "" {
		return errors.New("Contexto insuficiente. QDD se niega a operar a ciegas. Por favor declara el patrón arquitectónico (architecture) en config.yaml, o pide a tu asistente de IA conectado por MCP que ejecute la tool `qdd_learn`")
	}

	return nil
}
