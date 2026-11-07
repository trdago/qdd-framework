package cmd

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// FND-173: Go guarda el resultado de una prueba y solo mira los archivos de su módulo. El módulo es cli/, así que una
// prueba que lee docs/ o .github/ seguía pasando desde la caché con esos archivos rotos. Esas pruebas corren en el gate con
// -count=1; esta prueba impide que aparezca otro paquete así sin que nadie lo note.

var (
	subeConTexto = regexp.MustCompile(`"((?:\.\./)+)`)
	subeConJoin  = regexp.MustCompile(`Join\(((?:"\.\.",?\s*)+)`)
)

// nivelesQueSube dice cuánto sube el texto más alto del archivo. Solo cuenta lo que empieza subiendo: "a/../../x" es un dato
// de prueba, no una lectura.
func nivelesQueSube(fuente string) int {
	maximo := 0
	for _, m := range subeConTexto.FindAllStringSubmatch(fuente, -1) {
		maximo = max(maximo, strings.Count(m[1], "../"))
	}
	for _, m := range subeConJoin.FindAllStringSubmatch(fuente, -1) {
		maximo = max(maximo, strings.Count(m[1], `".."`))
	}
	return maximo
}

func TestNivelesQueSube_SoloCuentaLoQueEmpiezaSubiendo_FND173(t *testing.T) {
	casos := map[string]int{
		`os.ReadFile("../../docs/x.md")`:                  2,
		`filepath.Join("..", "..", "..", ".qdd")`:         3,
		`filepath.Join("..", "go.mod")`:                   1,
		`filepath.Join("..", "..", "..")`:                 3,
		`[]string{"a/../../x", "/etc/passwd"}`:            0,
		`os.ReadFile("adentro.txt")`:                      0,
		"filepath.Join(\"..\",\n\t\"..\", \"README.md\")": 2,
	}
	for fuente, niveles := range casos {
		if n := nivelesQueSube(fuente); n != niveles {
			t.Errorf("%s: %d, debía ser %d", fuente, n, niveles)
		}
	}
}

func TestGate_LasPruebasQueLeenFueraDelModuloCorrenSinCache_FND173(t *testing.T) {
	config, err := os.ReadFile(filepath.Join("..", "..", ".qdd", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	lineaSinCache := ""
	for _, linea := range strings.Split(string(config), "\n") {
		if strings.Contains(linea, "go test") && strings.Contains(linea, "-count=1") {
			lineaSinCache = linea
		}
	}
	raiz := ".."
	err = filepath.WalkDir(raiz, func(ruta string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == "core_assets" || d.Name() == "node_modules") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(ruta, "_test.go") {
			return nil
		}
		fuente, err := os.ReadFile(ruta)
		if err != nil {
			return err
		}
		paquete, err := filepath.Rel(raiz, filepath.Dir(ruta))
		if err != nil {
			return err
		}
		paquete = filepath.ToSlash(paquete)
		if nivelesQueSube(string(fuente)) <= strings.Count(paquete, "/")+1 {
			return nil
		}
		if !strings.Contains(lineaSinCache, "./"+paquete+"/") {
			t.Errorf("%s lee fuera de cli/: agrega ./%s/ a la línea de go test con -count=1 de .qdd/config.yaml", ruta, paquete)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// RFC 0020: un certificado no puede salir de suites recordadas.
func TestCertificar_SiempreEsUnaCorridaCompleta_RFC0020(t *testing.T) {
	for archivo, desde := range map[string]string{"certify.go": "func runCertify", "mcp_tools.go": "func registerCertifyTool"} {
		fuente, err := os.ReadFile(archivo)
		if err != nil {
			t.Fatal(err)
		}
		_, cuerpo, ok := strings.Cut(string(fuente), desde)
		if !ok {
			t.Fatalf("%s no tiene %s", archivo, desde)
		}
		fin := strings.Index(cuerpo, "\nfunc ")
		if fin >= 0 {
			cuerpo = cuerpo[:fin]
		}
		if !strings.Contains(cuerpo, "RunAllCompleto()") || strings.Contains(cuerpo, ".RunAll()") {
			t.Errorf("%s: %s tiene que usar RunAllCompleto", archivo, desde)
		}
	}
}

func TestAudit_TieneLaBanderaCompleto_RFC0020(t *testing.T) {
	if auditCmd.Flags().Lookup("completo") == nil {
		t.Fatal("qdd audit --completo es lo que permite repetirlas todas")
	}
}
