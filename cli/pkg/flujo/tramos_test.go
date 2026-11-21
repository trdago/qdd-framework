package flujo

import (
	"math/rand"
	"regexp"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// Pedido de la persona (2026-11-08): «en el panel de flujo no me gusta tener que bajar hasta abajo para una acción al
// nodo del flujo; me gustaría que fuera directa sobre el elemento para no perder el hilo». qdd dice, línea por línea, qué
// tramo del dibujo es de qué nodo, para que el panel lo haga pulsable ahí mismo.

var nombreAlAzar = regexp.MustCompile(`N\d\d`)

// revisarSegmentos: cada línea con nodos se parte en tramos que, juntos, son la línea exacta, y un tramo de un nodo
// nunca lleva el nombre de otro.
func revisarSegmentos(t *testing.T, f Flujo, lineas []string, segmentos [][]Segmento) {
	t.Helper()
	if len(segmentos) != len(lineas) {
		t.Fatalf("un renglón de segmentos por línea: %d líneas, %d renglones", len(lineas), len(segmentos))
	}
	nombres := map[string]string{}
	for _, n := range f.Nodos {
		nombres[n.Nombre] = n.ID
	}
	for i, fila := range segmentos {
		if fila == nil {
			continue
		}
		var junto strings.Builder
		for _, s := range fila {
			junto.WriteString(s.Texto)
			if s.Nodo == "" {
				continue
			}
			for _, nombre := range nombreAlAzar.FindAllString(s.Texto, -1) {
				if nombres[nombre] != s.Nodo {
					t.Fatalf("línea %d: el tramo de %s lleva %s: %q\n%s", i, s.Nodo, nombre, s.Texto, strings.Join(lineas, "\n"))
				}
			}
		}
		if junto.String() != lineas[i] {
			t.Fatalf("línea %d: los tramos no rearman la línea:\n%q\n%q", i, junto.String(), lineas[i])
		}
	}
}

// pulsables: los nodos que tienen un tramo con su nombre.
func pulsables(segmentos [][]Segmento) map[string]bool {
	vistos := map[string]bool{}
	for _, fila := range segmentos {
		for _, s := range fila {
			if s.Nodo != "" && nombreAlAzar.MatchString(s.Texto) {
				vistos[s.Nodo] = true
			}
		}
	}
	return vistos
}

func TestSegmentos_CadaNodoEsPulsableEnSuLugar(t *testing.T) {
	azar := rand.New(rand.NewSource(11))
	for ronda := 0; ronda < 400; ronda++ {
		f := flujoAlAzar(azar)
		for _, ancho := range []int{30, 50, 80} {
			lineas, segmentos := DibujarConSegmentos(f, ancho, sinEstado, "")
			revisarSegmentos(t, f, lineas, segmentos)
			vistos := pulsables(segmentos)
			for _, n := range f.Nodos {
				if !vistos[n.ID] {
					t.Fatalf("ronda %d, ancho %d: %s no se puede pulsar en el dibujo\n%s", ronda, ancho, n.Nombre, strings.Join(lineas, "\n"))
				}
			}
		}
	}
}

func TestSegmentos_LasLineasSonLasDeSiempre(t *testing.T) {
	azar := rand.New(rand.NewSource(12))
	for ronda := 0; ronda < 100; ronda++ {
		f := flujoAlAzar(azar)
		lineas, _ := DibujarConSegmentos(f, 60, sinEstado, "n1")
		if strings.Join(lineas, "\n") != strings.Join(DibujarElegido(f, 60, sinEstado, "n1"), "\n") {
			t.Fatalf("ronda %d: el dibujo cambió al pedir sus tramos", ronda)
		}
	}
}

func TestSegmentos_DosNodosEnUnaLineaSonDosTramos(t *testing.T) {
	f := deNodos(Nodo{ID: "a", Tipo: TipoPaso, Nombre: "N00", Siguen: []string{"b", "c"}}, paso("b", "N01", "d"), paso("c", "N02", "d"), paso("d", "N03"))
	lineas, segmentos := DibujarConSegmentos(f, 60, sinEstado, "")
	revisarSegmentos(t, f, lineas, segmentos)
	for i, l := range lineas {
		if !strings.Contains(l, "N01") || !strings.Contains(l, "N02") {
			continue
		}
		nodos := map[string]bool{}
		for _, s := range segmentos[i] {
			nodos[s.Nodo] = true
		}
		if !nodos["b"] || !nodos["c"] {
			t.Fatalf("lado a lado, cada caja es de su nodo: %+v", segmentos[i])
		}
		return
	}
	t.Fatalf("se esperaban las ramas lado a lado:\n%s", strings.Join(lineas, "\n"))
}

// En el panorama, la caja de un grupo es pulsable con el id del grupo: el panel la abre ahí mismo.
func TestSegmentos_LaCajaDeUnGrupoEsDelGrupo(t *testing.T) {
	_, segmentos := DibujarConSegmentos(Plegar(analisis(), Vista{Nivel: 1}), 60, sinEstado, "")
	grupos := map[string]bool{}
	for _, fila := range segmentos {
		for _, s := range fila {
			if strings.HasPrefix(s.Nodo, "grupo-") {
				grupos[s.Nodo] = true
			}
		}
	}
	if !grupos["grupo-descubrir"] || !grupos["grupo-filtrar"] || !grupos["grupo-analizar"] {
		t.Fatalf("cada caja de grupo es de su grupo: %v", grupos)
	}
}

func TestSegmentos_UnaLineaSinNodosNoTraeTramos(t *testing.T) {
	f := deNodos(paso("a", "N00", "b"), paso("b", "N01"))
	lineas, segmentos := DibujarConSegmentos(f, 40, sinEstado, "")
	for i, l := range lineas {
		if strings.TrimSpace(l) == "│" && segmentos[i] != nil {
			t.Fatalf("la línea que une dos nodos no es de ninguno: %+v", segmentos[i])
		}
	}
}
