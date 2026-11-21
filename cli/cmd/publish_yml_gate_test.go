package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/audit"
	"gopkg.in/yaml.v3"
)

// Rule: Prohibido usar 'else'
// Criterio de éxito de la ficha uso: una estable pasa a latest solo con sus horas de uso medidas. Lo decide VER-05 dentro
// de «qdd audit»; esta prueba exige que el job que hace npm publish no corra sin que antes pase ese gate.

type pasoDeWorkflow struct {
	Name            string `yaml:"name"`
	Run             string `yaml:"run"`
	If              string `yaml:"if"`
	ContinueOnError any    `yaml:"continue-on-error"`
}

type jobDeWorkflow struct {
	Needs           any              `yaml:"needs"`
	If              string           `yaml:"if"`
	ContinueOnError any              `yaml:"continue-on-error"`
	Steps           []pasoDeWorkflow `yaml:"steps"`
}

func (j jobDeWorkflow) necesita() []string {
	switch n := j.Needs.(type) {
	case string:
		return []string{n}
	case []any:
		var lista []string
		for _, x := range n {
			lista = append(lista, x.(string))
		}
		return lista
	}
	return nil
}

func tienePaso(j jobDeWorkflow, comando string) bool {
	for _, p := range j.Steps {
		if strings.Contains(p.Run, comando) && p.If == "" && p.ContinueOnError == nil {
			return true
		}
	}
	return false
}

func TestPublishYml_NpmPublishCorreSoloDespuesDelGateConVER05(t *testing.T) {
	datos, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "publish.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var flujo struct {
		Jobs map[string]jobDeWorkflow `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(datos, &flujo); err != nil {
		t.Fatal(err)
	}
	var publica []string
	for nombre, j := range flujo.Jobs {
		if tienePaso(j, "npm publish") {
			publica = append(publica, nombre)
		}
	}
	if len(publica) != 1 {
		t.Fatalf("un solo job publica en npm: %v", publica)
	}
	// Sube por needs hasta el job que corre «make audit». Un if (always(), failure()) o un continue-on-error en el
	// camino dejaría publicar con el gate en rojo.
	vistos := map[string]bool{}
	pendientes := []string{publica[0]}
	gate := ""
	for len(pendientes) > 0 {
		nombre := pendientes[0]
		pendientes = pendientes[1:]
		if vistos[nombre] {
			continue
		}
		vistos[nombre] = true
		j, ok := flujo.Jobs[nombre]
		if !ok {
			t.Fatalf("needs nombra un job que no existe: %s", nombre)
		}
		if j.If != "" || j.ContinueOnError != nil {
			t.Fatalf("el job %s tiene if o continue-on-error: puede correr aunque el gate falle", nombre)
		}
		if nombre != publica[0] && tienePaso(j, "make audit") {
			gate = nombre
		}
		pendientes = append(pendientes, j.necesita()...)
	}
	if gate == "" {
		t.Fatalf("el job %s no depende (needs) de un job que corra «make audit»", publica[0])
	}
	makefile, err := os.ReadFile(filepath.Join("..", "..", "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	receta := regexp.MustCompile(`(?m)^audit:.*\n((?:\t.*\n)+)`).FindStringSubmatch(string(makefile))
	if receta == nil || !regexp.MustCompile(`(?m)^\tqdd audit\s*$`).MatchString(receta[1]) {
		t.Fatalf("«make audit» corre «qdd audit» tal cual (sin || true ni -): %q", receta)
	}
	p := audit.LoadPolicies(filepath.Join("..", ".."))
	if !p.VersioningEnabled() || p.UsoMinimoHoras <= 0 {
		t.Fatalf("la política de qdd corre VER-05 con horas de uso: versioning %v, uso_minimo_horas %v", p.VersioningEnabled(), p.UsoMinimoHoras)
	}
}
