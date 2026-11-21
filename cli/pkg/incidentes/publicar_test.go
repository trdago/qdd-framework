package incidentes

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// GH de verdad, con un gh falso en el PATH que anota sus argumentos y su entrada: la búsqueda que evita duplicar un
// issue encuentra lo que el issue creado lleva en su cuerpo.

const ghQueAnota = `#!/bin/sh
n=$(ls "$GH_FALSO_DIR" | grep -c '\.args$')
n=$((n + 1))
for a in "$@"; do printf '%s\n' "$a"; done > "$GH_FALSO_DIR/$n.args"
cat > "$GH_FALSO_DIR/$n.stdin"
case "$2" in
  list) printf '%s\n' "$GH_FALSO_EXISTE" ;;
  create) echo "https://github.com/o/r/issues/7" ;;
esac
`

type llamadaGH struct {
	args    []string
	entrada string
}

// ghFalsoEnPath deja un gh falso primero en el PATH y devuelve cómo leer sus llamadas en orden.
func ghFalsoEnPath(t *testing.T, existe string) func() []llamadaGH {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("el gh falso es un script de sh")
	}
	bin, registro := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(ghQueAnota), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GH_FALSO_DIR", registro)
	t.Setenv("GH_FALSO_EXISTE", existe)
	return func() []llamadaGH {
		var llamadas []llamadaGH
		for n := 1; ; n++ {
			args, err := os.ReadFile(filepath.Join(registro, strconv.Itoa(n)+".args"))
			if err != nil {
				return llamadas
			}
			entrada, _ := os.ReadFile(filepath.Join(registro, strconv.Itoa(n)+".stdin"))
			llamadas = append(llamadas, llamadaGH{args: strings.Split(strings.TrimSuffix(string(args), "\n"), "\n"), entrada: string(entrada)})
		}
	}
}

func argDespuesDe(args []string, bandera string) string {
	i := slices.Index(args, bandera)
	if i < 0 || i+1 >= len(args) {
		return ""
	}
	return args[i+1]
}

func TestGH_BuscaPorLaFirmaQueLlevaElCuerpoYCreaConEseCuerpo(t *testing.T) {
	llamadas := ghFalsoEnPath(t, "")
	dir := t.TempDir()
	ahora := time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)
	if _, err := Registrar(dir, Nuevo(TipoPanico, "qdd estado set", "boom", nil, "2.8.0"), ahora); err != nil {
		t.Fatal(err)
	}
	todos, _ := Leer(dir)
	i := todos[0]
	if r := Publicar(dir, GH{Repo: "o/r"}, ahora, 5); r.Creados != 1 || r.Err != nil {
		t.Fatalf("crea el issue: %+v", r)
	}
	l := llamadas()
	if len(l) != 2 || l[0].args[1] != "list" || l[1].args[1] != "create" {
		t.Fatalf("primero busca y después crea: %+v", l)
	}
	busqueda := argDespuesDe(l[0].args, "--search")
	if busqueda != "qdd-incidente firma="+i.Firma+" in:body" || argDespuesDe(l[0].args, "--state") != "all" || argDespuesDe(l[0].args, "-R") != "o/r" {
		t.Fatalf("busca en todos los issues, abiertos y cerrados, por la firma en el cuerpo: %q", l[0].args)
	}
	if l[1].entrada != Cuerpo(i) || argDespuesDe(l[1].args, "--body-file") != "-" || argDespuesDe(l[1].args, "--title") != Titulo(i) {
		t.Fatalf("crea con el cuerpo estándar por la entrada:\n%q\n%q", l[1].args, l[1].entrada)
	}
	// Lo que se buscará la próxima vez está en el cuerpo creado: así un segundo PC comenta en vez de duplicar.
	marca, _, _ := strings.Cut(busqueda, " in:body")
	if !strings.Contains(l[1].entrada, marca) {
		t.Fatalf("el cuerpo no lleva %q: la búsqueda nunca lo encontraría y se duplicarían los issues", marca)
	}
}

func TestGH_UnIssueQueYaLlevaLaFirmaSeComentaNoSeDuplica(t *testing.T) {
	llamadas := ghFalsoEnPath(t, "https://github.com/o/r/issues/3")
	dir := t.TempDir()
	if _, err := Registrar(dir, Nuevo(TipoPanico, "qdd estado set", "boom", nil, "2.8.0"), time.Now()); err != nil {
		t.Fatal(err)
	}
	if r := Publicar(dir, GH{Repo: "o/r"}, time.Now(), 5); r.Comentados != 1 || r.Creados != 0 || r.Err != nil {
		t.Fatalf("comenta el que ya existe: %+v", r)
	}
	l := llamadas()
	if len(l) != 2 || l[1].args[1] != "comment" || l[1].args[2] != "https://github.com/o/r/issues/3" {
		t.Fatalf("busca y comenta el issue encontrado, sin crear otro: %+v", l)
	}
}
