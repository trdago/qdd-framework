package flujo

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"html"
	"strconv"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'
// Flujo para conversar (RFC 0025, fase 1): la foto que ve el cliente, el colega o quien levanta un proceso, y el archivo
// HTML que la muestra. La foto lleva solo lo que esa persona puede ver (nombre, descripción, tipo, siguen, grupo y nivel)
// limpio con la función que da quien la arma; nunca tareas, ítems ni comentarios internos. El archivo es uno solo, sin
// nada de afuera: la CSP deja correr solo su script y su estilo, por hash, y los datos van en un bloque JSON escapado.

const FormatoFoto = 1

// FotoNodo es un nodo como lo ve el cliente.
type FotoNodo struct {
	ID          string   `json:"id"`
	Tipo        string   `json:"tipo"`
	Nombre      string   `json:"nombre"`
	Descripcion string   `json:"descripcion,omitempty"`
	Grupo       string   `json:"grupo,omitempty"`
	Nivel       int      `json:"nivel"`
	Siguen      []string `json:"siguen,omitempty"`
}

// FotoVistaNodo es un nodo de un nivel ya plegado: el nodo mismo o la caja de un grupo, con lo que esconde.
type FotoVistaNodo struct {
	ID      string   `json:"id"`
	Nombre  string   `json:"nombre"`
	Siguen  []string `json:"siguen,omitempty"`
	Resumen string   `json:"resumen,omitempty"`
	Grupo   bool     `json:"grupo,omitempty"`
	Clase   string   `json:"clase,omitempty"` // la del grupo, en su caja del panorama
}

// FotoVista es un nivel: N1, N2 o N3.
type FotoVista struct {
	Nodos []FotoVistaNodo `json:"nodos"`
}

// Foto es lo que lleva el archivo.
type Foto struct {
	Formato     int          `json:"formato"`
	Flujo       string       `json:"flujo"`
	Nombre      string       `json:"nombre"`
	Descripcion string       `json:"descripcion,omitempty"`
	Plantilla   Plantilla    `json:"plantilla"`
	Revision    int          `json:"revision"`
	Version     *Version     `json:"version,omitempty"`
	Grupos      []Grupo      `json:"grupos,omitempty"`
	Generado    string       `json:"generado"`
	Nodos       []FotoNodo   `json:"nodos"`
	Vistas      [3]FotoVista `json:"vistas"`
}

// ArmarFoto arma la foto del flujo; limpiar quita del texto lo que no debe salir (rutas, secretos).
func ArmarFoto(f Flujo, limpiar func(string) string, ahora time.Time) Foto {
	foto := Foto{
		Formato: FormatoFoto, Flujo: f.ID, Nombre: limpiar(f.Nombre), Descripcion: limpiar(f.Descripcion),
		Plantilla: PlantillaDe(f), Revision: f.Revision, Generado: ahora.UTC().Format(time.RFC3339),
	}
	if len(f.Versiones) > 0 {
		v := f.Versiones[len(f.Versiones)-1]
		v.Nombre = limpiar(v.Nombre)
		foto.Version = &v
	}
	for _, g := range f.Grupos {
		foto.Grupos = append(foto.Grupos, Grupo{Nombre: limpiar(g.Nombre), Clase: limpiar(g.Clase), Descripcion: limpiar(g.Descripcion)})
	}
	niveles := Niveles(f)
	for _, n := range f.Nodos {
		foto.Nodos = append(foto.Nodos, FotoNodo{
			ID: n.ID, Tipo: n.Tipo, Nombre: limpiar(n.Nombre), Descripcion: limpiar(n.Descripcion),
			Grupo: limpiar(n.Grupo), Nivel: niveles[n.ID], Siguen: n.Siguen,
		})
	}
	for i := range foto.Vistas {
		foto.Vistas[i] = vistaDeFoto(f, i+1, limpiar)
	}
	return foto
}

func vistaDeFoto(f Flujo, nivel int, limpiar func(string) string) FotoVista {
	v := FotoVista{Nodos: []FotoVistaNodo{}}
	if nivel >= NivelDesvio {
		for _, n := range f.Nodos {
			v.Nodos = append(v.Nodos, FotoVistaNodo{ID: n.ID, Nombre: limpiar(n.Nombre), Siguen: n.Siguen})
		}
		return v
	}
	propios := map[string]bool{}
	for _, n := range f.Nodos {
		propios[n.ID] = true
	}
	for _, n := range Plegar(f, Vista{Nivel: nivel}).Nodos {
		vn := FotoVistaNodo{ID: n.ID, Nombre: limpiar(n.Nombre), Siguen: n.Siguen, Resumen: limpiar(n.Descripcion), Grupo: !propios[n.ID]}
		if vn.Grupo {
			g, _ := GrupoDe(f, n.Nombre)
			vn.Clase = limpiar(g.Clase)
		}
		v.Nodos = append(v.Nodos, vn)
	}
	return v
}

var (
	//go:embed archivo/flujo.html
	archivoHTML string
	//go:embed archivo/flujo.css
	archivoCSS string
	//go:embed archivo/flujo.js
	archivoJS string
	//go:embed archivo/fuentes/JetBrainsMono-Regular.woff2
	fuenteNormal []byte
	//go:embed archivo/fuentes/JetBrainsMono-Medium.woff2
	fuenteMedia []byte
	//go:embed archivo/fuentes/JetBrainsMono-Bold.woff2
	fuenteNegrita []byte

	// estiloDelArchivo es la fuente de la marca (JetBrains Mono 400/500/700, OFL, recortada) y el estilo, en un solo
	// bloque: el archivo no carga nada de afuera.
	estiloDelArchivo = caraDeFuente(400, fuenteNormal) + caraDeFuente(500, fuenteMedia) + caraDeFuente(700, fuenteNegrita) + archivoCSS
)

func caraDeFuente(peso int, datos []byte) string {
	return "@font-face {\n  font-family: \"JetBrains Mono\";\n  font-style: normal;\n  font-weight: " + strconv.Itoa(peso) +
		";\n  font-display: swap;\n  src: url(data:font/woff2;base64," + base64.StdEncoding.EncodeToString(datos) + ") format(\"woff2\");\n}\n"
}

func hashCSP(s string) string {
	suma := sha256.Sum256([]byte(s))
	return "'sha256-" + base64.StdEncoding.EncodeToString(suma[:]) + "'"
}

// ArchivoHTML arma el archivo de la foto. json.Marshal escapa <, > y &, así que ningún texto cierra el bloque de datos.
func ArchivoHTML(foto Foto) ([]byte, error) {
	datos, err := json.Marshal(foto)
	if err != nil {
		return nil, err
	}
	csp := "default-src 'none'; script-src " + hashCSP(archivoJS) + "; style-src " + hashCSP(estiloDelArchivo) +
		"; font-src data:; img-src data:; base-uri 'none'; form-action 'none'"
	r := strings.NewReplacer(
		"{{CSP}}", csp,
		"{{TITULO}}", html.EscapeString(foto.Nombre),
		"{{CSS}}", estiloDelArchivo,
		"{{JS}}", archivoJS,
		"{{DATOS}}", string(datos),
	)
	var b bytes.Buffer
	if _, err := r.WriteString(&b, archivoHTML); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
