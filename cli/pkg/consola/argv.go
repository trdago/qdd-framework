package consola

import (
	"regexp"
	"strings"
)

// Rule: Prohibido usar 'else'
// FND-212 (MEJ-043): «qdd consola pedir -- git -c credential.helper="!gh auth git-credential" push» llega como
// argumentos separados (la shell de quien lo pide ya quitó las comillas). Unidos con espacios, la shell del pedido los
// volvía a partir y git recibía «auth» como subcomando. Cada argumento con espacios o caracteres de la shell se cita
// para la shell que lo va a correr (sh -c o cmd /C): el texto que ve la persona es exactamente el que corre.

// palabraSegura es un argumento que ninguna shell parte ni interpreta: va sin comillas.
var palabraSegura = regexp.MustCompile(`^[A-Za-z0-9_@%+=:,./-]+$`)

// ComandoDeArgs arma el texto de shell de un pedido con los argumentos de la CLI. Un solo argumento ya es un texto de
// shell (como el comando de la tool MCP) y va tal cual.
func ComandoDeArgs(args []string, goos string) string {
	if len(args) == 1 {
		return args[0]
	}
	citar := citarPOSIX
	if goos == "windows" {
		citar = citarCmd
	}
	partes := make([]string, len(args))
	for i, a := range args {
		partes[i] = citar(a)
	}
	return strings.Join(partes, " ")
}

// citarPOSIX: entre comillas simples nada se interpreta; una comilla simple se cierra, se escapa y se vuelve a abrir.
func citarPOSIX(a string) string {
	if palabraSegura.MatchString(a) {
		return a
	}
	return "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
}

// citarCmd: en cmd, las comillas dobles juntan el argumento y el programa lee \" como una comilla (reglas de
// CommandLineToArgvW).
func citarCmd(a string) string {
	if palabraSegura.MatchString(a) {
		return a
	}
	return `"` + strings.ReplaceAll(a, `"`, `\"`) + `"`
}
