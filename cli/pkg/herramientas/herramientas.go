// Package herramientas guarda los ejecutables del proyecto como pestañas del panel (RFC 0011). La IA arma una vez la
// definición de cada uno (sus acciones y los campos que piden), QDD la valida y la prueba, y desde ahí el panel la
// dibuja y la corre sin IA. Se versiona en .qdd/herramientas/<id>.yaml; los secretos nunca van ahí, solo el nombre
// de la variable de entorno que los trae.
package herramientas

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Rule: Prohibido usar 'else'

// Dir es donde el proyecto versiona las definiciones.
const Dir = ".qdd/herramientas"

// Modos de una acción.
const (
	UnaVez = "una-vez" // corre, espera y muestra su salida
	EnVivo = "en-vivo" // muestra la salida mientras corre, hasta que la persona la detiene
)

// Tipos de campo.
var tipos = []string{"texto", "numero", "opcion", "si-no"}

// Herramienta es la definición de un ejecutable del proyecto.
type Herramienta struct {
	ID          string   `yaml:"id" json:"id"`
	Nombre      string   `yaml:"nombre" json:"nombre"`
	Descripcion string   `yaml:"descripcion,omitempty" json:"descripcion,omitempty"`
	Comando     []string `yaml:"comando" json:"comando"`                     // argv base, sin shell; relativo a la raíz del proyecto
	Entorno     []string `yaml:"entorno,omitempty" json:"entorno,omitempty"` // nombres de las variables que necesita, nunca sus valores
	Acciones    []Accion `yaml:"acciones" json:"acciones"`
	Prueba      *Prueba  `yaml:"prueba" json:"-"`
}

// Accion es un botón de la pestaña.
type Accion struct {
	ID        string  `yaml:"id" json:"id"`
	Nombre    string  `yaml:"nombre" json:"nombre"`
	Args      []Arg   `yaml:"args,omitempty" json:"-"`
	Campos    []Campo `yaml:"campos,omitempty" json:"campos,omitempty"`
	Modo      string  `yaml:"modo,omitempty" json:"modo,omitempty"` // una-vez (por defecto) o en-vivo
	Confirmar bool    `yaml:"confirmar,omitempty" json:"confirmar,omitempty"`
}

// Campo es lo que la persona completa antes de correr una acción.
type Campo struct {
	ID         string   `yaml:"id" json:"id"`
	Nombre     string   `yaml:"nombre,omitempty" json:"nombre,omitempty"`
	Tipo       string   `yaml:"tipo" json:"tipo"`
	Requerido  bool     `yaml:"requerido,omitempty" json:"requerido,omitempty"`
	Opciones   []string `yaml:"opciones,omitempty" json:"opciones,omitempty"`
	PorDefecto string   `yaml:"por_defecto,omitempty" json:"por_defecto,omitempty"`
}

// Prueba es la prueba de humo: el comando con estos argumentos debe salir con 0.
type Prueba struct {
	Args []string `yaml:"args" json:"args"`
}

// Arg es un argumento, o un grupo que va entero o no va (p. ej. [--chat, "{{chat}}"]: solo si chat tiene valor).
type Arg struct {
	Partes []string
	Grupo  bool
}

func (a *Arg) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		a.Partes = []string{n.Value}
		return nil
	}
	if n.Kind != yaml.SequenceNode {
		return fmt.Errorf("línea %d: un argumento es un texto o un grupo [..]", n.Line)
	}
	a.Grupo = true
	return n.Decode(&a.Partes)
}

func (a Arg) MarshalYAML() (interface{}, error) {
	if a.Grupo {
		return a.Partes, nil
	}
	return strings.Join(a.Partes, ""), nil
}

var (
	idPattern       = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	variablePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	campoPattern    = regexp.MustCompile(`\{\{\s*([^{}\s]+)\s*\}\}`)
	campoEntero     = regexp.MustCompile(`^\{\{\s*([^{}\s]+)\s*\}\}$`)
)

// Parse lee una definición; un campo que no existe es un error (p. ej. un «valores:» con un secreto).
func Parse(data []byte) (Herramienta, error) {
	var h Herramienta
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&h); err != nil {
		return Herramienta{}, fmt.Errorf("la definición no se lee: %v", err)
	}
	return h, nil
}

// Archivo es la ruta, relativa a la raíz, de la definición de id.
func Archivo(id string) string {
	return filepath.Join(Dir, id+".yaml")
}

