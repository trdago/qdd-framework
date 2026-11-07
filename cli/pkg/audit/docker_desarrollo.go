package audit

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Rule: Prohibido usar 'else'

// ReglaAppEnDocker avisa que el proyecto levanta la app en Docker para desarrollar. Decisión de la persona (2026-11-07): en
// desarrollo Docker es solo para la base de datos, y quien igual lo quiera para la app puede usarlo sabiendo que es más
// lento. Por eso nunca bloquea. El porqué, con sus fuentes, está en docs/docker-en-desarrollo.md.
const ReglaAppEnDocker = "DEV-01-APP-EN-DOCKER"

const porQueDockerEsLento = "En desarrollo es más lento: cada cambio tiene que cruzar hasta el contenedor, la recarga automática puede dejar de enterarse y a veces hay que reconstruir la imagen. Docker queda para la base de datos; la app corre directo en el PC. Puedes dejarlo así si lo necesitas. Por qué: https://github.com/trdago/qdd-framework/blob/main/docs/docker-en-desarrollo.md"

var (
	archivoCompose = regexp.MustCompile(`(?i)^(docker-)?compose(\.[\w-]+)*\.ya?ml$`)
	// composeDeOtroAmbiente: un compose de despliegue o de pruebas no es el desarrollo diario.
	composeDeOtroAmbiente = regexp.MustCompile(`(?i)\.(prod|production|stage|staging|deploy|ci|test|e2e)\b`)
	composeDeDesarrollo   = regexp.MustCompile(`(?i)\.(dev|develop|development|local)\b`)
	carpetaDeDespliegue   = regexp.MustCompile(`(?i)(^|/)(deploy|deployment|deployments)/`)
	imagenDeDatos         = regexp.MustCompile(`(?i)^(postgres|postgis|pgvector|timescale|mysql|mariadb|mongo|redis|valkey|memcached|elasticsearch|opensearch|cassandra|clickhouse|neo4j|influxdb|cockroach|minio)`)
	nombreDeDatos         = regexp.MustCompile(`(?i)^(db|database|base-?de-?datos)$`)
)

// carpetasSinDesarrollo no se recorren: dependencias, salidas y pruebas, que pueden traer su propio compose.
var carpetasSinDesarrollo = map[string]bool{
	"node_modules": true, "vendor": true, "dist": true, "build": true, "target": true, "tmp": true,
	"e2e": true, "test": true, "tests": true, "testdata": true, "fixtures": true,
}

type composeDeProyecto struct {
	Services map[string]servicioCompose `yaml:"services"`
}

type servicioCompose struct {
	Image   string      `yaml:"image"`
	Build   yaml.Node   `yaml:"build"`
	Volumes []yaml.Node `yaml:"volumes"`
}

// RunDockerDesarrolloCheck da un solo aviso por proyecto, con todo lo que encontró: es una sola decisión, aunque esté
// repartida en varios archivos.
func RunDockerDesarrolloCheck(cwd string) []Violation {
	var encontrados []string
	_ = filepath.WalkDir(cwd, func(ruta string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return saltarCarpeta(ruta, cwd, d.Name())
		}
		relativa, errRel := filepath.Rel(cwd, ruta)
		if errRel != nil {
			return nil
		}
		encontrados = append(encontrados, appEnDockerDe(ruta, filepath.ToSlash(relativa))...)
		return nil
	})
	if len(encontrados) == 0 {
		return nil
	}
	descripcion := "La app corre en Docker: " + strings.Join(encontrados, "; ") + ". " + porQueDockerEsLento
	return []Violation{{Category: "DESARROLLO", RuleID: ReglaAppEnDocker, Description: descripcion, Recomendacion: true}}
}

func saltarCarpeta(ruta, cwd, nombre string) error {
	if ruta == cwd || nombre == ".devcontainer" {
		return nil
	}
	if strings.HasPrefix(nombre, ".") || carpetasSinDesarrollo[nombre] {
		return filepath.SkipDir
	}
	return nil
}

func appEnDockerDe(ruta, relativa string) []string {
	if relativa == ".devcontainer.json" || relativa == ".devcontainer/devcontainer.json" {
		return []string{relativa}
	}
	nombre := filepath.Base(ruta)
	if !archivoCompose.MatchString(nombre) || composeDeOtroAmbiente.MatchString(nombre) {
		return nil
	}
	if carpetaDeDespliegue.MatchString(relativa) && !composeDeDesarrollo.MatchString(nombre) {
		return nil
	}
	servicios := serviciosDeApp(ruta)
	if len(servicios) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("%s (%s)", relativa, strings.Join(servicios, ", "))}
}

// serviciosDeApp devuelve, en orden, los servicios que no son datos y que construyen o montan el código del proyecto. Un
// compose ilegible no se adivina: lo que no se puede leer no se informa.
func serviciosDeApp(ruta string) []string {
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		return nil
	}
	var compose composeDeProyecto
	if yaml.Unmarshal(contenido, &compose) != nil {
		return nil
	}
	var app []string
	for nombre, s := range compose.Services {
		if esServicioDeDatos(nombre, s.Image) {
			continue
		}
		if !s.Build.IsZero() || montaElCodigo(s.Volumes) {
			app = append(app, nombre)
		}
	}
	sort.Strings(app)
	return app
}

func esServicioDeDatos(nombre, imagen string) bool {
	if imagen == "" {
		return imagenDeDatos.MatchString(nombre) || nombreDeDatos.MatchString(nombre)
	}
	return imagenDeDatos.MatchString(imagen[strings.LastIndex(imagen, "/")+1:])
}

func montaElCodigo(volumenes []yaml.Node) bool {
	for i := range volumenes {
		if esRutaDelProyecto(origenDeVolumen(&volumenes[i])) {
			return true
		}
	}
	return false
}

// origenDeVolumen lee las dos formas de compose: «./src:/app» y {type: bind, source: ./src, target: /app}.
func origenDeVolumen(v *yaml.Node) string {
	if v.Kind == yaml.ScalarNode {
		origen, _, _ := strings.Cut(v.Value, ":")
		return origen
	}
	var largo struct {
		Source string `yaml:"source"`
	}
	if v.Decode(&largo) != nil {
		return ""
	}
	return largo.Source
}

func esRutaDelProyecto(origen string) bool {
	return strings.HasPrefix(origen, ".") || strings.HasPrefix(origen, "${PWD") || strings.HasPrefix(origen, "$PWD")
}
