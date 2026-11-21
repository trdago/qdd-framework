package puertas

import (
	"os"
	"path/filepath"
	"strings"
)

// Los hooks de git llevan las puertas a cualquier IA (y a la persona, que puede saltarlas con --no-verify en su
// terminal): pre-commit revisa la puerta audit y commit-msg la finding. Conviven con lo que ya hay: un bloque marcado
// tras el shebang, que se agrega o se reemplaza; nunca se toca lo ajeno.
const (
	marcaInicio = "# >>> qdd puertas >>> (lo instala qdd puertas instalar; qdd doctor --fix lo repara)"
	marcaFin    = "# <<< qdd puertas <<<"
)

// Estado de un hook.
const (
	HookAlDia = "al día"
	HookFalta = "falta"
	HookViejo = "viejo"
)

// Hooks que instalan las puertas, con lo que corre cada uno. Solo el 2 de qdd (rechazo) corta el commit: sin qdd, con
// uno que no tiene 'puertas hook' o si falla, pasa. Funciona también con sh -e (husky).
var lineasHook = map[string]string{
	"pre-commit": `if command -v qdd >/dev/null 2>&1; then qdd puertas hook pre-commit || [ $? -ne 2 ] || exit 1; fi`,
	"commit-msg": `if command -v qdd >/dev/null 2>&1; then qdd puertas hook commit-msg "$1" || [ $? -ne 2 ] || exit 1; fi`,
}

var ordenHooks = []string{"pre-commit", "commit-msg"}

// EstadoHook es un hook de las puertas en este repositorio.
type EstadoHook struct {
	Hook   string
	Ruta   string
	Estado string
}

func bloque(hook string) string {
	return marcaInicio + "\n" + lineasHook[hook] + "\n" + marcaFin + "\n"
}

// carpetaHooks: con husky (core.hooksPath=.husky/_, también absoluto) los hooks van en el .husky/ de este worktree, que
// se versiona; con otro core.hooksPath, ahí; si no, en la carpeta de hooks de git (la común a todos los worktrees).
func carpetaHooks(dir string) (string, bool, error) {
	raiz, err := Raiz(dir)
	if err != nil {
		return "", false, err
	}
	propia, _ := gitSalida(raiz, nil, "config", "core.hooksPath")
	limpia := filepath.ToSlash(strings.TrimSuffix(propia, "/"))
	if limpia == ".husky/_" || strings.HasSuffix(limpia, "/.husky/_") {
		return filepath.Join(raiz, ".husky"), true, nil
	}
	if propia != "" {
		return absEn(raiz, propia), false, nil
	}
	comun, err := gitSalida(raiz, nil, "rev-parse", "--path-format=absolute", "--git-common-dir")
	return filepath.Join(comun, "hooks"), false, err
}

func absEn(raiz, ruta string) string {
	if filepath.IsAbs(ruta) {
		return ruta
	}
	return filepath.Join(raiz, ruta)
}

// EsRaiz dice si dir es la raíz de su repositorio: los hooks son de todo el repositorio, así que solo se instalan desde
// ahí (un proyecto en una subcarpeta, o una prueba, no los toca).
func EsRaiz(dir string) bool {
	raiz, err := Raiz(dir)
	if err != nil {
		return false
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	real, err := filepath.EvalSymlinks(abs)
	return err == nil && real == raiz
}

// EstadoHooks dice si cada hook de las puertas falta, está viejo o al día.
func EstadoHooks(raiz string) []EstadoHook {
	dir, _, err := carpetaHooks(raiz)
	var out []EstadoHook
	for _, hook := range ordenHooks {
		e := EstadoHook{Hook: hook, Ruta: filepath.Join(dir, hook), Estado: HookFalta}
		datos, errLeer := os.ReadFile(e.Ruta)
		if err == nil && errLeer == nil {
			e.Estado = estadoDe(string(datos), hook)
		}
		out = append(out, e)
	}
	return out
}

func estadoDe(texto, hook string) string {
	if strings.Contains(texto, bloque(hook)) {
		return HookAlDia
	}
	if strings.Contains(texto, marcaInicio) {
		return HookViejo
	}
	return HookFalta
}

// InstalarHooks deja al día los hooks de las puertas y devuelve los que cambió.
func InstalarHooks(raiz string) ([]string, error) {
	dir, husky, err := carpetaHooks(raiz)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	var cambios []string
	for _, hook := range ordenHooks {
		cambio, err := instalarHook(filepath.Join(dir, hook), hook, husky)
		if err != nil {
			return cambios, err
		}
		if cambio {
			cambios = append(cambios, filepath.Join(dir, hook))
		}
	}
	return cambios, nil
}

func instalarHook(ruta, hook string, husky bool) (bool, error) {
	datos, err := os.ReadFile(ruta)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	actual := string(datos)
	nuevo := conBloque(actual, hook, husky)
	if nuevo == actual {
		return false, nil
	}
	if err := os.WriteFile(ruta, []byte(nuevo), 0o755); err != nil {
		return false, err
	}
	return true, os.Chmod(ruta, 0o755)
}

// conBloque pone el bloque al día: reemplaza el que había o lo agrega tras el shebang. Un archivo nuevo lleva
// #!/bin/sh, salvo en husky, que corre sus hooks con sh.
func conBloque(texto, hook string, husky bool) string {
	if estadoDe(texto, hook) == HookAlDia {
		return texto
	}
	inicio, fin := strings.Index(texto, marcaInicio), strings.Index(texto, marcaFin)
	if inicio >= 0 && fin > inicio {
		resto := strings.TrimPrefix(texto[fin+len(marcaFin):], "\n")
		return texto[:inicio] + bloque(hook) + resto
	}
	if texto == "" && !husky {
		return "#!/bin/sh\n" + bloque(hook)
	}
	if !strings.HasPrefix(texto, "#!") {
		return bloque(hook) + texto
	}
	shebang, resto, _ := strings.Cut(texto, "\n")
	return shebang + "\n" + bloque(hook) + resto
}

// MensajeSinComentarios es el mensaje de commit que lee commit-msg, sin las líneas # que git agrega en el editor.
func MensajeSinComentarios(texto string) string {
	var lineas []string
	for _, l := range strings.Split(texto, "\n") {
		if !strings.HasPrefix(l, "#") {
			lineas = append(lineas, l)
		}
	}
	return strings.TrimSpace(strings.Join(lineas, "\n"))
}
