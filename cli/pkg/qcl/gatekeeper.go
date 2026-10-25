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
		return "", errors.New("Este proyecto aún no usa QDD (no hay carpeta .qdd/ aquí ni en las carpetas superiores). Ejecuta `qdd init` en la raíz del proyecto")
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
		return errors.New("QDD aún no sabe en qué lenguaje está el proyecto (no encontró go.mod, package.json, requirements.txt, pubspec.yaml...). Decláralo con:\n    qdd init --lenguaje Go --arquitectura \"API REST\"\n  o pide a tu IA que ejecute qdd_learn")
	}

	if config.Architecture == "" {
		return errors.New("QDD aún no sabe qué tipo de proyecto es. Decláralo con:\n    qdd init --arquitectura \"API REST\"\n  o pide a tu IA que ejecute qdd_learn")
	}

	return nil
}
