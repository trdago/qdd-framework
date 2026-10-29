package audit

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
)

var categoriasValidas = map[string]bool{"seguridad": true, "correccion": true, "auditabilidad": true, "madurez": true}

// RFC 0005 §3.1: una regla sin una razón que se pueda defender no se publica.
// Cada principio tiene nombre, por_que y una categoría válida.
func TestPrincipios_TienenRazonYCategoria(t *testing.T) {
	ps := Principios()
	if len(ps) == 0 {
		t.Fatal("el catálogo de principios está vacío")
	}
	vistos := map[string]bool{}
	for _, p := range ps {
		if vistos[p.ID] {
			t.Errorf("principio %s repetido", p.ID)
		}
		vistos[p.ID] = true
		if p.Nombre == "" || p.PorQue == "" {
			t.Errorf("%s sin nombre o sin por_que", p.ID)
		}
		if !categoriasValidas[p.Categoria] {
			t.Errorf("%s: categoría %q no válida", p.ID, p.Categoria)
		}
		if len(p.Reglas) == 0 {
			t.Errorf("%s no agrupa ninguna regla", p.ID)
		}
		if _, ok := detectores[p.ID]; !ok {
			t.Errorf("%s no declara su detector en detectores", p.ID)
		}
	}
	for id := range detectores {
		if !vistos[id] {
			t.Errorf("detectores declara %s, que no está en el catálogo", id)
		}
	}
}

// Toda regla que el motor emite pertenece a un principio. Si una regla nueva
// no tiene principio, su violación saldría sin por_que ni categoría.
func TestPrincipios_CadaReglaDelMotorTienePrincipio(t *testing.T) {
	ids := map[string]bool{}
	categorias := map[string]bool{}
	patronCategoria := regexp.MustCompile(`Category:\s*"([^"]+)"`)
	patron := regexp.MustCompile(`"((?:OWASP|QA|CLEAN|12F|CI|HEART|BP|DB|CERT|TRC|ISO\d+|SOC2)-[A-Z0-9]+(?:-[A-Z0-9]+)*)"`)
	for _, dir := range []string{".", "../blueprints"} {
		archivos, _ := filepath.Glob(filepath.Join(dir, "*.go"))
		for _, a := range archivos {
			if filepath.Ext(a) != ".go" || regexp.MustCompile(`_test\.go$`).MatchString(a) {
				continue
			}
			data, _ := os.ReadFile(a)
			for _, m := range patron.FindAllStringSubmatch(string(data), -1) {
				ids[m[1]] = true
			}
			for _, m := range patronCategoria.FindAllStringSubmatch(string(data), -1) {
				categorias[m[1]] = true
			}
		}
	}
	if len(ids) < 20 {
		t.Fatalf("solo se encontraron %d reglas en el código; el patrón dejó de reconocerlas", len(ids))
	}
	var sinPrincipio []string
	for id := range ids {
		if categorias[id] {
			continue
		}
		if _, ok := PrincipioDeRegla(id); !ok {
			sinPrincipio = append(sinPrincipio, id)
		}
	}
	sort.Strings(sinPrincipio)
	if len(sinPrincipio) > 0 {
		t.Errorf("reglas sin principio: %v", sinPrincipio)
	}
}

// RFC 0005 §3.1: por_que y categoria aparecen en cada violación.
func TestRunAll_ViolacionesConPrincipio(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc f(a bool) int {\n\tif a {\n\t\treturn 1\n\t} else {\n\t\treturn 2\n\t}\n}\n"), 0644)

	var encontrada bool
	for _, v := range NewEngine(dir).RunAll() {
		if v.Principio == "" || v.PorQue == "" || v.Categoria == "" {
			t.Errorf("%s sin principio, por_que o categoría: %+v", v.RuleID, v)
		}
		if v.RuleID == "CLEAN-01-NO-ELSE" {
			encontrada = true
			if v.Principio != "P-READABLE-FLOW" || v.Categoria != "auditabilidad" {
				t.Errorf("CLEAN-01 = %+v; want P-READABLE-FLOW, auditabilidad", v)
			}
		}
	}
	if !encontrada {
		t.Error("el motor no detectó el else de main.go")
	}
}
