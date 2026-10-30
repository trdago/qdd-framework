package estado

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'
// Escritura validada: 'set', 'cerrar' e 'init' nunca dejan un archivo que no pase el lector automático en el frente
// que tocan, y escriben de forma atómica.

// Cambio es una celda nueva (o reemplazada) de un frente.
type Cambio struct {
	Frente             string
	Nombre             string
	Etapa              string
	Estado             string
	Evidencia          string
	Bloqueo            string
	DeQuien            string
	Falla              string
	RequiereAprobacion string
	Aprobado           bool
}

// ErrSinArchivo: el proyecto todavía no tiene estado.yaml (se crea con Init).
var ErrSinArchivo = errors.New("el proyecto no tiene " + Ruta + ": créalo con 'qdd estado init'")

// Init crea el archivo con sus etapas. Falla si ya existe.
func Init(root string, etapas []string, now time.Time) error {
	if len(etapas) == 0 {
		return errors.New("init necesita al menos una etapa")
	}
	if _, err := os.Stat(filepath.Join(root, Ruta)); err == nil {
		return fmt.Errorf("%s ya existe", Ruta)
	}
	return Save(root, Estado{Formato: Formato, Etapas: etapas, Actualizado: now.Format(time.RFC3339), Frentes: []Frente{}})
}

func cargarParaEscribir(root string) (Estado, error) {
	e, err := Load(root)
	if errors.Is(err, os.ErrNotExist) {
		return Estado{}, ErrSinArchivo
	}
	return e, err
}

// Set crea o actualiza una celda. La aprobación pendiente se conserva aunque el cambio no la repita: un set no la
// salta (EPF-C06).
func Set(root string, c Cambio, now time.Time) error {
	e, err := cargarParaEscribir(root)
	if err != nil {
		return err
	}
	if !e.tieneEtapa(c.Etapa) {
		return fmt.Errorf("la etapa %q no está en el archivo (etapas: %s)", c.Etapa, strings.Join(e.Etapas, ", "))
	}
	estado := NormalizarEstado(c.Estado)
	if estado == "" {
		return fmt.Errorf("estado %q: usa ✅ ⏳ ❌ — (o ok, espera, falla, na)", c.Estado)
	}
	i := e.buscar(c.Frente)
	if i < 0 {
		e.Frentes = append(e.Frentes, frenteNuevo(e, c))
		i = len(e.Frentes) - 1
	}
	if c.Nombre != "" {
		e.Frentes[i].Nombre = c.Nombre
	}
	anterior := e.Frentes[i].Celdas[c.Etapa]
	celda := Celda{Estado: estado, Evidencia: c.Evidencia, Bloqueo: c.Bloqueo, DeQuien: c.DeQuien, Falla: c.Falla,
		RequiereAprobacion: anterior.RequiereAprobacion, Aprobado: anterior.Aprobado || c.Aprobado}
	if c.RequiereAprobacion != "" {
		celda.RequiereAprobacion = c.RequiereAprobacion
	}
	e.Frentes[i].Celdas[c.Etapa] = celda
	return guardarValidado(root, e, c.Frente, now)
}

func frenteNuevo(e Estado, c Cambio) Frente {
	nombre := c.Nombre
	if nombre == "" {
		nombre = c.Frente
	}
	f := Frente{ID: c.Frente, Nombre: nombre, Estado: EstadoAbierto, Celdas: map[string]Celda{}}
	for _, etapa := range e.Etapas {
		f.Celdas[etapa] = Celda{Estado: NoAplica}
	}
	return f
}

// Cerrar cierra (o descarta) un frente con su motivo (EPF-C05).
func Cerrar(root, frente, motivo string, descartado bool, now time.Time) error {
	if strings.TrimSpace(motivo) == "" {
		return errors.New("cerrar un frente exige su motivo (--motivo)")
	}
	e, err := cargarParaEscribir(root)
	if err != nil {
		return err
	}
	i := e.buscar(frente)
	if i < 0 {
		return fmt.Errorf("no hay un frente %q", frente)
	}
	e.Frentes[i].Estado = EstadoCerrado
	if descartado {
		e.Frentes[i].Estado = EstadoDescartado
	}
	e.Frentes[i].MotivoCierre = motivo
	e.Frentes[i].CerradoEl = now.Format(time.RFC3339)
	if err := guardarValidado(root, e, frente, now); err != nil {
		return err
	}
	// Una tarea cerrada o descartada deja de estar en curso (§3.13); la actividad es local y su falla no deshace el cierre.
	_ = Terminar(root, frente)
	return nil
}

// guardarValidado escribe solo si el frente tocado cumple las invariantes. Las violaciones de otros frentes no bloquean
// (se ven con 'check'), para no impedir arreglar un frente por culpa de otro.
func guardarValidado(root string, e Estado, frente string, now time.Time) error {
	var propias []string
	for _, v := range Check(e, nil) {
		if v.Frente == frente {
			propias = append(propias, v.String())
		}
	}
	if len(propias) > 0 {
		return fmt.Errorf("no se escribió: %s", strings.Join(propias, "; "))
	}
	e.Actualizado = now.Format(time.RFC3339)
	return Save(root, e)
}
