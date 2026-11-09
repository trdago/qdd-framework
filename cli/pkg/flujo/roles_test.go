package flujo

import (
	"math/rand"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// Pedido de la persona (2026-11-08): «le falta peso visual […] usar distintos tamaños de letras, colores para marcar
// cosas, siempre respetando el branding». qdd dice qué trozo de cada línea es el título del nodo, cuál su detalle y
// cuál su estado; el resto es borde. El panel pinta cada rol con la marca sin tocar las columnas.

func rolesDe(segmentos [][]Segmento, nodo string) map[string][]string {
	roles := map[string][]string{}
	for _, fila := range segmentos {
		for _, s := range fila {
			if s.Nodo != nodo {
				continue
			}
			roles[s.Rol] = append(roles[s.Rol], s.Texto)
		}
	}
	return roles
}

func TestRoles_ElTituloElDetalleYElEstadoVanMarcados(t *testing.T) {
	f := conDescripciones()
	f.Nodos[0].Tarea = "t1"
	estado := func(string) string { return "✅ hecho" }
	lineas, segmentos := DibujarConOpciones(f, 80, estado, Opciones{Detalle: true})
	revisarSegmentos(t, f, lineas, segmentos)
	a := rolesDe(segmentos, "a")
	if strings.Join(a[RolTitulo], " ") != "Barra inferior" {
		t.Fatalf("el título de a es solo su nombre, sin bordes: %q\n%s", a[RolTitulo], strings.Join(lineas, "\n"))
	}
	if !strings.HasPrefix(strings.Join(a[RolDetalle], " "), "Siempre a mano") {
		t.Fatalf("la descripción va como detalle: %q", a[RolDetalle])
	}
	if strings.Join(a[RolEstado], "") != "✅ hecho" {
		t.Fatalf("el estado de su tarea va aparte: %q", a[RolEstado])
	}
	if c := rolesDe(segmentos, "c"); strings.Join(c[RolTitulo], "") != "Resuelto" {
		t.Fatalf("un resultado también marca su título: %q", c[RolTitulo])
	}
}

// El borde nunca lleva letras: lo que se lee del nodo siempre tiene un rol, y el panel no lo pinta como trazo.
func TestRoles_ElBordeNuncaLlevaTexto(t *testing.T) {
	azar := rand.New(rand.NewSource(21))
	for ronda := 0; ronda < 200; ronda++ {
		f := flujoAlAzar(azar)
		for _, ancho := range []int{30, 60} {
			lineas, segmentos := DibujarConOpciones(f, ancho, sinEstado, Opciones{Elegido: "n1", Detalle: true})
			revisarSegmentos(t, f, lineas, segmentos)
			for i, fila := range segmentos {
				for _, s := range fila {
					if s.Nodo == "" || s.Rol != "" || !nombreAlAzar.MatchString(s.Texto) {
						continue
					}
					t.Fatalf("ronda %d, línea %d: un borde de %s lleva texto: %q\n%s", ronda, i, s.Nodo, s.Texto, strings.Join(lineas, "\n"))
				}
			}
		}
	}
}
