package herramientas

import (
	"path/filepath"
	"regexp"
	"strings"
)

// Rule: Prohibido usar 'else'

var noID = regexp.MustCompile(`[^a-z0-9]+`)

// MensajeParaConfigurar es lo que el panel le envía a la IA al agregar una herramienta (paso 2 de RFC 0011): leer su
// ayuda, proponer las acciones a la persona y guardarla con QDD. Vive en la CLI para que el formato y las reglas sean
// uno solo, el que valida Guardar.
func MensajeParaConfigurar(argv []string) string {
	comando := citar(argv)
	return `Configura ` + "`" + strings.Join(argv, " ") + "`" + ` como herramienta del panel de QDD (RFC 0011):
1. Lee su ayuda: qdd herramienta ayuda ` + comando + `
2. Propón sus acciones (cada una es un botón) con los campos que piden, y muéstramelas para que las acepte o pida cambios. Si solo funciona en modo interactivo (espera lo que escribo en su entrada), dilo y no la guardes.
3. Escribe la definición en un archivo temporal y guárdala con: qdd herramienta guardar <archivo>. QDD la valida y corre su prueba de humo; si falla, corrígela con lo que dice.

Formato:
id: ` + idSugerido(argv) + `
nombre: <lo que dice la pestaña>
descripcion: <una línea>
comando: [` + strings.Join(argv, ", ") + `]
entorno: [NOMBRE_DE_VARIABLE]   # solo si necesita alguna
acciones:
  - id: <id>
    nombre: <lo que dice el botón>
    args: [<subcomando>, "{{campo}}", [--opcion, "{{otro}}"]]
    campos:
      - {id: campo, tipo: texto, requerido: true}       # texto, numero, opcion (con opciones: […]) o si-no
    modo: una-vez                                        # o en-vivo: su salida se ve mientras corre, hasta ■ Detener
    confirmar: true                                      # en lo que borra o cambia datos
prueba: {args: [--help]}                                 # debe salir con 0

Reglas: cada {{campo}} es un argumento entero y nunca pasa por una shell. Un grupo [--x, "{{x}}"] va solo si su campo tiene valor; un si-no va en un grupo y decide si el grupo va. Los secretos van solo por su nombre en entorno, nunca su valor, ni aquí ni en la conversación. No la corras tú: la persona la corre desde su pestaña.`
}

func idSugerido(argv []string) string {
	ultimo := filepath.Base(argv[len(argv)-1])
	ultimo = strings.TrimSuffix(ultimo, filepath.Ext(ultimo))
	id := strings.Trim(noID.ReplaceAllString(strings.ToLower(ultimo), "-"), "-")
	if id == "" {
		return "herramienta"
	}
	return id
}

// citar escribe argv para una shell: cada argumento con espacios o símbolos va entre comillas simples.
func citar(argv []string) string {
	var out []string
	for _, a := range argv {
		if regexp.MustCompile(`^[A-Za-z0-9_./=:@%+-]+$`).MatchString(a) {
			out = append(out, a)
			continue
		}
		out = append(out, "'"+strings.ReplaceAll(a, "'", `'\''`)+"'")
	}
	return strings.Join(out, " ")
}
