package consola

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/qdd-framework/qdd/pkg/blueprints"
)

// Rule: Prohibido usar 'else'
// Pedido del usuario (2026-11-01): en el panel, los blueprints a un clic. Actualizar el catálogo, ver los instalados y
// los disponibles, actualizar uno cuya receta es la misma e instalar uno son deterministas (sin tokens); solo aplicar
// (construir o adaptar el módulo) es de la IA.

var inventario = blueprints.Inventario{Entradas: []blueprints.EntradaInventario{
	{ID: "auth", Nombre: "Usuarios y acceso", Origen: "https://x/bp", Instalada: "1.0.0", Publicada: "1.1.0", Ref: "v3.0.0", Estado: blueprints.CatConCambios, Cambio: "minor", Razones: []string{"se agregó una invariante"}},
	{ID: "estado", Nombre: "Estado por tareas", Origen: "https://x/bp", Instalada: "2.0.0", Publicada: "2.0.0", Ref: "v3.0.0", Estado: blueprints.CatSoloLock},
	{ID: "consola", Nombre: "Consola", Origen: "https://x/bp", Instalada: "1.0.0", Publicada: "1.0.0", Ref: "v3.0.0", Estado: blueprints.CatAlDia},
	{ID: "pagos", Nombre: "Pagos", Origen: "https://x/bp", Publicada: "1.0.0", Ref: "v3.0.0", Estado: blueprints.CatDisponible},
}}

func accionDe(t *testing.T, acciones []Accion, id string) Accion {
	t.Helper()
	a, err := BuscarAccion(acciones, id)
	if err != nil {
		t.Fatalf("%v: %+v", err, acciones)
	}
	return a
}

func TestBlueprints_ActualizarElCatalogoNoEscribeEnElProyecto(t *testing.T) {
	a := accionDe(t, AccionesDeInventario(inventario), "blueprint:sync")
	if a.Tipo != TipoComando || a.Confirmar || strings.Join(a.Args, " ") != "blueprint sync" {
		t.Fatalf("trae el catálogo sin pedir «s  Sí»: solo escribe la copia central: %+v", a)
	}
}

func TestBlueprints_ActualizarEInstalarSonDeterministasYPidenConfirmar(t *testing.T) {
	acciones := AccionesDeInventario(inventario)
	for id, args := range map[string]string{
		"blueprint:actualizar:estado": "blueprint add estado",
		"blueprint:actualizar:auth":   "blueprint add auth",
		"blueprint:instalar:pagos":    "blueprint add https://x/bp pagos",
	} {
		a := accionDe(t, acciones, id)
		if a.Tipo != TipoComando || !a.Confirmar || strings.Join(a.Args, " ") != args {
			t.Fatalf("%s escribe el lockfile: comando de qdd con «s  Sí»: %+v", id, a)
		}
	}
	if _, err := BuscarAccion(acciones, "blueprint:actualizar:consola"); err == nil {
		t.Fatal("uno al día no se actualiza")
	}
}

func TestBlueprints_AplicarConLaIAEsUnMensajeCorto(t *testing.T) {
	acciones := AccionesDeInventario(inventario)
	a := accionDe(t, acciones, "blueprint:aplicar:auth")
	if a.Tipo != TipoAtender || utf8.RuneCountInString(a.Detalle) > MaxMensajeAtender {
		t.Fatalf("un mensaje para la IA de %d caracteres como mucho: %+v", MaxMensajeAtender, a)
	}
	for _, quiere := range []string{"«Usuarios y acceso»", "qdd blueprint add auth", "qdd_blueprint_apply", "pregúntame"} {
		if !strings.Contains(a.Detalle, quiere) {
			t.Fatalf("el mensaje dice %q: %s", quiere, a.Detalle)
		}
	}
	if _, err := BuscarAccion(acciones, "blueprint:aplicar:pagos"); err == nil {
		t.Fatal("uno sin instalar se instala primero (sin tokens)")
	}
	if d := accionDe(t, acciones, "blueprint:aplicar:consola").Detalle; strings.Contains(d, "blueprint add") {
		t.Fatalf("uno al día solo se aplica: %s", d)
	}
}

func TestBlueprints_LosQueNoCambianLaRecetaSeActualizanJuntos(t *testing.T) {
	inv := inventario
	inv.Entradas = append(append([]blueprints.EntradaInventario{}, inv.Entradas...), blueprints.EntradaInventario{ID: "roles", Origen: "https://x/bp", Instalada: "1.0.0", Publicada: "1.0.1", Ref: "v3.0.0", Estado: blueprints.CatSoloLock})
	a := accionDe(t, AccionesDeInventario(inv), "blueprint:actualizar-sin-ia")
	if a.Tipo != TipoComando || !a.Confirmar || strings.Join(a.Args, " ") != "blueprint add estado roles" || !strings.Contains(a.Etiqueta, "2") {
		t.Fatalf("los 2 con la misma receta, en un solo comando de qdd con «s  Sí»: %+v", a)
	}
	if _, err := BuscarAccion(AccionesDeInventario(inventario), "blueprint:actualizar-sin-ia"); err == nil {
		t.Fatal("con uno solo basta su propio ⬆ Actualizar")
	}
}
