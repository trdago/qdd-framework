package estado

import (
	"fmt"
	"time"
)

// MotivoYaLoVio es el motivo de cierre de una tarea que la persona dio por vista desde el panel.
const MotivoYaLoVio = "la vio la persona desde el panel, sin medición"

// Visto es lo que dejó «✓ Ya lo vi»: las etapas que marcó y si con eso la tarea se cerró.
type Visto struct {
	Etapas  []string
	Cerrada bool
}

// PorVer lista las etapas en ⏳ de una tarea abierta que la persona puede dar por vistas. Una que requiere aprobación
// sin darla sigue esperando: ese ✅ lo da quien debía (EPF-C06).
func (e Estado) PorVer(id string) []string {
	i := e.buscar(id)
	if i < 0 || e.Frentes[i].Estado != EstadoAbierto {
		return nil
	}
	f := e.Frentes[i]
	var etapas []string
	for _, etapa := range e.EtapasDe(f) {
		c := f.Celdas[etapa]
		if NormalizarEstado(c.Estado) != Espera || (c.RequiereAprobacion != "" && !c.Aprobado) {
			continue
		}
		etapas = append(etapas, etapa)
	}
	return etapas
}

// NotaVista es la evidencia de una etapa dada por vista: dice quién y cuándo, y que nadie midió.
func NotaVista(now time.Time) string {
	return "la vio la persona el " + now.Format("02/01") + ", sin medición"
}

// YaLoVio deja ✅ las etapas por ver con NotaVista. Si no queda nada por hacer, cierra la tarea; una etapa sin empezar,
// con falla o esperando una aprobación es trabajo que falta y la deja abierta.
func YaLoVio(root, frente string, now time.Time) (Visto, error) {
	e, err := cargarParaEscribir(root)
	if err != nil {
		return Visto{}, err
	}
	if err := e.ValidarTarea(frente); err != nil {
		return Visto{}, err
	}
	etapas := e.PorVer(frente)
	if len(etapas) == 0 {
		return Visto{}, fmt.Errorf("la tarea '%s' no tiene etapas en ⏳ por ver", frente)
	}
	i := e.buscar(frente)
	for _, etapa := range etapas {
		c := e.Frentes[i].Celdas[etapa]
		c.Estado, c.Evidencia, c.Bloqueo, c.DeQuien = Hecho, NotaVista(now), "", ""
		e.Frentes[i].Celdas[etapa] = c
	}
	cerrada := !e.quedaPorHacer(e.Frentes[i])
	if cerrada {
		e.Frentes[i].Estado = EstadoCerrado
		e.Frentes[i].MotivoCierre = MotivoYaLoVio
		e.Frentes[i].CerradoEl = now.Format(time.RFC3339)
	}
	if err := guardarValidado(root, e, frente, now); err != nil {
		return Visto{}, err
	}
	if !cerrada {
		return Visto{Etapas: etapas}, nil
	}
	// Igual que al cerrar: deja de estar en curso; la actividad es local y su falla no deshace el cierre (§3.13).
	_ = Terminar(root, frente)
	return Visto{Etapas: etapas, Cerrada: true}, nil
}

func (e Estado) quedaPorHacer(f Frente) bool {
	for _, etapa := range e.EtapasDe(f) {
		s := NormalizarEstado(f.Celdas[etapa].Estado)
		if s != Hecho && s != NoAplica {
			return true
		}
	}
	return false
}
