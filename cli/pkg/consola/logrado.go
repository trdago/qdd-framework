package consola

import (
	"encoding/json"
	"os"
	"sort"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// Logrado es el historial de lo que salió bien (rediseño de QDD Móvil elegido por la persona, 2026-11-09): pedidos
// hechos con su salida, tareas cerradas con su motivo, objetivos cumplidos con su evidencia y lo que llegó del teléfono.
// Las tareas y los objetivos salen del estado; los pedidos salen de la cola a los VidaHecho, así que qdd los anota al
// terminar bien en un archivo local de la máquina. Si no se puede escribir, solo falta esa entrada: nada se bloquea.

// Tipos de entrada en lo logrado.
const (
	LogroPedido   = "pedido"
	LogroTarea    = "tarea"
	LogroObjetivo = "objetivo"
	LogroMensaje  = "mensaje"

	// ConLogrado es lo que pide qdd consola panel --con para recibir lo logrado.
	ConLogrado = "logrado"
	// OrigenPanel es el de un pedido que la persona confirmó en el PC.
	OrigenPanel = "panel"

	VidaLogrado     = 14 * 24 * time.Hour
	TopeLogrado     = 60
	TopeSalidaLogro = 2000
	// topeAnotados: lo que guarda el archivo local, más que TopeLogrado para que las tareas no lo desplacen.
	topeAnotados = 200

	prefijoLogroPedido   = "pedido:"
	prefijoLogroTelefono = "telefono:"
)

// Logro es una entrada de lo logrado. En es RFC3339 en UTC.
type Logro struct {
	Tipo      string `json:"tipo"`
	ID        string `json:"id"`
	Titulo    string `json:"titulo"`
	En        string `json:"en"`
	Detalle   string `json:"detalle,omitempty"`
	Salida    string `json:"salida,omitempty"`
	Origen    string `json:"origen,omitempty"`
	Motivo    string `json:"motivo,omitempty"`
	Criterio  string `json:"criterio,omitempty"`
	Evidencia string `json:"evidencia,omitempty"`
	Estado    string `json:"estado,omitempty"`
}

// Logrado es lo que salió bien en los últimos VidaLogrado, del más nuevo al más viejo, como mucho TopeLogrado.
func Logrado(root string, now time.Time) []Logro {
	todos := append(leerLogrados(root), logradoDelEstado(root)...)
	desde := now.Add(-VidaLogrado)
	out := []Logro{}
	for _, l := range todos {
		t, err := time.Parse(time.RFC3339, l.En)
		if err != nil || t.Before(desde) {
			continue
		}
		l.En = t.UTC().Format(time.RFC3339)
		out = append(out, l)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].En > out[j].En })
	return out[:min(len(out), TopeLogrado)]
}

func logradoDelEstado(root string) []Logro {
	e, err := estado.Load(root)
	if err != nil {
		return nil
	}
	var out []Logro
	for _, f := range e.Frentes {
		if f.Estado != estado.EstadoCerrado || f.CerradoEl == "" {
			continue
		}
		out = append(out, Logro{Tipo: LogroTarea, ID: f.ID, Titulo: f.Nombre, En: f.CerradoEl, Motivo: f.MotivoCierre})
	}
	for _, o := range e.Objetivos {
		if o.Estado != estado.ObjetivoCumplido || o.CerradoEl == "" {
			continue
		}
		out = append(out, Logro{Tipo: LogroObjetivo, ID: o.ID, Titulo: o.Nombre, En: o.CerradoEl, Criterio: o.Criterio, Evidencia: o.Evidencia})
	}
	return out
}

// anotarPedidosLogrados anota los pedidos de la cola que terminaron bien. Quien lo llama tiene la cola bloqueada.
func anotarPedidosLogrados(root string, cola []Pedido) {
	var nuevos []Logro
	for _, p := range cola {
		if p.Estado != EstadoHecho || p.Codigo != 0 || p.Fin == nil {
			continue
		}
		nuevos = append(nuevos, Logro{
			Tipo: LogroPedido, ID: prefijoLogroPedido + p.ID, Titulo: p.Motivo, En: p.Fin.UTC().Format(time.RFC3339),
			Detalle: p.Comando, Salida: finalDe(p.Salida, TopeSalidaLogro-1), Origen: OrigenPanel,
		})
	}
	anotarLogrados(root, nuevos)
}

// anotarTelefonoLogrado anota lo del teléfono que terminó bien: un comando como pedido de origen telefono, una
// instrucción entregada como mensaje. Quien lo llama tiene la cola del teléfono bloqueada.
func anotarTelefonoLogrado(root string, cola []Ejecucion) {
	var nuevos []Logro
	for _, e := range cola {
		if e.Estado != EstadoHecho || e.Codigo != 0 || e.Fin == nil {
			continue
		}
		l := Logro{Tipo: LogroPedido, ID: prefijoLogroTelefono + e.Solicitud, Titulo: e.Texto, En: e.Fin.UTC().Format(time.RFC3339), Detalle: e.Texto, Salida: finalDe(e.Salida, TopeSalidaLogro-1), Origen: OrigenTelefono}
		if e.Tipo == EjecutarIA {
			l = Logro{Tipo: LogroMensaje, ID: prefijoLogroTelefono + e.Solicitud, Titulo: e.Texto, En: l.En, Estado: "entregado"}
		}
		nuevos = append(nuevos, l)
	}
	anotarLogrados(root, nuevos)
}

func rutaLogrados(root string) (string, error) {
	return estado.RutaLocal(root, "logrado")
}

func leerLogrados(root string) []Logro {
	ruta, err := rutaLogrados(root)
	if err != nil {
		return nil
	}
	b, err := os.ReadFile(ruta)
	if err != nil {
		return nil
	}
	var l []Logro
	if json.Unmarshal(b, &l) != nil {
		return nil
	}
	return l
}

// anotarLogrados suma lo nuevo (una entrada ya anotada no se repite) y olvida lo de más de VidaLogrado.
func anotarLogrados(root string, nuevos []Logro) {
	if len(nuevos) == 0 {
		return
	}
	ruta, err := rutaLogrados(root)
	if err != nil {
		return
	}
	_ = conArchivo(ruta, "lo logrado", func() error {
		anotados := leerLogrados(root)
		vistos := map[string]bool{}
		for _, l := range anotados {
			vistos[l.ID] = true
		}
		for _, l := range nuevos {
			if vistos[l.ID] {
				continue
			}
			vistos[l.ID] = true
			anotados = append(anotados, l)
		}
		return guardarJSON(ruta, vigentes(anotados))
	})
}

// vigentes deja lo de los últimos VidaLogrado (contado desde lo más nuevo anotado), como mucho topeAnotados.
func vigentes(l []Logro) []Logro {
	sort.SliceStable(l, func(i, j int) bool { return l[i].En > l[j].En })
	if len(l) == 0 {
		return l
	}
	ultimo, err := time.Parse(time.RFC3339, l[0].En)
	if err != nil {
		return l[:min(len(l), topeAnotados)]
	}
	out := []Logro{}
	for _, x := range l[:min(len(l), topeAnotados)] {
		t, err := time.Parse(time.RFC3339, x.En)
		if err != nil || ultimo.Sub(t) > VidaLogrado {
			continue
		}
		out = append(out, x)
	}
	return out
}
