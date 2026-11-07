package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/gitenv"
)

// Rule: Prohibido usar 'else'
// RFC 0020: una suite que declara sus rutas no se repite si ya pasó con esos mismos archivos. Es lo que hace la caché de Go,
// para cualquier suite y cualquier lenguaje, y no depende de contra qué rama se compare. Lo recordado vive en la carpeta
// común de git: no se commitea y los worktrees lo comparten. Todo lo que no se puede asegurar, corre: sin rutas, una ruta
// sin archivos, fuera de git, en el CI o pidiéndolo con qdd audit --completo.

const (
	archivoSuites       = "qdd-suites.json"
	envPruebasCompletas = "QDD_PRUEBAS_COMPLETAS"
	// vigenciaSuite: una herramienta puede cambiar sin que cambie ningún archivo.
	vigenciaSuite = 24 * time.Hour
)

// PedirPruebasCompletas hace que este proceso corra todas las suites, ya hayan pasado o no.
func PedirPruebasCompletas() {
	os.Setenv(envPruebasCompletas, "1")
}

// RunAllCompleto es RunAll sin suites recordadas: un certificado sale siempre de una corrida completa.
func (e *EngineCoordinator) RunAllCompleto() []Violation {
	antes, estaba := os.LookupEnv(envPruebasCompletas)
	PedirPruebasCompletas()
	defer restaurarPruebasCompletas(antes, estaba)
	return e.RunAll()
}

func restaurarPruebasCompletas(antes string, estaba bool) {
	if estaba {
		os.Setenv(envPruebasCompletas, antes)
		return
	}
	os.Unsetenv(envPruebasCompletas)
}

// huellaDeSuite resume el comando y el contenido de sus rutas. Vacía significa que hay que correrla.
func huellaDeSuite(tc testCommand) string {
	if len(tc.Rutas) == 0 || os.Getenv("CI") != "" || os.Getenv(envPruebasCompletas) != "" {
		return ""
	}
	h := sha256.New()
	fmt.Fprintf(h, "%s\x00%s\x00%s\x00", tc.Label, strings.Join(tc.Rutas, "\x00"), runtime.GOOS)
	for _, ruta := range tc.Rutas {
		archivos := archivosDeRuta(tc.Dir, ruta)
		if len(archivos) == 0 {
			return ""
		}
		for _, archivo := range archivos {
			fmt.Fprintf(h, "%s\x00", archivo)
			h.Write(contenidoParaHuella(filepath.Join(tc.Dir, archivo)))
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

// archivosDeRuta da los archivos que git ve en la ruta, commiteados o no; los ignorados no cuentan.
func archivosDeRuta(dir, ruta string) []string {
	salida, err := gitDeSuites(dir, "ls-files", "-co", "--exclude-standard", "-z", "--", ruta)
	if err != nil {
		return nil
	}
	return strings.FieldsFunc(salida, func(r rune) bool { return r == 0 })
}

// contenidoParaHuella: un archivo borrado o ilegible también cambia la huella.
func contenidoParaHuella(ruta string) []byte {
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		return []byte("\x00sin contenido\x00")
	}
	return contenido
}

func gitDeSuites(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = gitenv.Limpio(os.Environ())
	salida, err := cmd.Output()
	return string(salida), err
}

func rutaDeSuitesRecordadas(dir string) (string, error) {
	comun, err := gitDeSuites(dir, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}
	comun = strings.TrimSpace(comun)
	if filepath.IsAbs(comun) {
		return filepath.Join(comun, archivoSuites), nil
	}
	return filepath.Join(dir, comun, archivoSuites), nil
}

// leerSuitesRecordadas: lo ilegible se trata como vacío, porque lo único que cuesta es volver a correr.
func leerSuitesRecordadas(ruta string, ahora time.Time) map[string]time.Time {
	vigentes := map[string]time.Time{}
	b, err := os.ReadFile(ruta)
	if err != nil {
		return vigentes
	}
	var recordadas map[string]time.Time
	if json.Unmarshal(b, &recordadas) != nil {
		return vigentes
	}
	for huella, en := range recordadas {
		if ahora.Sub(en) < vigenciaSuite {
			vigentes[huella] = en
		}
	}
	return vigentes
}

// yaPasoConEsaHuella dice cuándo pasó la suite con estos mismos archivos.
func yaPasoConEsaHuella(dir, huella string) (time.Time, bool) {
	if huella == "" {
		return time.Time{}, false
	}
	ruta, err := rutaDeSuitesRecordadas(dir)
	if err != nil {
		return time.Time{}, false
	}
	en, ok := leerSuitesRecordadas(ruta, time.Now())[huella]
	return en, ok
}

// recordarSuite anota una suite que pasó. Si no se puede escribir, la próxima vez corre de nuevo.
func recordarSuite(dir, huella string) {
	if huella == "" {
		return
	}
	ruta, err := rutaDeSuitesRecordadas(dir)
	if err != nil {
		return
	}
	ahora := time.Now()
	recordadas := leerSuitesRecordadas(ruta, ahora)
	recordadas[huella] = ahora
	b, err := json.Marshal(recordadas)
	if err != nil {
		return
	}
	temporal := fmt.Sprintf("%s.%d", ruta, os.Getpid())
	if os.WriteFile(temporal, b, 0o600) != nil {
		return
	}
	if os.Rename(temporal, ruta) != nil {
		os.Remove(temporal)
	}
}