// Cargar lee las definiciones del proyecto, ordenadas por id.
func Cargar(root string) ([]Herramienta, []error) {
	files, _ := filepath.Glob(filepath.Join(root, Dir, "*.yaml"))
	sort.Strings(files)
	var out []Herramienta
	var errs []error
	for _, f := range files {
		h, err := cargarArchivo(f)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %v", filepath.ToSlash(filepath.Join(Dir, filepath.Base(f))), err))
			continue
		}
		out = append(out, h)
	}
	return out, errs
}

func cargarArchivo(f string) (Herramienta, error) {
	data, err := os.ReadFile(f)
	if err != nil {
		return Herramienta{}, err
	}
	h, err := Parse(data)
	if err != nil {
		return Herramienta{}, err
	}
	if h.ID+".yaml" != filepath.Base(f) {
		return Herramienta{}, fmt.Errorf("su id es «%s»: el archivo debe llamarse %s.yaml", h.ID, h.ID)
	}
	return h, nil
}

// Buscar devuelve la herramienta guardada con ese id.
func Buscar(root, id string) (Herramienta, error) {
	if !idPattern.MatchString(id) {
		return Herramienta{}, fmt.Errorf("«%s» no es un id de herramienta", id)
	}
	h, err := cargarArchivo(filepath.Join(root, Archivo(id)))
	if os.IsNotExist(err) {
		return Herramienta{}, fmt.Errorf("no hay una herramienta «%s» (qdd herramienta listar)", id)
	}
	return h, err
}

// Guardar valida la definición, corre su prueba de humo y, si todo pasa, la guarda tal como vino (con sus
// comentarios). Devuelve la ruta relativa o, si no se guarda, por qué.
func Guardar(root string, data []byte) (string, []string) {
	h, err := Parse(data)
	if err != nil {
		return "", []string{err.Error()}
	}
	if problemas := append(Validar(h), validarComando(root, h.Comando)...); len(problemas) > 0 {
		return "", problemas
	}
	if err := PruebaDeHumo(root, h); err != nil {
		return "", []string{err.Error()}
	}
	rel := Archivo(h.ID)
	if err := os.MkdirAll(filepath.Join(root, Dir), 0755); err != nil {
		return "", []string{err.Error()}
	}
	if err := os.WriteFile(filepath.Join(root, rel), data, 0644); err != nil {
		return "", []string{err.Error()}
	}
	return rel, nil
}

// Aviso es lo que impide usarla en esta máquina (el ejecutable no está o falta una variable), o "" si está lista.
func Aviso(root string, h Herramienta) string {
	if p := validarComando(root, h.Comando); len(p) > 0 {
		return p[0]
	}
	if err := entornoCompleto(h); err != nil {
		return err.Error()
	}
	return ""
}

// Quitar borra la definición de id.
func Quitar(root, id string) error {
	if _, err := Buscar(root, id); err != nil {
		return err
	}
	return os.Remove(filepath.Join(root, Archivo(id)))
}

// Validar revisa la definición sin correr nada: lo que el gate exige (HER-01).
func Validar(h Herramienta) []string {
	var p []string
	if !idPattern.MatchString(h.ID) {
		p = append(p, fmt.Sprintf("id «%s»: minúsculas, números y guiones", h.ID))
	}
	if strings.TrimSpace(h.Nombre) == "" {
		p = append(p, "falta el nombre (lo que dice la pestaña)")
	}
	if len(h.Comando) == 0 || strings.TrimSpace(h.Comando[0]) == "" {
		p = append(p, "falta el comando")
	}
	for _, v := range h.Entorno {
		if !variablePattern.MatchString(v) {
			p = append(p, fmt.Sprintf("entorno: «%s» no es el nombre de una variable (solo el nombre, nunca su valor)", v))
		}
	}
	if len(h.Acciones) == 0 {
		p = append(p, "no tiene acciones: cada una es un botón de la pestaña")
	}
	if h.Prueba == nil {
		p = append(p, "falta la prueba de humo (prueba: {args: [--help]}): sin ella no se sabe si funciona")
	}
	p = append(p, secretos("comando", h.Comando)...)
	if h.Prueba != nil {
		p = append(p, secretos("prueba", h.Prueba.Args)...)
	}
	vistas := map[string]bool{}
	for _, a := range h.Acciones {
		if vistas[a.ID] {
			p = append(p, fmt.Sprintf("la acción «%s» está dos veces", a.ID))
		}
		vistas[a.ID] = true
		p = append(p, validarAccion(a)...)
	}
	return p
}

