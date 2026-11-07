package remoto

import (
	"context"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/flujo"
)

// Rule: Prohibido usar 'else'
// «Sincronizar» (la persona por qdd-movile, 2026-11-07; eligió «releer y avisar»): el teléfono manda el evento
// sincronizar sin payload; el PC olvida lo que ya había enviado, manda el estado completo releído de cero y, al final,
// sincronizado con lo que el tablero de cada proyecto con una consola abierta ya no dice bien. Nada se corrige: revisar
// solo lee. Es gratis, como sync, porque no escribe nada. qdd sincronizar da la misma revisión en el PC.

// Los tipos de inconsistencia que revisa Sincronizar.
const (
	InconsistenciaEsperaTrasPedido   = "espera_tras_pedido"   // ⏳ de la persona cuyo pedido ya terminó (FND-176)
	InconsistenciaListaSinCerrar     = "lista_sin_cerrar"     // tarea con todo listo que sigue abierta (MEJ-010)
	InconsistenciaTerminadaConEspera = "terminada_con_espera" // cerrada o descartada que conserva un ⏳ (FND-177)
	InconsistenciaFlujoInvalido      = "flujo_invalido"       // flujo que no se puede usar; detalle = id: problema
)

// Inconsistencia es algo que el tablero ya no dice bien. Tarea es su nombre.
type Inconsistencia struct {
	Tipo    string `json:"tipo"`
	Tarea   string `json:"tarea,omitempty"`
	Etapa   string `json:"etapa,omitempty"`
	Detalle string `json:"detalle,omitempty"`
}

// ProyectoSincronizado es un proyecto en la respuesta: Consola dice si se revisó (hay una sesión de Claude Code viva).
type ProyectoSincronizado struct {
	Clave           string           `json:"clave"`
	Nombre          string           `json:"nombre"`
	Consola         bool             `json:"consola"`
	Inconsistencias []Inconsistencia `json:"inconsistencias"`
}

// Sincronizado es el evento con que el PC responde a sincronizar, siempre después del estado.
type Sincronizado struct {
	Version       int                    `json:"version"`
	En            time.Time              `json:"en"`
	EstadoEnviado bool                   `json:"estado_enviado"`
	Proyectos     []ProyectoSincronizado `json:"proyectos"`
}

// Revisar lee el tablero y los flujos del proyecto y dice lo que ya no está bien, sin escribir nada.
func Revisar(root string, now time.Time) []Inconsistencia {
	out := []Inconsistencia{}
	for _, e := range consola.EsperasTrasPedido(root, now) {
		out = append(out, Inconsistencia{Tipo: InconsistenciaEsperaTrasPedido, Tarea: e.Tarea, Etapa: e.Etapa, Detalle: e.Bloqueo})
	}
	if e, err := estado.Load(root); err == nil {
		out = append(out, desfasadas(e)...)
	}
	for _, f := range flujo.Listar(root, func(string) string { return "" }) {
		if f.Problema == "" {
			continue
		}
		out = append(out, Inconsistencia{Tipo: InconsistenciaFlujoInvalido, Detalle: f.ID + ": " + f.Problema})
	}
	return out
}

func desfasadas(e estado.Estado) []Inconsistencia {
	nombres := map[string]string{}
	for _, f := range e.Frentes {
		nombres[f.ID] = f.Nombre
	}
	d := estado.Desfasadas(e)
	var out []Inconsistencia
	for _, id := range d.Listas {
		out = append(out, Inconsistencia{Tipo: InconsistenciaListaSinCerrar, Tarea: nombres[id]})
	}
	for _, id := range d.CerradasConEspera {
		out = append(out, Inconsistencia{Tipo: InconsistenciaTerminadaConEspera, Tarea: nombres[id]})
	}
	return out
}

// Sincronizacion revisa cada proyecto publicado que tiene una consola abierta.
func Sincronizacion(proyectos []ProyectoEstado, now time.Time) []ProyectoSincronizado {
	out := make([]ProyectoSincronizado, 0, len(proyectos))
	for _, p := range proyectos {
		s := ProyectoSincronizado{Clave: p.Clave, Nombre: p.Nombre, Inconsistencias: []Inconsistencia{}}
		if p.Error == "" && consola.HaySesion(p.Ruta, now) {
			s.Consola, s.Inconsistencias = true, Revisar(p.Ruta, now)
		}
		out = append(out, s)
	}
	return out
}

// alSincronizar relee de cero (armar lee cada proyecto del disco, y mandarEstado reemplaza lo enviado y reenvía los
// flujos) y responde al final con sincronizado.
func (b *Bucle) alSincronizar(ctx context.Context) {
	e := b.armar()
	b.mandarEstado(ctx, e)
	b.broadcast(ctx, "sincronizado", Sincronizado{Version: VersionProtocolo, En: b.ahora().UTC(), EstadoEnviado: b.base, Proyectos: Sincronizacion(e.Proyectos, b.ahora())})
}
