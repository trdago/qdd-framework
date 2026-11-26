package baul

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

// Rule: Prohibido usar 'else'
// Las pruebas del baúl no tocan el llavero ni el 1Password reales, ni el QDD_HOME ni el HOME de la persona, aunque una
// prueba nueva olvide llamar a entorno: TestMain fija todo eso antes de correr cualquiera. El Secret Service real queda
// fuera de alcance (DBUS_SESSION_BUS_ADDRESS a un socket que no existe), así que sin el llavero de prueba guardar falla
// en vez de escribir en el de la persona.

// homeReal es el HOME con que se lanzaron las pruebas, antes de que TestMain lo cambie.
var homeReal string

func TestMain(m *testing.M) {
	homeReal, _ = os.UserHomeDir()
	dir, err := os.MkdirTemp("", "qdd-baul-pruebas-")
	if err != nil {
		os.Exit(1)
	}
	os.Setenv("HOME", filepath.Join(dir, "home"))
	os.Setenv("QDD_HOME", filepath.Join(dir, "qdd"))
	os.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+filepath.Join(dir, "sin-dbus"))
	keyring.MockInit()
	binOp = filepath.Join(dir, "sin-op")
	codigo := m.Run()
	os.RemoveAll(dir)
	os.Exit(codigo)
}

// contrato: baul#df85a0cf
func TestPruebas_NoTocanElLlaveroNi1PasswordNiElQDDHomeReal(t *testing.T) {
	ruta, err := rutaEn("indice.json")
	if err != nil {
		t.Fatal(err)
	}
	if homeReal != "" && strings.HasPrefix(ruta, homeReal+string(filepath.Separator)) {
		t.Fatalf("sin entorno, el índice del baúl caería en la carpeta de la persona: %s", ruta)
	}
	if h, _ := os.UserHomeDir(); h == homeReal {
		t.Fatalf("el HOME de las pruebas es el de la persona: %s", h)
	}
	if _, err := exec.LookPath(binOp); err == nil {
		t.Fatalf("las pruebas no alcanzan un 1Password real: binOp = %s", binOp)
	}
	if runtime.GOOS == "linux" {
		if err := keyring.Set(servicioLlavero, "centinela-de-pruebas", "x"); err != nil {
			t.Fatalf("sin el llavero de prueba, guardar iría al Secret Service de la persona: %v", err)
		}
		keyring.Delete(servicioLlavero, "centinela-de-pruebas")
	}
}

// Todo el recorrido de un token (guardar, pedir permiso, permitir, usar, renombrar, editar y quitar) deja archivos en
// QDD_HOME, el proyecto y el HOME; ninguno contiene el valor, y el comando lo recibe en su entorno, nunca en sus
// argumentos.
// contrato: baul#d66e3930
func TestBaul_ElValorNuncaQuedaEnUnArchivoNiEnLosArgumentos(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("usa sh")
	}
	root := entorno(t)
	guardarCloudflare(t)
	var salida bytes.Buffer
	Usar(opcionesUsar(root, &salida, "true"))
	permitirTodo(t, root)
	argumentos := filepath.Join(t.TempDir(), "argumentos")
	salida.Reset()
	codigo, err := Usar(opcionesUsar(root, &salida, "sh", "-c", `printf '%s\n' "$0" "$@" > "$1"; printf 'token=%s\n' "$CLOUDFLARE_API_TOKEN"`, "sh", argumentos))
	if err != nil || codigo != 0 {
		t.Fatalf("con permiso corre: %d %v", codigo, err)
	}
	if b, _ := os.ReadFile(argumentos); len(b) == 0 || strings.Contains(string(b), secreto) {
		t.Fatal("el valor llega por el entorno, nunca por los argumentos del comando")
	}
	if strings.Contains(salida.String(), secreto) {
		t.Fatal("la salida del comando sale tapada")
	}
	if err := Renombrar("cf", "cf-prod", ahora); err != nil {
		t.Fatal(err)
	}
	descripcion := "DNS"
	if err := Editar("cf-prod", Cambios{Descripcion: &descripcion}, ahora); err != nil {
		t.Fatal(err)
	}
	revisados := sinElValor(t, secreto, os.Getenv("QDD_HOME"), root, os.Getenv("HOME"))
	if err := Quitar("cf-prod", ahora); err != nil {
		t.Fatal(err)
	}
	revisados += sinElValor(t, secreto, os.Getenv("QDD_HOME"), root, os.Getenv("HOME"))
	if revisados < 4 {
		t.Fatalf("la prueba necesita los archivos del baúl, la cola y el registro: revisó %d", revisados)
	}
}

// sinElValor recorre las carpetas y falla si algún archivo contiene valor; devuelve cuántos archivos revisó.
func sinElValor(t *testing.T, valor string, carpetas ...string) int {
	t.Helper()
	revisados := 0
	for _, c := range carpetas {
		filepath.WalkDir(c, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			b, _ := os.ReadFile(p)
			revisados++
			if bytes.Contains(b, []byte(valor)) {
				t.Errorf("un archivo de QDD guarda el valor: %s", p)
			}
			return nil
		})
	}
	return revisados
}
