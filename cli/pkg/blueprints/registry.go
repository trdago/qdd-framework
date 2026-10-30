package blueprints

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// configFile es el archivo donde QDD recuerda el repositorio de blueprints de
// la organización (clave 'blueprints').
const configFile = ".qdd/config.yaml"

// Registro es el repositorio de blueprints de la organización. QDD lo guarda
// en el proyecto y en el registro central la primera vez que se importa desde
// él; después basta con nombrar los blueprints (qdd blueprint add auth).
type Registro struct {
	Origen string `yaml:"origen"`
	// Ref es del esquema anterior: hoy se importa siempre la última versión
	// publicada, salvo que se pida otra con --ref. Se lee para no romper
	// config.yaml existentes y se borra al volver a guardar el registro.
	Ref string `yaml:"ref,omitempty"`
	// Central indica que el registro viene de la organización del proyecto
	// (registro central de QDD) y no de su .qdd/config.yaml.
	Central bool `yaml:"-"`
}

// LoadRegistro devuelve el repositorio de blueprints del proyecto: el de su
// .qdd/config.yaml o, si no tiene, el de su organización según su remote git.
func LoadRegistro(root string) (Registro, error) {
	reg, err := registroDelProyecto(root)
	if err != nil || reg.Origen != "" {
		return reg, err
	}
	org, ok := OrganizacionDe(root)
	if !ok {
		return Registro{}, nil
	}
	return Registro{Origen: org.Origen, Central: true}, nil
}

