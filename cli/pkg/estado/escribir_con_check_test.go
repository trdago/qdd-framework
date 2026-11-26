package estado

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// Rule: Prohibido usar 'else'
// FND-270: cambiar las etapas de un ciclo (qdd estado ciclo) o pasar un objetivo a otro ciclo (qdd estado objetivo
// --ciclo) reforma sus tareas y escribía estado.yaml sin pasar por Check de esas tareas. Con una Puerta nueva y la
// Producción ya en ✅, el archivo quedaba con EPF-C04, que recién veía qdd audit.

func desplegada(id, objetivo string) Frente {
	return tarea(id, objetivo, map[string]Celda{"Código": {Estado: Hecho, Evidencia: "abc1234"}, "Producción": {Estado: Hecho, Evidencia: "v1.2.0"}})
}

// noSeEscribe: el cambio se rechaza, estado.yaml queda byte a byte igual y sin violaciones.
func noSeEscribe(t *testing.T, que string, e Estado, cambiar func(root string) error) {
	t.Helper()
	root := guardarEn(t, e)
	ruta := filepath.Join(root, Ruta)
	antes, _ := os.ReadFile(ruta)
	if err := cambiar(root); err == nil {
		t.Fatalf("%s: una tarea que quedaría sin cumplir Check no se escribe", que)
	}
	despues, _ := os.ReadFile(ruta)
	if !bytes.Equal(antes, despues) {
		t.Fatalf("%s: estado.yaml cambió:\n%s", que, despues)
	}
	if vs := Check(mustLoad(t, root), nil); len(vs) != 0 {
		t.Fatalf("%s: quedó con %s", que, idsDe(vs))
	}
}

func mustLoad(t *testing.T, root string) Estado {
	t.Helper()
	e, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// contrato: estado#f47c5e80
func TestEscribir_CambiarUnCicloPasaPorCheckDeSusTareas_FND270(t *testing.T) {
	conPuerta := []string{"Código", "Puerta", "Producción"}
	propio := Estado{Etapas: []string{"Código", "Producción"},
		Ciclos:    map[string][]string{"despliegue": {"Código", "Producción"}},
		Objetivos: []Objetivo{{ID: "o", Nombre: "O", Criterio: "c", Estado: ObjetivoAbierto, Ciclo: "despliegue"}},
		Frentes:   []Frente{desplegada("t", "o")}}
	noSeEscribe(t, "qdd estado ciclo despliegue", propio, func(root string) error {
		return DeclararCiclo(root, "despliegue", conPuerta, ahora)
	})
	delProyecto := Estado{Etapas: []string{"Código", "Producción"}, Frentes: []Frente{desplegada("t", "")}}
	noSeEscribe(t, "qdd estado ciclo proyecto", delProyecto, func(root string) error {
		return DeclararCiclo(root, CicloProyecto, conPuerta, ahora)
	})
	aOtro := Estado{Etapas: []string{"Código", "Producción"},
		Ciclos:    map[string][]string{"conpuerta": conPuerta},
		Objetivos: []Objetivo{{ID: "o", Nombre: "O", Criterio: "c", Estado: ObjetivoAbierto}},
		Frentes:   []Frente{desplegada("t", "o")}}
	noSeEscribe(t, "qdd estado objetivo --ciclo", aOtro, func(root string) error {
		return GuardarObjetivo(root, CambioObjetivo{ID: "o", Ciclo: "conpuerta", CambiaCiclo: true}, ahora)
	})
	// Sin la Producción hecha, la Puerta nueva entra en «·» y se escribe.
	enCurso := delProyecto
	enCurso.Frentes = []Frente{tarea("t", "", map[string]Celda{"Código": {Estado: Hecho, Evidencia: "abc1234"}, "Producción": {Estado: SinEmpezar}})}
	root := guardarEn(t, enCurso)
	if err := DeclararCiclo(root, CicloProyecto, conPuerta, ahora); err != nil {
		t.Fatalf("un ciclo que deja las tareas cumpliendo se escribe: %v", err)
	}
	if c := mustLoad(t, root).Frentes[0].Celdas["Puerta"]; c.Estado != SinEmpezar {
		t.Fatalf("la etapa nueva queda sin empezar: %+v", c)
	}
}

// contrato: estado#f47c5e80
// Las escrituras de una tarea (set, cerrar, aprobar, ya lo vi, tarea) se rechazan si esa tarea no cumple Check.
func TestEscribir_CadaEscrituraDeUnaTareaPasaPorSuCheck(t *testing.T) {
	// Pruebas en ✅ sin evidencia (EPF-C02): ninguna de estas escrituras toca esa celda.
	rota := Estado{Etapas: []string{"Código", "Pruebas", "Producción"}, Frentes: []Frente{
		tarea("t", "", map[string]Celda{
			"Código":     {Estado: Espera, Bloqueo: "¿sigo?", DeQuien: "la persona"},
			"Pruebas":    {Estado: Hecho},
			"Producción": {Estado: SinEmpezar, RequiereAprobacion: "la persona"},
		}),
	}}
	casos := map[string]func(root string) error{
		"set": func(root string) error {
			return Set(root, Cambio{Frente: "t", Etapa: "Producción", Estado: "·"}, ahora)
		},
		"cerrar":   func(root string) error { return Cerrar(root, "t", "listo", false, ahora) },
		"aprobar":  func(root string) error { return Aprobar(root, "t", "Producción", ahora) },
		"ya lo vi": func(root string) error { _, err := YaLoVio(root, "t", ahora); return err },
		"tarea":    func(root string) error { return GuardarTarea(root, CambioTarea{ID: "t", Nombre: "Otro nombre"}, ahora) },
	}
	for que, cambiar := range casos {
		root := guardarEn(t, rota)
		ruta := filepath.Join(root, Ruta)
		antes, _ := os.ReadFile(ruta)
		if err := cambiar(root); err == nil {
			t.Fatalf("%s: la tarea con un ✅ sin evidencia no se escribe", que)
		}
		if despues, _ := os.ReadFile(ruta); !bytes.Equal(antes, despues) {
			t.Fatalf("%s: estado.yaml cambió", que)
		}
	}
}