func validarAccion(a Accion) []string {
	var p []string
	if !idPattern.MatchString(a.ID) {
		p = append(p, fmt.Sprintf("acción «%s»: el id es de minúsculas, números y guiones", a.ID))
	}
	if strings.TrimSpace(a.Nombre) == "" {
		p = append(p, fmt.Sprintf("acción %s: falta su nombre (lo que dice el botón)", a.ID))
	}
	if a.Modo != "" && a.Modo != UnaVez && a.Modo != EnVivo {
		p = append(p, fmt.Sprintf("acción %s: el modo «%s» no existe (una-vez o en-vivo)", a.ID, a.Modo))
	}
	campos := map[string]Campo{}
	for _, c := range a.Campos {
		campos[c.ID] = c
		p = append(p, validarCampo(a.ID, c)...)
	}
	usados := map[string]bool{}
	var partes []string
	for _, arg := range a.Args {
		partes = append(partes, arg.Partes...)
		for _, parte := range arg.Partes {
			p = append(p, validarParte(a.ID, parte, arg.Grupo, campos, usados)...)
		}
	}
	p = append(p, secretos("acción "+a.ID, partes)...)
	for _, c := range a.Campos {
		if !usados[c.ID] {
			p = append(p, fmt.Sprintf("acción %s: el campo «%s» no se usa en args", a.ID, c.ID))
		}
	}
	return p
}

func validarParte(accion, parte string, enGrupo bool, campos map[string]Campo, usados map[string]bool) []string {
	if !strings.Contains(parte, "{{") {
		return nil
	}
	m := campoEntero.FindStringSubmatch(parte)
	if m == nil {
		return []string{fmt.Sprintf("acción %s: «%s»: un campo es un argumento entero, solo («{{campo}}»), para que lo escrito nunca se mezcle con otro argumento", accion, parte)}
	}
	c, ok := campos[m[1]]
	if !ok {
		return []string{fmt.Sprintf("acción %s: «{{%s}}» no está en sus campos", accion, m[1])}
	}
	usados[c.ID] = true
	if c.Tipo == "si-no" && !enGrupo {
		return []string{fmt.Sprintf("acción %s: el si-no «%s» va en un grupo (p. ej. [--%s, \"{{%s}}\"]): decide si el grupo va", accion, c.ID, c.ID, c.ID)}
	}
	return nil
}

func validarCampo(accion string, c Campo) []string {
	var p []string
	if !idPattern.MatchString(strings.ReplaceAll(c.ID, "_", "-")) {
		p = append(p, fmt.Sprintf("acción %s: el campo «%s» necesita un id de minúsculas, números, guiones o «_»", accion, c.ID))
	}
	if !inList(tipos, c.Tipo) {
		p = append(p, fmt.Sprintf("acción %s: el campo %s es de un tipo que no existe («%s»; %s)", accion, c.ID, c.Tipo, strings.Join(tipos, ", ")))
	}
	if c.Tipo == "opcion" && len(c.Opciones) == 0 {
		p = append(p, fmt.Sprintf("acción %s: el campo %s es una opción sin opciones", accion, c.ID))
	}
	if c.PorDefecto != "" {
		if err := valorValido(c, c.PorDefecto); err != nil {
			p = append(p, fmt.Sprintf("acción %s: por_defecto de %s: %v", accion, c.ID, err))
		}
	}
	return p
}

var (
	// un secreto con su nombre: --token=…, password=…
	secretoConNombre = regexp.MustCompile(`(?i)(token|secret|secreto|password|passwd|clave|api[_-]?key|apikey)[^=\s]*=\S+`)
	// una bandera de secreto: --token, --password; lo que la sigue es su valor
	banderaDeSecreto = regexp.MustCompile(`(?i)^--?[a-z-]*(token|secret|secreto|password|passwd|clave|api-?key|apikey)$`)
	// formatos conocidos de credenciales
	credencial = regexp.MustCompile(`(ghp_|gho_|ghs_|github_pat_|glpat-|xox[abpr]-|npm_[A-Za-z0-9]{30,}|AKIA[0-9A-Z]{16}|sk-[A-Za-z0-9_-]{16,})`)
)

// secretos encuentra valores de secretos escritos en la definición; nunca los repite en el mensaje.
func secretos(donde string, partes []string) []string {
	for i, parte := range partes {
		if campoPattern.MatchString(parte) {
			continue
		}
		siguienteLiteral := i+1 < len(partes) && !campoPattern.MatchString(partes[i+1])
		if secretoConNombre.MatchString(parte) || credencial.MatchString(parte) || (banderaDeSecreto.MatchString(parte) && siguienteLiteral) {
			return []string{fmt.Sprintf("%s: trae un secreto. Los secretos van en una variable de entorno (entorno: [NOMBRE]) o en un campo, nunca escritos en la definición", donde)}
		}
	}
	return nil
}

func inList(lista []string, v string) bool {
	for _, x := range lista {
		if x == v {
			return true
		}
	}
	return false
}
