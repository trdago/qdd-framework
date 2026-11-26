package consola

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// Seguro contra lanzar dos veces lo mismo (FND-154, pedido de la persona, 2026-11-05). Un pedido hecho sale de la cola, y
// dos textos distintos para lo mismo pasaban como pedidos distintos: «Cerrar issue #112» corrió dos veces en un minuto.
// qdd recuerda 24 h el efecto de lo que terminó bien: lo mismo no se vuelve a pedir, salvo que se diga con de_nuevo. Es
// memoria local de la máquina, y si no se puede leer no bloquea nada: el seguro no detiene el trabajo.

// VidaHechos es cuánto recuerda qdd lo que ya corrió.
const VidaHechos = 24 * time.Hour

var (
	reCerrarIssue = regexp.MustCompile(`^gh\s+issue\s+close\s+#?(\d+)\b`)
	reMergearPR   = regexp.MustCompile(`^(?:gh\s+pr\s+merge|qdd\s+consola\s+mergear-pr)\s+#?(\d+)\b`)
	reRepo        = regexp.MustCompile(`(?:^|\s)(?:-R|--repo)[=\s]+(\S+)`)
	reComillas    = regexp.MustCompile(`'[^']*'|"[^"]*"`)
)

// efectoDe dice qué logra un comando, para reconocerlo aunque cambie el texto: cerrar un issue y mergear un PR valen por
// su número. Un comando encadenado hace algo más que eso, así que vale por su texto completo, igual que cualquier otro.
func efectoDe(comando, carpeta string) string {
	sinTexto := reComillas.ReplaceAllString(strings.TrimSpace(comando), "''")
	generico := "comando:" + carpeta + ":" + strings.Join(strings.Fields(comando), " ")
	if strings.ContainsAny(sinTexto, ";&|\n") {
		return generico
	}
	if m := reCerrarIssue.FindStringSubmatch(sinTexto); m != nil {
		return "issue:" + repoDe(sinTexto) + ":" + m[1]
	}
	if m := reMergearPR.FindStringSubmatch(sinTexto); m != nil {
		return "pr:" + m[1]
	}
	return generico
}

func repoDe(comando string) string {
	m := reRepo.FindStringSubmatch(comando)
	if m == nil {
		return ""
	}
	return m[1]
}

// YaHecho es la respuesta a pedir algo que ya corrió: dice qué fue, cuándo y cómo repetirlo.
type YaHecho struct {
	Motivo string
	Fin    time.Time
}

func (y *YaHecho) Error() string {
	return fmt.Sprintf("ya se hizo el %s («%s»): no se pide de nuevo. Si de verdad hay que repetirlo, pídelo con de_nuevo (--de-nuevo)", y.Fin.Local().Format("02/01 15:04"), y.Motivo)
}

// hechoAnotado es un pedido que terminó bien. Tecla y Tarea dicen qué ⏳ ya quedó respondido (FND-176).
type hechoAnotado struct {
	Efecto string    `json:"efecto"`
	Motivo string    `json:"motivo"`
	Fin    time.Time `json:"fin"`
	Tecla  string    `json:"tecla,omitempty"`
	Tarea  string    `json:"tarea,omitempty"`
	// Huella: en un git push, las ramas y tags que había al terminar (MEJ-053). Si cambiaron, no es repetido.
	Huella string `json:"huella,omitempty"`
}

func rutaHechos(root string) (string, error) {
	return estado.RutaLocal(root, "pedidos-hechos")
}

func leerHechos(root string) []hechoAnotado {
	ruta, err := rutaHechos(root)
	if err != nil {
		return nil
	}
	b, err := os.ReadFile(ruta)
	if err != nil {
		return nil
	}
	var hechos []hechoAnotado
	if json.Unmarshal(b, &hechos) != nil {
		return nil
	}
	return hechos
}

// hechoReciente busca el efecto en lo que corrió en las últimas VidaHechos; un push con otra huella no cuenta. Quien
// lo llama tiene la cola bloqueada.
func hechoReciente(root, efecto, huella string, now time.Time) *YaHecho {
	var ya *YaHecho
	for _, h := range leerHechos(root) {
		if h.Huella != "" && h.Huella != huella {
			continue
		}
		if h.Efecto == efecto && now.Sub(h.Fin) < VidaHechos {
			ya = &YaHecho{Motivo: h.Motivo, Fin: h.Fin}
		}
	}
	return ya
}

// hechosRecientes es lo que terminó bien en las últimas VidaHechos. Quien lo llama tiene la cola bloqueada.
func hechosRecientes(root string, now time.Time) []hechoAnotado {
	var out []hechoAnotado
	for _, h := range leerHechos(root) {
		if now.Sub(h.Fin) < VidaHechos {
			out = append(out, h)
		}
	}
	return out
}

// anotarHecho recuerda lo que terminó bien y olvida lo vencido. Quien lo llama tiene la cola bloqueada. Si no se puede
// escribir, solo se pierde el seguro de este pedido: el resultado sigue en la cola.
func anotarHecho(root string, p Pedido, fin time.Time) {
	ruta, err := rutaHechos(root)
	if err != nil {
		return
	}
	hechos := []hechoAnotado{{Efecto: efectoDe(p.Comando, p.Carpeta), Motivo: p.Motivo, Fin: fin, Tecla: p.Tecla, Tarea: p.Tarea, Huella: huellaPush(p.Comando, p.Carpeta)}}
	for _, h := range leerHechos(root) {
		if fin.Sub(h.Fin) < VidaHechos {
			hechos = append(hechos, h)
		}
	}
	b, err := json.MarshalIndent(hechos, "", "  ")
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Dir(ruta), 0o700) != nil {
		return
	}
	tmp := ruta + ".tmp"
	if os.WriteFile(tmp, b, 0o600) != nil {
		return
	}
	_ = os.Rename(tmp, ruta)
}

// teclasHechas son, por tarea, las teclas de sus pedidos que terminaron bien en las últimas VidaHechos.
func teclasHechas(root string, now time.Time) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for _, h := range leerHechos(root) {
		if h.Tarea == "" || h.Tecla == "" || now.Sub(h.Fin) >= VidaHechos {
			continue
		}
		if out[h.Tarea] == nil {
			out[h.Tarea] = map[string]bool{}
		}
		out[h.Tarea][h.Tecla] = true
	}
	return out
}
