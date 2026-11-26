package estado

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'
// Esperas que se responden solas (pedido de la persona, 2026-11-08): un ⏳ que la IA podría contestarse lleva una
// verificación tipada, y qdd la revisa cada CadaVerificacion. Cumplida, la celda pasa a ✅ con su evidencia; un error se
// avisa y nunca la cierra. Solo tipos conocidos, nunca un comando suelto: el estado se commitea y no puede hacer correr
// cualquier cosa en el PC de la persona. La señal (SenalVerifica) va en el ⏳, así que el panel, el teléfono y
// 'qdd estado ver' la muestran sin cambiar, y la espera no se pregunta.

const (
	// CadaVerificacion es cada cuánto se revisa un ⏳ con --si-toca.
	CadaVerificacion = 10 * time.Minute
	// SenalVerifica es lo que lleva el ⏳ para decir que se responde solo.
	SenalVerifica = "⟳ se responde sola"
)

// TiposVerificacion son las verificaciones que qdd sabe hacer, con su forma.
var TiposVerificacion = map[string]*regexp.Regexp{
	"npm":     regexp.MustCompile(`^(@[a-z0-9._-]+/)?[a-z0-9._-]+@[0-9A-Za-z.+-]+$`), // npm:<paquete>@<versión> está publicado
	"pr":      regexp.MustCompile(`^[0-9]+$`),                                        // pr:<n> está mergeado
	"tag":     regexp.MustCompile(`^[0-9A-Za-z._/-]+$`),                              // tag:<tag> está en origin
	"pedido":  regexp.MustCompile(`^[0-9]+$`),                                        // pedido:<n> del panel está hecho
	"archivo": regexp.MustCompile(`^[^\x00]+$`),                                      // archivo:<ruta> existe en el proyecto
}

// Verificadores hacen cada tipo: (cumplida, qué se vio, error). Los reales viven en cmd (usan npm, gh, git y el panel).
type Verificadores map[string]func(root, arg string) (bool, string, error)

// ResumenVerificacion es lo que hizo una pasada.
type ResumenVerificacion struct {
	Revisadas []string `json:"revisadas"`
	Cumplidas []string `json:"cumplidas"`
	Avisos    []string `json:"avisos"`
}

// SepararVerificacion parte «tipo:argumento» y lo valida.
func SepararVerificacion(v string) (string, string, error) {
	tipo, arg, ok := strings.Cut(strings.TrimSpace(v), ":")
	forma, conocido := TiposVerificacion[tipo]
	if !ok || !conocido {
		return "", "", fmt.Errorf("verificación %q: usa %s (tipo:argumento), nunca un comando", v, tiposConocidos())
	}
	if !forma.MatchString(arg) {
		return "", "", fmt.Errorf("verificación %q: el argumento de %s no tiene la forma esperada", v, tipo)
	}
	return tipo, arg, nil
}

func tiposConocidos() string {
	var tipos []string
	for t := range TiposVerificacion {
		tipos = append(tipos, t+":")
	}
	sort.Strings(tipos)
	return strings.Join(tipos, ", ")
}

// conVerificacion valida la verificación de un cambio y agrega la señal al ⏳.
func conVerificacion(c Cambio, estado string) (string, string, error) {
	v := strings.TrimSpace(c.Verifica)
	if v == "" {
		return "", c.Bloqueo, nil
	}
	if estado != Espera {
		return "", "", fmt.Errorf("solo un ⏳ se responde solo: --verifica va con el estado espera")
	}
	if _, _, err := SepararVerificacion(v); err != nil {
		return "", "", err
	}
	bloqueo := strings.TrimSpace(c.Bloqueo)
	if strings.Contains(bloqueo, SenalVerifica) {
		return v, bloqueo, nil
	}
	return v, strings.TrimSpace(bloqueo + " · " + SenalVerifica + " (" + v + ")"), nil
}

type ultimaVerificacion struct {
	En        time.Time `json:"en"`
	Resultado string    `json:"resultado"`
}

