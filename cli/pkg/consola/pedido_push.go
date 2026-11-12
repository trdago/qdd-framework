package consola

import (
	"crypto/sha256"
	"encoding/hex"
	"os/exec"
	"regexp"
	"strings"
)

// Rule: Prohibido usar 'else'
// MEJ-053: un git push vale por lo que hay para subir, no por su texto. El seguro de FND-154 recuerda junto al push la
// huella de las ramas y tags locales; si cambió (hay commits o tags nuevos), el mismo comando ya no es repetido.

var rePush = regexp.MustCompile(`^(?:\w+=\S*\s+)*git\s+push\b`)

// huellaPush es la huella de las ramas y tags de la carpeta si el comando es un git push solo; "" en cualquier otro caso
// (un comando encadenado, otro comando o una carpeta sin git): entonces el seguro sigue como antes.
func huellaPush(comando, carpeta string) string {
	comando = strings.TrimSpace(comando)
	if strings.ContainsAny(comando, ";&|\n") || !rePush.MatchString(comando) {
		return ""
	}
	out, err := exec.Command("git", "-C", carpeta, "for-each-ref", "--format=%(objectname) %(refname)", "refs/heads", "refs/tags").Output()
	if err != nil {
		return ""
	}
	suma := sha256.Sum256(out)
	return hex.EncodeToString(suma[:8])
}
