package estado

import (
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'

func estadoConDesfases() Estado {
	return Estado{
		Formato: 1,
		Etapas:  []string{"Código", "PR"},
		Frentes: []Frente{
			{ID: "cerrada", Nombre: "Cerrada con espera", Estado: EstadoCerrado, Celdas: map[string]Celda{"Código": {Estado: Espera, Bloqueo: "¿Falta algo?", DeQuien: "persona"}, "PR": {Estado: Falla, Falla: "rompió"}}},
			{ID: "descartada", Nombre: "Descartada", Estado: EstadoDescartado, Celdas: map[string]Celda{"Código": {Estado: Espera, Bloqueo: "x", DeQuien: "persona"}}},
			{ID: "lista", Nombre: "Lista y abierta", Estado: EstadoAbierto, Celdas: map[string]Celda{"Código": {Estado: Hecho, Evidencia: "abc"}, "PR": {Estado: NoAplica}}},
			{ID: "viva", Nombre: "Viva", Estado: EstadoAbierto, Celdas: map[string]Celda{"Código": {Estado: Espera, Bloqueo: "¿Sigue?", DeQuien: "persona"}, "PR": {Estado: Hecho, Evidencia: "#1"}}},
			{ID: "bien", Nombre: "Cerrada sin esperas", Estado: EstadoCerrado, Celdas: map[string]Celda{"Código": {Estado: Hecho, Evidencia: "abc"}, "PR": {Estado: Hecho, Evidencia: "#2"}}},
		},
	}
}

// MEJ-010: en sfrc, 6 tareas cerradas seguían con un ⏳ a nombre de la persona y aparecían en esperas (FND-151).
func TestDatos_LoQuePendeDeUnaTareaCerradaNoSeLista_FND151(t *testing.T) {
	d := Datos(estadoConDesfases(), time.Now())
	if len(d.Esperas) != 1 || d.Esperas[0].Frente != "Viva" {
		t.Fatalf("solo espera la tarea abierta: %+v", d.Esperas)
	}
	if len(d.Fallas) != 0 {
		t.Fatalf("una tarea cerrada no falla: %+v", d.Fallas)
	}
}

func TestDesfasadas_DiceCadaTareaQueYaNoDiceLoQuePasa_MEJ010(t *testing.T) {
	got := Desfasadas(estadoConDesfases())
	if len(got.Listas) != 1 || got.Listas[0] != "lista" {
		t.Fatalf("abierta con todo listo: %+v", got.Listas)
	}
	if len(got.CerradasConEspera) != 1 || got.CerradasConEspera[0] != "cerrada" {
		t.Fatalf("cerrada que conserva un ⏳: %+v", got.CerradasConEspera)
	}
}

func TestDesfasadas_SinNadaQueAvisarVaVacia_MEJ010(t *testing.T) {
	e := Estado{Formato: 1, Etapas: []string{"Código"}, Frentes: []Frente{
		{ID: "viva", Nombre: "Viva", Estado: EstadoAbierto, Celdas: map[string]Celda{"Código": {Estado: Espera, Bloqueo: "¿Sigue?", DeQuien: "persona"}}},
	}}
	got := Desfasadas(e)
	if len(got.Listas)+len(got.CerradasConEspera) != 0 {
		t.Fatalf("sin avisos: %+v", got)
	}
}
