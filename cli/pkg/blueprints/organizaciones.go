package blueprints

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// organizacionesFile es el registro central: qué repositorio de blueprints usa
// cada organización y qué proyectos le pertenecen, según su remote git. Así
// todos los proyectos de una organización ven sus blueprints sin configurar
// nada. Lo administra QDD; no se edita a mano.
const organizacionesFile = "organizaciones.yaml"

// Organizacion asocia un repositorio de blueprints con los proyectos cuyo
// remote git coincide con alguno de sus patrones (p. ej. github.com/acme/*).
type Organizacion struct {
	Origen    string   `yaml:"origen"`
	Proyectos []string `yaml:"proyectos"`
}

type registroCentral struct {
	Organizaciones []Organizacion `yaml:"organizaciones"`
}

func organizacionesPath() (string, error) {
	dir, err := centralDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, organizacionesFile), nil
}

// LoadOrganizaciones lee el registro central; vacío si no hay.
func LoadOrganizaciones() ([]Organizacion, error) {
	p, err := organizacionesPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var r registroCentral
	if err := yaml.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("%s: YAML inválido: %w", p, err)
	}
	return r.Organizaciones, nil
}

func saveOrganizaciones(orgs []Organizacion) error {
	p, err := organizacionesPath()
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(registroCentral{Organizaciones: orgs})
	if err != nil {
		return err
	}
	header := "# Registro central de blueprints de QDD (RFC 0004). Lo administra QDD: no se edita a mano.\n"
	return os.WriteFile(p, append([]byte(header), data...), 0600)
}

// RegistrarOrganizacion asocia los proyectos del mismo dueño que el de root
// (su remote git sin el último segmento) con el repositorio de blueprints. Un
// patrón pertenece a una sola organización: la última que se registró.
func RegistrarOrganizacion(root string, reg Registro) error {
	patron := patronProyecto(root)
	if patron == "" || reg.Origen == "" {
		return nil
	}
	orgs, err := LoadOrganizaciones()
	if err != nil {
		return err
	}
	var out []Organizacion
	registrada := false
	for _, o := range orgs {
		o.Proyectos = sinPatron(o.Proyectos, patron)
		if o.Origen == reg.Origen {
			o.Proyectos = append(o.Proyectos, patron)
			registrada = true
		}
		if len(o.Proyectos) == 0 {
			continue
		}
		out = append(out, o)
	}
	if !registrada {
		out = append(out, Organizacion{Origen: reg.Origen, Proyectos: []string{patron}})
	}
	return saveOrganizaciones(out)
}

func sinPatron(patrones []string, patron string) []string {
	var out []string
	for _, p := range patrones {
		if p == patron {
			continue
		}
		out = append(out, p)
	}
	return out
}

// OrganizacionDe busca la organización del proyecto por su remote git.
func OrganizacionDe(root string) (Organizacion, bool) {
	remote := normalizarRemote(remoteDe(root))
	if remote == "" {
		return Organizacion{}, false
	}
	orgs, _ := LoadOrganizaciones()
	for _, o := range orgs {
		for _, p := range o.Proyectos {
			if ok, _ := path.Match(p, remote); ok {
				return o, true
			}
		}
	}
	return Organizacion{}, false
}

func remoteDe(root string) string {
	remote, err := git(root, "remote", "get-url", "origin")
	if err != nil {
		return ""
	}
	return remote
}

// patronProyecto es el patrón de los proyectos del mismo dueño:
// git@github.com:acme/tienda.git → github.com/acme/*.
func patronProyecto(root string) string {
	remote := normalizarRemote(remoteDe(root))
	i := strings.LastIndex(remote, "/")
	if i <= 0 || !strings.Contains(remote[:i], "/") {
		return ""
	}
	return remote[:i] + "/*"
}

// normalizarRemote lleva un remote git a host/dueño/repo, sin usuario,
// puerto, esquema ni ".git". Una ruta local no tiene dueño: devuelve "".
func normalizarRemote(u string) string {
	u = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSpace(u), "/"), ".git")
	if i := strings.Index(u, "://"); i >= 0 {
		host, rest, _ := strings.Cut(u[i+3:], "/")
		return unirRemote(host, rest)
	}
	host, rest, ok := strings.Cut(u, ":")
	if !ok || strings.ContainsAny(host, "/\\") {
		return ""
	}
	return unirRemote(host, rest)
}

func unirRemote(host, rest string) string {
	host = host[strings.LastIndex(host, "@")+1:]
	host, _, _ = strings.Cut(host, ":")
	rest = strings.Trim(rest, "/")
	if host == "" || rest == "" {
		return ""
	}
	return strings.ToLower(host) + "/" + rest
}
