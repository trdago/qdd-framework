// Package ambientes es la fase 3 del baúl (RFC 0023): .qdd/ambientes.yaml declara, por ambiente y sin valores
// secretos, lo que cada uno necesita, y 'qdd ambiente levantar' compara eso con lo que existe. Empieza por el DNS de
// Cloudflare y solo arma el plan: no cambia nada (aplicar llega en otra versión, siempre tras el clic de la persona).
package ambientes

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/qdd-framework/qdd/pkg/baul"
	"gopkg.in/yaml.v3"
)

// Rule: Prohibido usar 'else'

const (
	// Ruta es el archivo, versionado con el proyecto.
	Ruta = ".qdd/ambientes.yaml"
	// Formato es la versión del archivo que este qdd entiende.
	Formato = 1
)

// TiposDNS son los registros que se pueden declarar.
var TiposDNS = []string{"A", "AAAA", "CNAME", "TXT", "MX"}

// Archivo es .qdd/ambientes.yaml.
type Archivo struct {
	Formato   int                 `yaml:"formato"`
	Ambientes map[string]Ambiente `yaml:"ambientes"`
}

// Ambiente es lo que declara un ambiente; por ahora, su DNS.
type Ambiente struct {
	DNS DNS `yaml:"dns"`
}

// DNS es la zona de Cloudflare y los registros que el ambiente espera en ella.
type DNS struct {
	Zona      string     `yaml:"zona"`
	Registros []Registro `yaml:"registros"`
}

// Registro es un registro DNS; Nombre queda completo (app.ejemplo.cl), como lo devuelve Cloudflare.
type Registro struct {
	Tipo    string `yaml:"tipo" json:"tipo"`
	Nombre  string `yaml:"nombre" json:"nombre"`
	Valor   string `yaml:"valor" json:"valor"`
	Proxied bool   `yaml:"proxied,omitempty" json:"proxied,omitempty"`
}

// Cargar lee y valida el archivo; uno que no se entiende es un error que dice qué arreglar.
func Cargar(root string) (Archivo, error) {
	data, err := os.ReadFile(filepath.Join(root, Ruta))
	if errors.Is(err, os.ErrNotExist) {
		return Archivo{}, fmt.Errorf("no hay %s: decláralo con la zona de Cloudflare y los registros de cada ambiente (RFC 0023)", Ruta)
	}
	if err != nil {
		return Archivo{}, err
	}
	var a Archivo
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&a); err != nil {
		return Archivo{}, fmt.Errorf("%s ilegible: %w", Ruta, err)
	}
	if a.Formato > Formato || a.Formato < 1 {
		return Archivo{}, fmt.Errorf("%s es formato %d y este qdd entiende el %d: actualiza qdd", Ruta, a.Formato, Formato)
	}
	for nombre, amb := range a.Ambientes {
		normal, err := validar(nombre, amb)
		if err != nil {
			return Archivo{}, fmt.Errorf("%s, ambiente %s: %w", Ruta, nombre, err)
		}
		a.Ambientes[nombre] = normal
	}
	return a, nil
}

func validar(nombre string, amb Ambiente) (Ambiente, error) {
	if !slices.Contains(baul.Ambientes, nombre) {
		return amb, fmt.Errorf("no es un ambiente del baúl (%s)", strings.Join(baul.Ambientes, ", "))
	}
	zona := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(amb.DNS.Zona)), ".")
	if zona == "" && len(amb.DNS.Registros) > 0 {
		return amb, errors.New("dns.zona vacía")
	}
	amb.DNS.Zona = zona
	vistos := map[string]bool{}
	for i, r := range amb.DNS.Registros {
		r.Tipo = strings.ToUpper(strings.TrimSpace(r.Tipo))
		r.Nombre = completo(r.Nombre, zona)
		if !slices.Contains(TiposDNS, r.Tipo) {
			return amb, fmt.Errorf("registro %d: tipo %q (se puede %s)", i+1, r.Tipo, strings.Join(TiposDNS, ", "))
		}
		if strings.TrimSpace(r.Valor) == "" {
			return amb, fmt.Errorf("registro %d (%s %s): falta el valor", i+1, r.Tipo, r.Nombre)
		}
		clave := r.Tipo + " " + r.Nombre
		if vistos[clave] {
			return amb, fmt.Errorf("registro %s repetido: uno por tipo y nombre", clave)
		}
		vistos[clave] = true
		amb.DNS.Registros[i] = r
	}
	return amb, nil
}

// completo deja el nombre como lo devuelve Cloudflare: «@» es la zona y uno corto lleva la zona al final.
func completo(nombre, zona string) string {
	n := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(nombre)), ".")
	if n == "@" || n == "" || n == zona {
		return zona
	}
	if strings.HasSuffix(n, "."+zona) {
		return n
	}
	return n + "." + zona
}
