package puertas

import "testing"

// Rule: Prohibido usar 'else'
// FND-299 (notificador_de_ofertas, 2026-11-22): la puerta audit rechazó dos commits porque estado.yaml cambió después
// del qdd audit (8 min ahí). Las reglas piden a la IA mover el tablero mientras trabaja, y cada escritura del tablero ya
// se valida antes de guardarse. La huella deja fuera .qdd/project/estado.yaml (elegido por el modo goal, la opción
// recomendada); cualquier otro archivo la sigue cambiando.

func TestHuella_ElTableroNoLaCambiaYLoDemasSi_FND299(t *testing.T) {
	dir := repoDePrueba(t)
	escribir(t, dir, "a.go", "package a\n")
	escribir(t, dir, ".qdd/project/estado.yaml", "formato: 1\n")
	h := mustHuella(t, dir)
	escribir(t, dir, ".qdd/project/estado.yaml", "formato: 1\nactualizado: \"2026-11-22T23:02:33-03:00\"\n")
	if h != mustHuella(t, dir) {
		t.Fatal("mover el tablero después del audit no lo deja viejo")
	}
	git(t, dir, "add", ".qdd/project/estado.yaml")
	if h != mustHuella(t, dir) {
		t.Fatal("tampoco con el tablero en el índice")
	}
	escribir(t, dir, ".qdd/project/modules/x.yaml", "id: x\n")
	if h == mustHuella(t, dir) {
		t.Fatal("otro archivo de .qdd/ sí cambia la huella")
	}
}
