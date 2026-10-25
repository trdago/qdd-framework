package cmd

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// completarConfig fija lenguaje y arquitectura en un config.yaml existente sin
// tocar el resto (ni los comentarios). Sin forzar, solo completa lo vacío: la
// detección nunca pisa lo que el usuario declaró.
func completarConfig(configPath string, meta ProjectMetadata, forzar bool) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil
	}
	raiz := doc.Content[0]
	cambio := false
	if len(meta.Languages) > 0 && (forzar || valorVacio(raiz, "languages")) {
		lista := &yaml.Node{Kind: yaml.SequenceNode, Style: yaml.FlowStyle}
		for _, l := range meta.Languages {
			lista.Content = append(lista.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: l})
		}
		fijarClave(raiz, "languages", lista)
		cambio = true
	}
	if meta.Architecture != "" && (forzar || valorVacio(raiz, "architecture")) {
		fijarClave(raiz, "architecture", &yaml.Node{Kind: yaml.ScalarNode, Value: meta.Architecture})
		cambio = true
	}
	if !cambio {
		return nil
	}
	out, err := yaml.Marshal(&doc)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, out, 0644)
}

func buscarClave(m *yaml.Node, key string) int {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return i
		}
	}
	return -1
}

func valorVacio(m *yaml.Node, key string) bool {
	i := buscarClave(m, key)
	if i < 0 {
		return true
	}
	v := m.Content[i+1]
	return len(v.Content) == 0 && strings.TrimSpace(v.Value) == ""
}

func fijarClave(m *yaml.Node, key string, valor *yaml.Node) {
	i := buscarClave(m, key)
	if i < 0 {
		m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, valor)
		return
	}
	m.Content[i+1] = valor
}
