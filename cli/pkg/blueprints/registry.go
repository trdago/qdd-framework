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
// la primera vez que se importa desde él; después basta con nombrar los
// blueprints (qdd blueprint add auth auditoria).
type Registro struct {
	Origen string `yaml:"origen"`
	Ref    string `yaml:"ref,omitempty"`
}

// LoadRegistro lee el registro de .qdd/config.yaml; vacío si no hay.
func LoadRegistro(root string) (Registro, error) {
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

// AddManaged importa como Add, pero administra el registro de la
// organización: sin origen usa el de .qdd/config.yaml (y su ref, si no se
// pide otra), y la primera vez que se importa desde un repositorio de varios
// blueprints lo guarda para que después baste con nombrarlos.
func AddManaged(in AddInput) (AddResult, []string) {
	reg, err := LoadRegistro(in.Root)
	if err != nil {
		return AddResult{}, []string{err.Error()}
	}
	usaRegistro := strings.TrimSpace(in.Origen) == ""
	if usaRegistro && reg.Origen == "" {
		return AddResult{}, []string{"todavía no hay repositorio de blueprints de la organización: indícalo una vez y QDD lo recordará (qdd blueprint add git@github.com:org/qdd-blueprints.git --ref v1.0.0 auth)"}
	}
	if usaRegistro {
		in.Origen = reg.Origen
		in.Ref = firstNonEmpty(in.Ref, reg.Ref)
	}
	entries, problems := Add(in)
	if len(problems) > 0 {
		return AddResult{}, problems
	}
	nuevo, ok := registroDe(entries, in.Ref)
	if !ok || (reg.Origen != "" && nuevo.Origen != reg.Origen) || nuevo == reg {
		return AddResult{Entries: entries, Registro: reg}, nil
	}
	if err := SaveRegistro(in.Root, nuevo); err != nil {
		return AddResult{Entries: entries, Registro: reg}, []string{"se importó, pero no se pudo guardar el registro en config.yaml: " + err.Error()}
	}
	return AddResult{Entries: entries, Registro: nuevo, Guardado: true}, nil
}

// registroDe deduce el registro a partir de lo importado: solo cuenta si los
// blueprints salieron de subcarpetas de un mismo repositorio (no de una
// carpeta que es en sí un blueprint).
func registroDe(entries []LockEntry, ref string) (Registro, bool) {
	if len(entries) == 0 {
		return Registro{}, false
	}
	for _, e := range entries {
		if e.Subdir == "" || e.Origen != entries[0].Origen {
			return Registro{}, false
		}
	}
	return Registro{Origen: entries[0].Origen, Ref: ref}, true
}
