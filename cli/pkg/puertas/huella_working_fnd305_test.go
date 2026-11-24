package puertas

import "testing"

// Rule: Prohibido usar 'else'
// FND-305: en un proyecto nuevo ningún commit pasaba. qdd anota el comando que corre en .qdd/working/comando y guarda su
// caché en .qdd/cache; 'qdd init' no los ignora en git, así que el propio hook cambiaba la huella que iba a comparar.

func TestHuella_LoQueQddEscribeAlCorrerNoLaCambia_FND305(t *testing.T) {
	dir := repoDePrueba(t)
	escribir(t, dir, "a.go", "package a\n")
	escribir(t, dir, ".qdd/working/comando", "audit")
	escribir(t, dir, ".qdd/working/flujos/x.json", "{}\n")
	h := mustHuella(t, dir)
	escribir(t, dir, ".qdd/working/comando", "hook")
	escribir(t, dir, ".qdd/cache/wisdom/manifest.json", "{}\n")
	if h != mustHuella(t, dir) {
		t.Fatal("correr qdd después del audit no lo deja viejo")
	}
	git(t, dir, "add", "-A")
	if h != mustHuella(t, dir) {
		t.Fatal("tampoco con esas carpetas en el índice")
	}
	escribir(t, dir, ".qdd/workingno.md", "x\n")
	if h == mustHuella(t, dir) {
		t.Fatal("otro archivo de .qdd sí cambia la huella")
	}
}
