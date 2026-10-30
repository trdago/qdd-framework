// Package modules gestiona las fichas de módulo de QDD: el "corazón" de cada
// módulo (objetivo operativo, objetivo tecnológico, invariantes y cómo se
// prueba) guardado de forma estándar en .qdd/project/modules/<id>.yaml, para
// que cualquier IA pueda seguir desarrollando el proyecto sin perder su intención.
package modules

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Dir es la carpeta de fichas, relativa a la raíz del proyecto.
const Dir = ".qdd/project/modules"

const (
	EstadoBorrador = "borrador"
	EstadoAprobado = "aprobado"
)

type ObjetivoOperativo struct {
	Problema        string `yaml:"problema"`
	Usuarios        string `yaml:"usuarios,omitempty"`
	CriterioDeExito string `yaml:"criterio_de_exito"`
	CasoReal        string `yaml:"caso_real,omitempty"`
}

type ObjetivoTecnologico struct {
	Arquitectura string   `yaml:"arquitectura"`
	Decisiones   []string `yaml:"decisiones,omitempty"`
	Invariantes  []string `yaml:"invariantes,omitempty"`
}

type Pruebas struct {
	Comandos   []string `yaml:"comandos"`
	Goldensets []string `yaml:"goldensets,omitempty"`
}

// Module es una ficha de módulo.
type Module struct {
	ID                  string              `yaml:"id"`
	Nombre              string              `yaml:"nombre"`
	Estado              string              `yaml:"estado"`
	Rutas               []string            `yaml:"rutas"`
	Blueprint           string              `yaml:"blueprint,omitempty"` // p. ej. "auth@1.2.0", si se generó desde un blueprint
	ObjetivoOperativo   ObjetivoOperativo   `yaml:"objetivo_operativo"`
	ObjetivoTecnologico ObjetivoTecnologico `yaml:"objetivo_tecnologico"`
	Pruebas             Pruebas             `yaml:"pruebas"`
	ADRs                []string            `yaml:"adrs,omitempty"`
	Findings            []string            `yaml:"findings,omitempty"`
	Fuentes             []string            `yaml:"fuentes,omitempty"`
	UltimaRevision      string              `yaml:"ultima_revision,omitempty"`
	RevisadoEnCommit    string              `yaml:"revisado_en_commit,omitempty"`

	File string `yaml:"-"`
}

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// Parse lee una ficha desde YAML.
func Parse(data []byte) (Module, error) {
	var m Module
	if err := yaml.Unmarshal(data, &m); err != nil {
		return Module{}, fmt.Errorf("YAML inválido: %w", err)
	}
	return m, nil
}

// Validate devuelve los problemas de la ficha; vacío significa ficha válida.
func (m Module) Validate(root string) []string {
	var problems []string
	if !idPattern.MatchString(m.ID) {
		problems = append(problems, "id: obligatorio, en minúsculas, solo letras, números, '-' y '_' (p. ej. 'validador-ofertas')")
	}
	if strings.TrimSpace(m.Nombre) == "" {
		problems = append(problems, "nombre: obligatorio")
	}
	if m.Estado != EstadoBorrador && m.Estado != EstadoAprobado {
		problems = append(problems, "estado: debe ser 'borrador' o 'aprobado'")
	}
	if strings.TrimSpace(m.ObjetivoOperativo.Problema) == "" {
		problems = append(problems, "objetivo_operativo.problema: obligatorio (qué problema de negocio u operación resuelve el módulo)")
	}
	if strings.TrimSpace(m.ObjetivoOperativo.CriterioDeExito) == "" {
		problems = append(problems, "objetivo_operativo.criterio_de_exito: obligatorio (cómo se sabe que el módulo cumple)")
	}
	if strings.TrimSpace(m.ObjetivoTecnologico.Arquitectura) == "" {
		problems = append(problems, "objetivo_tecnologico.arquitectura: obligatorio (cómo está construido y por qué)")
	}
	if len(m.Pruebas.Comandos) == 0 && len(m.Pruebas.Goldensets) == 0 {
		problems = append(problems, "pruebas: declara al menos un comando o un goldenset (cómo se verifica el módulo)")
	}
	return append(problems, m.validateRutas(root)...)
}

