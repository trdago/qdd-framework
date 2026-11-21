package invitaciones

import (
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// RFC 0025, fase 3b: este PC anota cada enlace que publicó (sin token) para reusarlo, encontrarlo al revocar la
// invitación y borrarlo de Cloudflare.

func TestPublicados_SeAnotanSeReusanYSeQuitan(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	ahora := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	solo := Publicado{ID: "a", Flujo: "venta", URL: "https://qdd-flujos.ana.workers.dev/f/a", Cuenta: "c", Espacio: "e", Creado: ahora}
	conInvitacion := Publicado{ID: "b", Flujo: "venta", Invitacion: "inv-1", URL: "https://qdd-flujos.ana.workers.dev/f/b", Cuenta: "c", Espacio: "e", Creado: ahora}
	for _, p := range []Publicado{solo, conInvitacion} {
		if err := AnotarPublicado(root, p); err != nil {
			t.Fatal(err)
		}
	}
	l, _ := Leer(root)
	if p, ok := l.SoloLectura("venta"); !ok || p.ID != "a" {
		t.Fatalf("el de solo lectura de un flujo se reusa: %+v %v", p, ok)
	}
	if p, ok := l.DeInvitacion("inv-1"); !ok || p.ID != "b" {
		t.Fatalf("el de una invitación se encuentra al revocarla: %+v %v", p, ok)
	}
	if err := MarcarQuitado(root, "b"); err != nil {
		t.Fatal(err)
	}
	l, _ = Leer(root)
	if _, ok := l.DeInvitacion("inv-1"); ok {
		t.Fatal("uno quitado ya no se ofrece")
	}
	if err := MarcarQuitado(root, "zz"); err == nil {
		t.Fatal("uno que no está se dice")
	}
}
