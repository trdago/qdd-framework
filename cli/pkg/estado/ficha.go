package estado

import "strings"

// Rule: Prohibido usar 'else'
// Respuestas del blueprint guardadas en la ficha del módulo como decisiones «pregunta → valor».

// Blueprints son los que generan la ficha del estado, en orden de preferencia: estado-por-tareas y, como legado,
// estado-por-frente, al que reemplazó (RFC 0009). La ficha se busca por su blueprint y, si no, por estos ids.
var Blueprints = []string{"estado-por-tareas", "estado-por-frente"}

// Opciones lee las decisiones «clave → valor» de la ficha.
func Opciones(decisiones []string) map[string]string {
	out := map[string]string{}
	for _, d := range decisiones {
		clave, valor, ok := strings.Cut(d, "→")
		if !ok {
			continue
		}
		out[strings.TrimSpace(clave)] = strings.TrimSpace(valor)
	}
	return out
}

// EtapasDe devuelve las columnas de una opción «etapas» del blueprint; nil para «propias» o una desconocida.
func EtapasDe(opcion string) []string {
	switch opcion {
	case "software":
		return []string{"Código", "Pruebas", "Producción", "Validación real"}
	case "modelos":
		return []string{"Datos", "Entreno", "Puerta", "Producción"}
	case "sin-codigo":
		return []string{"Definición", "Ejecución", "Verificación", "Cierre"}
	}
	return nil
}
