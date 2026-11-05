package consola

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// Sincronía panel–chat (pedido del usuario, 2026-11-01): lo que el panel espera de la persona (un pedido, una ficha
// por aprobar, una etapa ⏳ suya) puede resolverse en el chat: «ya lo mergeé», «no, eso no va», «hazlo distinto». Con
// cada mensaje, «qdd consola ciclo --contexto» le dice a la IA qué espera el panel, solo si cambió desde la última vez
// (sin cambios no gasta tokens); la IA lo refleja en el acto y el panel lo muestra en su próxima lectura.

const (
	// MaxEsperas es cuántas esperas se nombran; el resto se cuenta.
	MaxEsperas = 8
	// maxTextoEspera recorta cada una: el contexto es una línea corta, no el detalle.
	maxTextoEspera = 70
)

// Esperas son las cosas que el panel espera de la persona, una por línea: los pedidos que puede ejecutar, las fichas
// en borrador y las etapas ⏳ cuyo «de quién» es la persona.
func Esperas(root string, now time.Time) []string {
	var out []string
	for _, p := range PedidosA(root, now) {
		if !p.Ejecutable() || p.Opcional() || len(EsperaDe(root, p.ID)) > 0 {
			continue
		}
		out = append(out, fmt.Sprintf("pedido %s «%s»", p.Tecla, recortar(p.Motivo, maxTextoEspera)))
	}
	for _, m := range borradores(root) {
		out = append(out, fmt.Sprintf("ficha «%s» por aprobar (%s)", recortar(m.Nombre, maxTextoEspera), m.ID))
	}
	e, err := estado.Load(root)
	if err != nil {
		return out
	}
	for _, f := range e.Abiertos() {
		for _, etapa := range e.EtapasDe(f) {
			c := f.Celdas[etapa]
			if estado.NormalizarEstado(c.Estado) != estado.Espera || !esDeLaPersona(c.DeQuien) {
				continue
			}
			out = append(out, fmt.Sprintf("⏳ %s · %s: %s", f.Nombre, etapa, recortar(c.Bloqueo, maxTextoEspera)))
		}
	}
	return out
}

func esDeLaPersona(deQuien string) bool {
	return strings.Contains(strings.ToLower(deQuien), "persona")
}

// ContextoCiclo es lo que la IA recibe con el mensaje de la persona: qué pedidos de su sesión se retiraron, qué espera el
// panel y cómo reflejar lo que ella resuelva en el chat. Vacío si no hay nada nuevo.
func ContextoCiclo(root, sesion string, now time.Time) string {
	var partes []string
	partes = append(partes, tomarRetirados(root, sesion, now)...)
	if esperas := contextoEsperas(root, now); esperas != "" {
		partes = append(partes, esperas)
	}
	return strings.Join(partes, "\n")
}

// contextoEsperas dice qué espera el panel de la persona. Vacío si no espera nada o si es lo mismo que ya recibió.
func contextoEsperas(root string, now time.Time) string {
	esperas := Esperas(root, now)
	suma := sha256.Sum256([]byte(strings.Join(esperas, "\n")))
	huella := hex.EncodeToString(suma[:8])
	ruta, err := RutaLocal(root, "ciclo-contexto")
	if err != nil {
		return ""
	}
	if previa, err := os.ReadFile(ruta); err == nil && string(previa) == huella {
		return ""
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err == nil {
		_ = os.WriteFile(ruta, []byte(huella), 0o600)
	}
	if len(esperas) == 0 {
		return ""
	}
	resto := ""
	if len(esperas) > MaxEsperas {
		resto = fmt.Sprintf(" · y %d más", len(esperas)-MaxEsperas)
		esperas = esperas[:MaxEsperas]
	}
	return "[qdd · panel] Espera a la persona: " + strings.Join(esperas, " · ") + resto + ". " +
		"Si su mensaje lo acepta, rechaza, cambia o deja sin efecto, refléjalo ya: lo hizo ella → qdd consola descartar <tecla> --hecho; " +
		"ya no va → qdd consola descartar <tecla>; cambia → pídelo de nuevo; ⏳ → qdd_estado set; ficha → apruébala solo si ella lo dice."
}

// HechoPorLaPersona marca hecho un pedido que la persona resolvió fuera del panel (lo dijo en el chat): sale de
// «▶ Para ti» como hecho y libera a los que lo esperan.
func HechoPorLaPersona(root, idOTecla, nota string, now time.Time) error {
	return conCola(root, now, func(cola []Pedido) ([]Pedido, error) {
		i := indiceDe(cola, idOTecla)
		if i < 0 {
			return nil, fmt.Errorf("no hay un pedido '%s' en la cola", idOTecla)
		}
		if cola[i].Estado == EstadoEjecutando {
			return nil, fmt.Errorf("el pedido %s se está ejecutando: espera a que termine", idOTecla)
		}
		fin := now
		cola[i].Estado, cola[i].Fin, cola[i].Codigo, cola[i].Aviso, cola[i].AvisoEn = EstadoHecho, &fin, 0, "", nil
		cola[i].Salida = strings.TrimSpace("hecho por la persona fuera del panel. " + nota)
		return cola, nil
	})
}

// indiceDe busca un pedido por su id o por la tecla con que lo ve la persona (entre los abiertos).
func indiceDe(cola []Pedido, idOTecla string) int {
	for i, p := range cola {
		if p.ID == idOTecla {
			return i
		}
	}
	for i, p := range cola {
		if p.Tecla == idOTecla && p.Abierto() {
			return i
		}
	}
	return -1
}
