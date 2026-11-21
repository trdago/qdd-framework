package puertas

import (
	"errors"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// Contrato de la ficha puertas: decidir es una función de los hechos, sin modelos ni red.

// Cada puerta con hechos que la hacen pasar, rechazar, no aplicar o fallar: decidir dos veces lo mismo da lo mismo,
// y si pasa la IA no recibe nada (0 tokens).
// contrato: puertas#13be0fdb
func TestDecidir_LaMismaAccionConLosMismosHechosDaLoMismo(t *testing.T) {
	activa := Politica{Activa: true}
	commit := Accion{Puerta: PuertaAudit}
	fix := Accion{Puerta: PuertaFinding}
	editar := Accion{Puerta: PuertaFicha, Archivo: "cli/pkg/baul/verificar.go"}
	casos := []Hechos{
		{Accion: commit, Politica: Politica{}},
		{Accion: commit, Politica: activa},
		{Accion: commit, Politica: activa, Marca: "a", Huella: "a"},
		{Accion: commit, Politica: activa, Marca: "a", Huella: "b"},
		{Accion: commit, Politica: activa, ErrHuella: errors.New("git falló")},
		{Accion: Accion{Puerta: PuertaSalto}, Politica: activa},
		{Accion: Accion{Puerta: PuertaPublicar, Que: "gh pr merge"}, Politica: activa},
		{Accion: Accion{Puerta: PuertaPublicar, Que: "npm publish"}, Politica: Politica{Activa: true, Publicar: PublicarIA}},
		{Accion: fix, Politica: activa, Mensaje: "feat: x"},
		{Accion: fix, Politica: activa, Mensaje: "fix: x"},
		{Accion: fix, Politica: activa, Mensaje: "fix: x (FND-9)", Findings: map[string]bool{"FND-9": true}},
		{Accion: fix, Politica: activa, Mensaje: "fix: x (FND-8)", Findings: map[string]bool{"FND-9": true}},
		{Accion: editar, Politica: activa},
		{Accion: editar, Politica: activa, Ficha: "baul"},
		{Accion: editar, Politica: activa, Ficha: "baul", Leida: true},
	}
	resultados := map[string]bool{}
	for i, h := range casos {
		primera := Decidir(h)
		for n := 0; n < 20; n++ {
			if otra := Decidir(h); !reflect.DeepEqual(otra, primera) {
				t.Fatalf("caso %d: la misma acción con los mismos hechos dio %+v y luego %+v", i, primera, otra)
			}
		}
		if primera.Resultado != Rechaza && primera.Mensaje != "" {
			t.Errorf("caso %d: si no rechaza, la IA no lee nada (0 tokens): %+v", i, primera)
		}
		resultados[primera.Resultado] = true
	}
	for _, r := range []string{Pasa, Rechaza, NoAplica, Inactiva, Error} {
		if !resultados[r] {
			t.Errorf("los casos cubren el resultado %q", r)
		}
	}
}

// Decidir no llama a un modelo: el paquete solo usa la biblioteca estándar y yaml, sin red ni otros paquetes de qdd
// (donde viven los clientes de modelos).
// contrato: puertas#13be0fdb
func TestPuertas_SinModelosNiRed(t *testing.T) {
	archivos, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	permitidos := map[string]bool{"gopkg.in/yaml.v3": true}
	for _, archivo := range archivos {
		if strings.HasSuffix(archivo, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), archivo, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			ruta, _ := strconv.Unquote(imp.Path.Value)
			deTerceros := strings.Contains(strings.Split(ruta, "/")[0], ".")
			if ruta == "net" || strings.HasPrefix(ruta, "net/") || (deTerceros && !permitidos[ruta]) {
				t.Errorf("%s importa %s: decidir no usa la red ni un modelo", archivo, ruta)
			}
		}
	}
}