func (m Module) validateRutas(root string) []string {
	if len(m.Rutas) == 0 {
		return []string{"rutas: declara al menos una carpeta o archivo del módulo"}
	}
	var problems []string
	for _, r := range m.Rutas {
		clean := filepath.Clean(r)
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			problems = append(problems, fmt.Sprintf("rutas: '%s' debe ser relativa a la raíz del proyecto", r))
			continue
		}
		if _, err := os.Stat(filepath.Join(root, clean)); err != nil {
			problems = append(problems, fmt.Sprintf("rutas: '%s' no existe en el proyecto", r))
		}
	}
	return problems
}

// Load lee todas las fichas del proyecto. Devuelve también los errores de
// lectura o parseo (una ficha rota no impide leer las demás).
func Load(root string) ([]Module, []error) {
	files, _ := filepath.Glob(filepath.Join(root, Dir, "*.yaml"))
	sort.Strings(files)
	var mods []Module
	var errs []error
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", f, err))
			continue
		}
		m, err := Parse(data)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", f, err))
			continue
		}
		m.File = f
		mods = append(mods, m)
	}
	return mods, errs
}

// Save valida y escribe la ficha como BORRADOR: una IA puede proponer o
// actualizar el corazón de un módulo, pero solo un humano lo aprueba
// (qdd modules approve <id>).
func Save(root string, m Module) (string, []string) {
	m.Estado = EstadoBorrador
	m.UltimaRevision = ""
	m.RevisadoEnCommit = ""
	if problems := m.Validate(root); len(problems) > 0 {
		return "", problems
	}
	return write(root, m)
}

// TextoAprobada es el aviso de una ficha recién aprobada. Sin commits en el repositorio no hay commit que citar: se
// dice así en vez de dejar «(revisada en )».
func TextoAprobada(m Module) string {
	if m.RevisadoEnCommit == "" {
		return fmt.Sprintf("[+] Ficha '%s' aprobada (sin commit de referencia: el repositorio aún no tiene commits).", m.ID)
	}
	return fmt.Sprintf("[+] Ficha '%s' aprobada (revisada en %s).", m.ID, m.RevisadoEnCommit)
}

// Approve marca la ficha como aprobada y revisada en el commit actual.
func Approve(root, id string) (Module, []string) {
	m, ok := Find(root, id)
	if !ok {
		return Module{}, []string{fmt.Sprintf("no existe la ficha '%s' en %s", id, Dir)}
	}
	m.Estado = EstadoAprobado
	m.UltimaRevision = time.Now().Format("2006-01-02")
	m.RevisadoEnCommit = headCommit(root)
	if problems := m.Validate(root); len(problems) > 0 {
		return Module{}, problems
	}
	if _, problems := write(root, m); len(problems) > 0 {
		return Module{}, problems
	}
	return m, nil
}

// Find busca una ficha por id.
func Find(root, id string) (Module, bool) {
	mods, _ := Load(root)
	for _, m := range mods {
		if m.ID == id {
			return m, true
		}
	}
	return Module{}, false
}

func write(root string, m Module) (string, []string) {
	dir := filepath.Join(root, Dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", []string{err.Error()}
	}
	data, err := yaml.Marshal(m)
	if err != nil {
		return "", []string{err.Error()}
	}
	path := filepath.Join(dir, m.ID+".yaml")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", []string{err.Error()}
	}
	return path, nil
}

// Covering devuelve las fichas cuyas rutas contienen la ruta dada.
func Covering(mods []Module, path string) []Module {
	p := filepath.ToSlash(filepath.Clean(path))
	var out []Module
	for _, m := range mods {
		if m.covers(p) {
			out = append(out, m)
		}
	}
	return out
}

func (m Module) covers(p string) bool {
	for _, r := range m.Rutas {
		ruta := strings.TrimSuffix(filepath.ToSlash(filepath.Clean(r)), "/")
		if p == ruta || strings.HasPrefix(p, ruta+"/") {
			return true
		}
	}
	return false
}

// Drift cuenta los commits que tocaron las rutas del módulo después de su
// última revisión aprobada. Devuelve -1 si la ficha nunca fue revisada o si
// no se puede calcular (sin git, commit inexistente).
func Drift(root string, m Module) int {
	if m.RevisadoEnCommit == "" {
		return -1
	}
	args := append([]string{"rev-list", "--count", m.RevisadoEnCommit + "..HEAD", "--"}, m.Rutas...)
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return -1
	}
	var n int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &n); err != nil {
		return -1
	}
	return n
}

func headCommit(root string) string {
	cmd := exec.Command("git", "rev-parse", "--short=12", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
