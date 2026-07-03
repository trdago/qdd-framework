package qcl

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type ConfigFile struct {
	Project      string   `yaml:"project"`
	Languages    []string `yaml:"languages"`
	Databases    []string `yaml:"databases"`
	Architecture string   `yaml:"architecture"`
}

func CheckMinimumAlignment() error {
	qddDir := filepath.Join(".", ".qdd")
	if _, err := os.Stat(qddDir); os.IsNotExist(err) {
		return errors.New("No existe la carpeta .qdd/. Alerta: Debes ejecutar `qdd init`")
	}

	configPath := filepath.Join(qddDir, "config.yaml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		return errors.New("No se pudo leer .qdd/config.yaml")
	}

	var config ConfigFile
	if err := yaml.Unmarshal(content, &config); err != nil {
		return errors.New("El archivo .qdd/config.yaml tiene un formato inválido")
	}

	if len(config.Languages) == 0 {
		return errors.New("Contexto insuficiente. QDD se niega a operar a ciegas. Por favor declara el lenguaje (languages) en config.yaml o ejecuta `qdd learn`")
	}

	if config.Architecture == "" {
		return errors.New("Contexto insuficiente. QDD se niega a operar a ciegas. Por favor declara el patrón arquitectónico (architecture) en config.yaml o ejecuta `qdd learn`")
	}

	return nil
}