func registroDelProyecto(root string) (Registro, error) {
	data, err := os.ReadFile(filepath.Join(root, configFile))
	if os.IsNotExist(err) {
		return Registro{}, nil
	}
	if err != nil {
		return Registro{}, err
	}
	var cfg struct {
		Blueprints Registro `yaml:"blueprints"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Registro{}, fmt.Errorf("%s: YAML inválido: %w", configFile, err)
	}
	return cfg.Blueprints, nil
}

// SaveRegistro escribe la clave 'blueprints' de .qdd/config.yaml conservando
// el resto del archivo (claves, orden y comentarios).
func SaveRegistro(root string, r Registro) error {
	path := filepath.Join(root, configFile)
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("%s: YAML inválido: %w", configFile, err)
	}
	if len(doc.Content) == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}}}
	}
	mapping := doc.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return fmt.Errorf("%s: se esperaba un mapa de claves en la raíz", configFile)
	}
	var value yaml.Node
	if err := value.Encode(r); err != nil {
		return err
	}
	setKey(mapping, "blueprints", &value)
	out, err := yaml.Marshal(&doc)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, out, 0644)
}

func setKey(mapping *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value != key {
			continue
		}
		value.HeadComment = mapping.Content[i+1].HeadComment
		mapping.Content[i+1] = value
		return
	}
	k := &yaml.Node{Kind: yaml.ScalarNode, Value: key, HeadComment: "Repositorio de blueprints de la organización (lo administra 'qdd blueprint add')."}
	mapping.Content = append(mapping.Content, k, value)
}

// EsOrigen distingue un origen (carpeta o repositorio) de un id de blueprint:
// los ids no llevan '/', ':' ni '\', ni empiezan con '.' o '~'.
func EsOrigen(arg string) bool {
	return strings.ContainsAny(arg, "/:\\") || strings.HasPrefix(arg, ".") || strings.HasPrefix(arg, "~")
}

// AddResult describe lo que hizo AddManaged.
type AddResult struct {
	Entries  []LockEntry
	Registro Registro // registro vigente tras importar
	Guardado bool     // si se guardó o actualizó el registro en config.yaml
}

// AddManaged importa como Add, pero administra las fuentes: sin origen, cada
// id ya importado se actualiza desde la fuente que dice el lockfile y los
// nuevos salen del repositorio del proyecto o de su organización (FND-065).
// La primera vez que se importa desde un repositorio de varios blueprints lo
// guarda en el proyecto y en el registro central, para que este y los demás
// proyectos de la organización solo tengan que nombrarlos. Si una fuente
// falla, el lockfile queda como estaba.
func AddManaged(in AddInput) (AddResult, []string) {
	reg, err := LoadRegistro(in.Root)
	if err != nil {
		return AddResult{}, []string{err.Error()}
	}
	if strings.TrimSpace(in.Origen) != "" {
		return addManagedDesde(in, reg)
	}
	grupos, problems := porFuente(in, reg)
	if len(problems) > 0 {
		return AddResult{}, problems
	}
	previo, err := os.ReadFile(filepath.Join(in.Root, LockFile))
	if err != nil && !os.IsNotExist(err) {
		return AddResult{}, []string{err.Error()}
	}
	res := AddResult{Registro: reg}
	var warnings []string
	for _, g := range grupos {
		r, w := addManagedDesde(g, reg)
		if len(r.Entries) == 0 && len(w) > 0 {
			return AddResult{}, append(w, restaurarLockfile(in.Root, previo)...)
		}
		res.Entries = append(res.Entries, r.Entries...)
		warnings = append(warnings, w...)
		if r.Guardado {
			res.Registro, res.Guardado = r.Registro, true
		}
	}
	return res, warnings
}

// porFuente reparte lo pedido entre las fuentes: un id del lockfile va a su
// fuente; un id nuevo, un --paquete o --todos, al repositorio registrado.
func porFuente(in AddInput, reg Registro) ([]AddInput, []string) {
	lock, err := LoadLock(in.Root)
	if err != nil {
		return nil, []string{err.Error()}
	}
	var grupos []AddInput
	agregar := func(origen, id string) {
		for i := range grupos {
			if grupos[i].Origen == origen && id != "" {
				grupos[i].Blueprints = append(grupos[i].Blueprints, id)
				return
			}
		}
		g := in
		g.Origen, g.Blueprints = origen, nil
		if id != "" {
			g.Blueprints = []string{id}
		}
		grupos = append(grupos, g)
	}
	var nuevos []string
	for _, id := range in.Blueprints {
		e, ok := lock.Entry(id)
		if !ok {
			nuevos = append(nuevos, id)
			continue
		}
		if e.Subdir == "" {
			agregar(origenDeEntrada(e), "") // la fuente es en sí el blueprint
			continue
		}
		agregar(e.Origen, id)
	}
	if len(nuevos) > 0 || in.Paquete != "" || in.Todos || len(in.Blueprints) == 0 {
		if reg.Origen == "" {
			return nil, []string{"todavía no hay repositorio de blueprints de la organización: indícalo una vez y QDD lo recordará para todos sus proyectos (qdd blueprint add git@github.com:org/qdd-blueprints.git auth)"}
		}
		g := in
		g.Origen, g.Blueprints = reg.Origen, nuevos
		g, grupos = unirAlRegistro(g, grupos)
		grupos = append([]AddInput{g}, grupos...)
	}
	if in.Ref != "" && len(grupos) > 1 {
		return nil, []string{"--ref es una versión de una sola fuente, y lo pedido viene de varias: impórtalos por separado"}
	}
	return grupos, nil
}

// unirAlRegistro suma a reg los ids de grupos cuya fuente es el repositorio
// registrado (un solo Add por fuente) y devuelve los demás grupos.
func unirAlRegistro(reg AddInput, grupos []AddInput) (AddInput, []AddInput) {
	repo, _, _ := resolverOrigen(reg.Origen)
	var otros []AddInput
	for _, g := range grupos {
		if g.Origen != repo || len(g.Blueprints) == 0 || reg.Paquete != "" || reg.Todos {
			otros = append(otros, g)
			continue
		}
		reg.Blueprints = append(reg.Blueprints, g.Blueprints...)
	}
	return reg, otros
}

// origenDeEntrada es el origen con el que se vuelve a importar una entrada
// del lockfile: el repositorio y, si la tiene, su subcarpeta.
func origenDeEntrada(e LockEntry) string {
	if e.Subdir == "" {
		return e.Origen
	}
	if exists(e.Origen) {
		return filepath.Join(e.Origen, filepath.FromSlash(e.Subdir))
	}
	return e.Origen + "//" + e.Subdir
}

func restaurarLockfile(root string, previo []byte) []string {
	path := filepath.Join(root, LockFile)
	err := os.Remove(path)
	if previo != nil {
		err = os.WriteFile(path, previo, 0644)
	}
	if err != nil && !os.IsNotExist(err) {
		return []string{"no se pudo dejar el lockfile como estaba: " + err.Error()}
	}
	return nil
}

func addManagedDesde(in AddInput, reg Registro) (AddResult, []string) {
	entries, problems := Add(in)
	if len(problems) > 0 {
		return AddResult{}, problems
	}
	nuevo, ok := registroDe(entries)
	if !ok || (reg.Origen != "" && nuevo.Origen != reg.Origen) {
		return AddResult{Entries: entries, Registro: reg}, nil
	}
	var warnings []string
	if err := RegistrarOrganizacion(in.Root, nuevo); err != nil {
		warnings = append(warnings, "se importó, pero no se pudo actualizar el registro central: "+err.Error())
	}
	if nuevo == reg {
		return AddResult{Entries: entries, Registro: reg}, warnings
	}
	if err := SaveRegistro(in.Root, nuevo); err != nil {
		return AddResult{Entries: entries, Registro: reg}, append(warnings, "se importó, pero no se pudo guardar el registro en config.yaml: "+err.Error())
	}
	return AddResult{Entries: entries, Registro: nuevo, Guardado: true}, warnings
}

// registroDe deduce el registro a partir de lo importado: solo cuenta si los
// blueprints salieron de subcarpetas de un mismo repositorio (no de un
// repositorio que es en sí un blueprint).
func registroDe(entries []LockEntry) (Registro, bool) {
	if len(entries) == 0 {
		return Registro{}, false
	}
	for _, e := range entries {
		if e.Subdir == "" || e.Origen != entries[0].Origen {
			return Registro{}, false
		}
	}
	return Registro{Origen: entries[0].Origen}, true
}
