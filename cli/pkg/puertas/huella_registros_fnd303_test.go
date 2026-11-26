package puertas

import "testing"

// Rule: Prohibido usar 'else'
// FND-303 (proyecto-b, 2026-11-23): con FND-299 la puerta siguió rechazando el commit: un qdd_estado set por
// MCP escribió también el registro de herramientas del MCP (mcp_eventos.json; un MCP anterior a FND-059 escribe
// cognitive_history.json). La persona eligió sacar los dos de la huella, como el tablero: qdd los escribe solo al usar
// sus herramientas y no son código.

func TestHuella_LosRegistrosDelMCPNoLaCambian_FND303(t *testing.T) {
	dir := repoDePrueba(t)
	escribir(t, dir, "a.go", "package a\n")
	escribir(t, dir, ".qdd/project/metrics/mcp_eventos.json", "[]\n")
	escribir(t, dir, ".qdd/project/metrics/cognitive_history.json", "[]\n")
	h := mustHuella(t, dir)
	escribir(t, dir, ".qdd/project/metrics/mcp_eventos.json", `[{"tool":"qdd_estado"}]`+"\n")
	escribir(t, dir, ".qdd/project/metrics/cognitive_history.json", `[{"tool":"qdd_estado"}]`+"\n")
	if h != mustHuella(t, dir) {
		t.Fatal("una herramienta del MCP después del audit no lo deja viejo")
	}
	git(t, dir, "add", ".qdd/project/metrics")
	if h != mustHuella(t, dir) {
		t.Fatal("tampoco con los registros en el índice")
	}
	escribir(t, dir, ".qdd/project/metrics/otro.json", "{}\n")
	if h == mustHuella(t, dir) {
		t.Fatal("otro archivo de metrics/ sí cambia la huella")
	}
}
