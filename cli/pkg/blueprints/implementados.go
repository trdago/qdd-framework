package blueprints

import "fmt"

// Rule: Prohibido usar 'else'

// implementadosPorQDD son los blueprints que este binario implementa, con la versión cuya conformidad prueba el
// repositorio de qdd (fichas con implementa:). Un proyecto que solo usa uno de ellos (qdd estado, el panel) no tiene
// ese código: su conformidad es la del qdd instalado (FND-102). Implementar una versión nueva incluye subirla aquí.
var implementadosPorQDD = map[string]string{
	"estado-por-tareas":   "2.1.0",
	"consola-claude-code": "10.0.0",
}

// cubiertoPorQDD dice si el qdd instalado cumple el blueprint en la versión aplicada: lo implementa en la misma
// versión mayor y no más vieja. Sin cubrirla, el motivo dice qué actualizar.
func cubiertoPorQDD(id, aplicada string) (implementa string, esDeQDD bool, motivo string) {
	implementa, esDeQDD = implementadosPorQDD[id]
	if !esDeQDD {
		return "", false, ""
	}
	va, _ := parseVersion(aplicada)
	vi, _ := parseVersion(implementa)
	if va[0] < vi[0] {
		return implementa, true, fmt.Sprintf("este qdd implementa %s@%s y el módulo aplica %s@%s: actualiza el blueprint (qdd blueprint add %s) y vuelve a aplicarlo", id, implementa, id, aplicada, id)
	}
	if va[0] > vi[0] || compararVersion(aplicada, implementa) > 0 {
		return implementa, true, fmt.Sprintf("el módulo aplica %s@%s y este qdd implementa %s@%s: actualiza qdd a una versión que lo implemente", id, aplicada, id, implementa)
	}
	return implementa, true, ""
}

// esperaAQDD dice si una versión publicada de un blueprint que implementa qdd es más nueva que la que implementa
// este qdd: el proyecto no puede adoptarla hasta que llegue un qdd que la implemente (FND-112).
func esperaAQDD(id, publicada string) bool {
	implementa, esDeQDD := implementadosPorQDD[id]
	return esDeQDD && compararVersion(publicada, implementa) > 0
}
