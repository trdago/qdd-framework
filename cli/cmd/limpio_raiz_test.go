package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'

// capaDePrueba es un tar.gz con un archivo, como una capa OCI.
func capaDePrueba(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	contenido := []byte("Debian de prueba\n")
	if err := tw.WriteHeader(&tar.Header{Name: "etc/issue", Mode: 0o644, Size: int64(len(contenido))}); err != nil {
		t.Fatal(err)
	}
	tw.Write(contenido)
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func sha(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// registroFalso deja en el PATH un curl que responde como el registro de Docker: el token, el manifiesto y la capa
// que se le dan, desde archivos.
func registroFalso(t *testing.T, manifiesto, capa []byte) {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "manifiesto"), manifiesto, 0o644)
	os.WriteFile(filepath.Join(dir, "capa"), capa, 0o644)
	curl := `#!/bin/sh
salida=""
url=""
while [ $# -gt 0 ]; do
  case "$1" in
    -o) salida="$2"; shift 2 ;;
    -*) shift ;;
    *) url="$1"; shift ;;
  esac
done
case "$url" in
  *auth.docker.io*) echo '{"token":"de-prueba"}' ;;
  */manifests/*) cat "` + dir + `/manifiesto" ;;
  */blobs/*) cp "` + dir + `/capa" "$salida" ;;
  *) exit 22 ;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "curl"), []byte(curl), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// raizConDigest copia raiz.sh fijando la imagen en digest (el de un manifiesto de prueba) y la corre con su caché en
// una carpeta temporal. Devuelve la salida, la carpeta donde quedaría la raíz y el error.
func raizConDigest(t *testing.T, digest string) (string, string, error) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "e2e", "limpio", "raiz.sh"))
	if err != nil {
		t.Fatal(err)
	}
	script := string(data)
	original := `DIGEST="sha256:69c6e263db01e9946c7743bbbe9f7b1ba0df384d27ec09e08e5832d6acd3daf3"`
	if digest != "" {
		if !strings.Contains(script, original) {
			t.Fatal("raiz.sh ya no fija el digest como esta prueba espera")
		}
		script = strings.Replace(script, original, `DIGEST="sha256:`+digest+`"`, 1)
	}
	ruta := filepath.Join(t.TempDir(), "raiz.sh")
	os.WriteFile(ruta, []byte(script), 0o755)
	cache := t.TempDir()
	c := exec.Command("bash", ruta)
	c.Env = append(os.Environ(), "QDD_LIMPIO_CACHE="+cache)
	out, err := c.CombinedOutput()
	d := digest
	if d == "" {
		d = strings.TrimPrefix(strings.Trim(strings.TrimPrefix(original, "DIGEST="), `"`), "sha256:")
	}
	return string(out), filepath.Join(cache, d), err
}

func noQuedoRaiz(t *testing.T, raiz string) {
	t.Helper()
	if _, err := os.Stat(raiz); err == nil {
		t.Errorf("la preparación siguió y dejó la raíz en %s", raiz)
	}
}

// Un manifiesto o una capa que no coincide con su sha256 corta la preparación: sale con error y no deja raíz.
// contrato: ambiente-limpio#2774f199
func TestLimpioRaiz_UnShaQueNoCoincideCortaLaPreparacion(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("raiz.sh prepara una raíz de amd64")
	}
	for _, p := range []string{"bash", "jq", "sha256sum", "tar"} {
		if _, err := exec.LookPath(p); err != nil {
			t.Skipf("sin %s", p)
		}
	}
	capa := capaDePrueba(t)
	manifiestoCon := func(digestCapa string) []byte {
		return []byte(`{"layers":[{"digest":"sha256:` + digestCapa + `"}]}`)
	}

	t.Run("manifiesto distinto del digest fijado", func(t *testing.T) {
		registroFalso(t, manifiestoCon(sha(capa)), capa)
		out, raiz, err := raizConDigest(t, "")
		if err == nil || !strings.Contains(out, "el manifiesto no coincide") {
			t.Fatalf("raiz.sh con un manifiesto ajeno: %v %q; want error del manifiesto", err, out)
		}
		noQuedoRaiz(t, raiz)
	})

	t.Run("capa distinta de su digest", func(t *testing.T) {
		m := manifiestoCon(sha([]byte("otra capa")))
		registroFalso(t, m, capa)
		out, raiz, err := raizConDigest(t, sha(m))
		if err == nil || !strings.Contains(out, "no coincide con su sha256") {
			t.Fatalf("raiz.sh con una capa alterada: %v %q; want error de la capa", err, out)
		}
		noQuedoRaiz(t, raiz)
	})

	t.Run("todo coincide", func(t *testing.T) {
		m := manifiestoCon(sha(capa))
		registroFalso(t, m, capa)
		out, raiz, err := raizConDigest(t, sha(m))
		if err != nil {
			t.Fatalf("raiz.sh con manifiesto y capa íntegros: %v %q", err, out)
		}
		if _, err := os.Stat(filepath.Join(raiz, ".lista")); err != nil {
			t.Errorf("con todo íntegro la raíz queda lista: %v", err)
		}
		if data, _ := os.ReadFile(filepath.Join(raiz, "etc", "issue")); string(data) != "Debian de prueba\n" {
			t.Errorf("la raíz no trae la capa: %q", data)
		}
	})
}