// Verificar revisa los ⏳ con verificación de las tareas abiertas; con siToca, solo los que no se revisaron en
// CadaVerificacion. Una cumplida pasa a ✅ con la evidencia de lo que se vio.
func Verificar(root string, vs Verificadores, siToca bool, now time.Time) (ResumenVerificacion, error) {
	r := ResumenVerificacion{Revisadas: []string{}, Cumplidas: []string{}, Avisos: []string{}}
	e, err := Load(root)
	if err != nil {
		return r, err
	}
	ultimas := leerUltimas(root)
	for _, f := range e.Abiertos() {
		for _, etapa := range e.EtapasDe(f) {
			c := f.Celdas[etapa]
			if c.Verifica == "" || NormalizarEstado(c.Estado) != Espera {
				continue
			}
			clave := f.ID + "/" + etapa
			if siToca && now.Sub(ultimas[clave].En) < CadaVerificacion {
				continue
			}
			r.Revisadas = append(r.Revisadas, clave)
			cumplida, visto, err := verificarUna(root, vs, c.Verifica)
			ultimas[clave] = ultimaVerificacion{En: now.UTC(), Resultado: resultado(cumplida, visto, err)}
			if err != nil {
				r.Avisos = append(r.Avisos, clave+": "+err.Error())
				continue
			}
			if !cumplida {
				continue
			}
			cambio := Cambio{Frente: f.ID, Etapa: etapa, Estado: Hecho, Evidencia: "⟳ " + visto + " (verificado por qdd, " + c.Verifica + ")"}
			if err := Set(root, cambio, now); err != nil {
				r.Avisos = append(r.Avisos, clave+": "+err.Error())
				continue
			}
			delete(ultimas, clave)
			r.Cumplidas = append(r.Cumplidas, clave)
		}
	}
	return r, guardarUltimas(root, ultimas)
}

func verificarUna(root string, vs Verificadores, v string) (bool, string, error) {
	tipo, arg, err := SepararVerificacion(v)
	if err != nil {
		return false, "", err
	}
	hacer, ok := vs[tipo]
	if !ok {
		return false, "", fmt.Errorf("este qdd no sabe verificar %s:", tipo)
	}
	return hacer(root, arg)
}

func resultado(cumplida bool, visto string, err error) string {
	if err != nil {
		return "error: " + err.Error()
	}
	if cumplida {
		return "cumplida: " + visto
	}
	return "todavía no: " + visto
}

func rutaUltimas(root string) (string, error) {
	dir, err := RutaLocal(root, "verificaciones")
	if err != nil {
		return "", err
	}
	return dir + ".json", nil
}

func leerUltimas(root string) map[string]ultimaVerificacion {
	ultimas := map[string]ultimaVerificacion{}
	ruta, err := rutaUltimas(root)
	if err != nil {
		return ultimas
	}
	datos, err := os.ReadFile(ruta)
	if err != nil {
		return ultimas
	}
	_ = json.Unmarshal(datos, &ultimas)
	return ultimas
}

func guardarUltimas(root string, ultimas map[string]ultimaVerificacion) error {
	ruta, err := rutaUltimas(root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		return err
	}
	datos, err := json.MarshalIndent(ultimas, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ruta, datos, 0o600)
}

// RevisionVerificaciones es lo que doctor ve de las esperas que se responden solas.
type RevisionVerificaciones struct {
	Hay       bool
	Fallan    []string // tarea/etapa: el error de su última revisión
	Invalidas []string // tarea/etapa: una verificación que este qdd no entiende (editada a mano)
}

// RevisarVerificaciones lee, sin correr nada, cómo terminó la última revisión de cada ⏳ verificado.
func RevisarVerificaciones(root string) RevisionVerificaciones {
	var r RevisionVerificaciones
	e, err := Load(root)
	if err != nil {
		return r
	}
	ultimas := leerUltimas(root)
	for _, f := range e.Abiertos() {
		for _, etapa := range e.EtapasDe(f) {
			c := f.Celdas[etapa]
			if c.Verifica == "" || NormalizarEstado(c.Estado) != Espera {
				continue
			}
			r.Hay = true
			clave := f.ID + "/" + etapa
			if _, _, err := SepararVerificacion(c.Verifica); err != nil {
				r.Invalidas = append(r.Invalidas, clave+" ("+c.Verifica+")")
				continue
			}
			if u := ultimas[clave]; strings.HasPrefix(u.Resultado, "error: ") {
				r.Fallan = append(r.Fallan, clave+": "+strings.TrimPrefix(u.Resultado, "error: "))
			}
		}
	}
	return r
}
