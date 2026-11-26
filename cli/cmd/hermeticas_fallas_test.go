package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'

// FND-322: las herméticas en rojo mostraban solo las últimas 40 líneas del registro: la prueba que falló, en un paquete
// anterior, no salía en el log del CI.
func TestHermeticas_UnaFallaDeUnPaqueteAnteriorSeMuestra_FND322(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("las herméticas corren con sh")
	}
	registro := filepath.Join(t.TempDir(), "go-test.log")
	lineas := "--- FAIL: TestLaQueFalla (0.01s)\n    cosa_test.go:12: esperaba 3\nFAIL\nFAIL\tejemplo/paquete\t0.1s\n" +
		strings.Repeat("ok  \tejemplo/otro\t0.1s\n", 60) + "FAIL\n"
	if err := os.WriteFile(registro, []byte(lineas), 0o600); err != nil {
		t.Fatal(err)
	}
	salida, err := exec.Command("sh", "../../e2e/hermeticas/fallas.sh", registro).CombinedOutput()
	if err != nil {
		t.Fatalf("%v: %s", err, salida)
	}
	for _, parte := range []string{"--- FAIL: TestLaQueFalla", "cosa_test.go:12: esperaba 3", "ejemplo/paquete"} {
		if !strings.Contains(string(salida), parte) {
			t.Errorf("no dice «%s»:\n%s", parte, salida)
		}
	}
}

func TestHermeticas_CorrerUsaFallas_FND322(t *testing.T) {
	if !strings.Contains(leerDoc(t, "../../e2e/hermeticas/correr.sh"), `/e2e/hermeticas/fallas.sh" "$caja/go-test.log"`) {
		t.Error("correr.sh no muestra las fallas con fallas.sh")
	}
}
