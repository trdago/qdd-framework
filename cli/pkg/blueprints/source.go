package blueprints

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// LockEntry fija de dónde salió un blueprint importado y su huella, para que
// el gate detecte si la copia del proyecto se alteró a mano.
type LockEntry struct {
	ID         string `yaml:"id"`
	Version    string `yaml:"version"`
	Origen     string `yaml:"origen"`
	Subdir     string `yaml:"subdir,omitempty"`
	Ref        string `yaml:"ref,omitempty"`
	Commit     string `yaml:"commit,omitempty"`
	SHA256     string `yaml:"sha256"`
	AgregadoEl string `yaml:"agregado_el"`
}

// Lock es el contenido de .qdd/blueprints.lock.yaml.
type Lock struct {
	Blueprints []LockEntry `yaml:"blueprints"`
}

// Entry busca la entrada de un blueprint por id.
func (l Lock) Entry(id string) (LockEntry, bool) {
	for _, e := range l.Blueprints {
		if e.ID == id {
			return e, true
		}
	}
	return LockEntry{}, false
}

// CatalogName es el índice opcional de un repositorio con varios blueprints.
const CatalogName = "catalogo.yaml"

// Catalogo es el índice de un repositorio de blueprints de la organización.
// Sin catálogo, cada carpeta de primer nivel con un blueprint.yaml es un
// blueprint con el nombre de la carpeta.
type Catalogo struct {
	Organizacion string              `yaml:"organizacion,omitempty"`
	Blueprints   map[string]string   `yaml:"blueprints,omitempty"` // id → carpeta
	Paquetes     map[string][]string `yaml:"paquetes,omitempty"`   // nombre → ids
}

// AddInput describe qué importar.
type AddInput struct {
	Root       string
	Origen     string   // carpeta local, o repositorio git (opcionalmente repo//subcarpeta)
	Ref        string   // tag, rama o commit del repositorio git
	Blueprints []string // ids a importar de un repositorio con varios
	Paquete    string   // paquete del catálogo a importar
	Todos      bool     // importar todos los blueprints del repositorio
}

// Add importa uno o varios blueprints al proyecto con un solo clon: los
// valida TODOS antes de copiar nada, copia cada carpeta a .qdd/blueprints/<id>
// y registra origen, commit y huella en el lockfile. Un repositorio git se
// clona con las credenciales git del usuario, así que un repositorio privado
// solo lo puede importar quien tenga acceso a él.
func Add(in AddInput) ([]LockEntry, []string) {
	origen := strings.TrimSpace(in.Origen)
	if origen == "" || strings.HasPrefix(origen, "-") {
		return nil, []string{"origen: indica una carpeta o un repositorio git (p. ej. git@github.com:org/qdd-blueprints.git)"}
	}
	base, subdir, entry, cleanup, err := fetch(origen, in.Ref)
	defer cleanup()
	if err != nil {
		return nil, []string{err.Error()}
	}
	carpetas, problems := seleccionar(base, subdir, in)
	if len(problems) > 0 {
		return nil, problems
	}
	bps, problems := parseSeleccion(base, carpetas)
	if len(problems) > 0 {
		return nil, problems
	}
	var entries []LockEntry
	for i, b := range bps {
		e := entry
		e.Subdir = carpetas[i]
		installed, err := install(in.Root, b, e)
		if err != nil {
			return entries, []string{err.Error()}
		}
		entries = append(entries, installed)
	}
	return entries, nil
}

// seleccionar decide qué carpetas del origen importar.
func seleccionar(base, subdir string, in AddInput) ([]string, []string) {
	pidio := len(in.Blueprints) > 0 || in.Paquete != "" || in.Todos
	if subdir != "" && pidio {
		return nil, []string{"con 'repo//subcarpeta' se importa solo esa subcarpeta; para elegir varios, indica el repositorio sin '//'"}
	}
	if subdir != "" {
		return []string{subdir}, nil
	}
	if _, err := os.Stat(filepath.Join(base, ManifestName)); err == nil && !pidio {
		return []string{""}, nil
	}
	cat, disponibles, err := catalogo(base)
	if err != nil {
		return nil, []string{err.Error()}
	}
	ids, problems := idsPedidos(in, cat, disponibles)
	if len(problems) > 0 {
		return nil, problems
	}
	var carpetas []string
	for _, id := range ids {
		carpetas = append(carpetas, disponibles[id])
	}
	return carpetas, nil
}

func idsPedidos(in AddInput, cat Catalogo, disponibles map[string]string) ([]string, []string) {
	pedidos := append([]string{}, in.Blueprints...)
	if in.Paquete != "" {
		paquete, ok := cat.Paquetes[in.Paquete]
		if !ok {
			return nil, []string{fmt.Sprintf("el repositorio no tiene el paquete '%s'. %s", in.Paquete, oferta(cat, disponibles))}
		}
		pedidos = append(pedidos, paquete...)
	}
	if in.Todos {
		pedidos = append(pedidos, sortedKeys(disponibles)...)
	}
	pedidos = dedupe(pedidos)
	if len(pedidos) == 0 {
		return nil, []string{"el repositorio tiene varios blueprints: elige cuáles (qdd blueprint add <repo> auth auditoria), un --paquete o --todos. " + oferta(cat, disponibles)}
	}
	var problems []string
	for _, id := range pedidos {
		if _, ok := disponibles[id]; !ok {
			problems = append(problems, fmt.Sprintf("el repositorio no tiene el blueprint '%s'. %s", id, oferta(cat, disponibles)))
		}
	}
	return pedidos, problems
}

func oferta(cat Catalogo, disponibles map[string]string) string {
	s := "Blueprints: " + firstNonEmpty(strings.Join(sortedKeys(disponibles), ", "), "ninguno")
	if len(cat.Paquetes) == 0 {
		return s + "."
	}
	var paquetes []string
	for _, nombre := range sortedKeys(cat.Paquetes) {
		paquetes = append(paquetes, fmt.Sprintf("%s (%s)", nombre, strings.Join(cat.Paquetes[nombre], ", ")))
	}
	return s + ". Paquetes: " + strings.Join(paquetes, "; ") + "."
}

// catalogo lee catalogo.yaml (si existe) y devuelve los blueprints
// disponibles: id → carpeta relativa a la raíz del repositorio.
func catalogo(base string) (Catalogo, map[string]string, error) {
	var cat Catalogo
	data, err := os.ReadFile(filepath.Join(base, CatalogName))
	if err == nil {
		if err := yaml.Unmarshal(data, &cat); err != nil {
			return Catalogo{}, nil, fmt.Errorf("%s: YAML inválido: %w", CatalogName, err)
		}
	}
	disponibles := descubrir(base)
	for id, carpeta := range cat.Blueprints {
		if !isInside(carpeta) {
			return Catalogo{}, nil, fmt.Errorf("%s: la carpeta '%s' de '%s' debe ser relativa al repositorio", CatalogName, carpeta, id)
		}
		disponibles[id] = filepath.ToSlash(filepath.Clean(carpeta))
	}
	for nombre, ids := range cat.Paquetes {
		for _, id := range ids {
			if _, ok := disponibles[id]; ok {
				continue
			}
			return Catalogo{}, nil, fmt.Errorf("%s: el paquete '%s' incluye '%s', que no existe en el repositorio", CatalogName, nombre, id)
		}
	}
	return cat, disponibles, nil
}

func descubrir(base string) map[string]string {
	out := map[string]string{}
	entries, _ := os.ReadDir(base)
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if _, err := os.Stat(filepath.Join(base, e.Name(), ManifestName)); err != nil {
			continue
		}
		out[e.Name()] = e.Name()
	}
	return out
}

// parseSeleccion lee y valida todos los blueprints elegidos; si alguno falla,
// no se importa ninguno.
func parseSeleccion(base string, carpetas []string) ([]Blueprint, []string) {
	var bps []Blueprint
	var problems []string
	ids := map[string]string{}
	for _, c := range carpetas {
		b, err := ParseDir(filepath.Join(base, filepath.FromSlash(c)))
		if err != nil {
			problems = append(problems, err.Error())
			continue
		}
		label := firstNonEmpty(c, b.ID)
		if invalid := b.Validate(); len(invalid) > 0 {
			problems = append(problems, prefix(fmt.Sprintf("blueprint '%s' inválido: ", label), invalid)...)
			continue
		}
		if previa, dup := ids[b.ID]; dup {
			problems = append(problems, fmt.Sprintf("las carpetas '%s' y '%s' declaran el mismo id '%s'", previa, c, b.ID))
			continue
		}
		ids[b.ID] = c
		bps = append(bps, b)
	}
	return bps, problems
}

func install(root string, b Blueprint, entry LockEntry) (LockEntry, error) {
	dest := filepath.Join(root, Dir, b.ID)
	if err := os.RemoveAll(dest); err != nil {
		return LockEntry{}, err
	}
	if err := copyTree(b.Dir, dest); err != nil {
		return LockEntry{}, err
	}
	sum, err := HashDir(dest)
	if err != nil {
		return LockEntry{}, err
	}
	entry.ID = b.ID
	entry.Version = b.Version
	entry.SHA256 = sum
	entry.AgregadoEl = time.Now().Format("2006-01-02")
	return entry, saveLockEntry(root, entry)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// fetch deja el origen en una carpeta local y devuelve su raíz, la subcarpeta
// pedida con 'repo//subcarpeta' y los datos del lockfile. Devuelve siempre
// una función de limpieza (para el clon temporal), aunque falle.
func fetch(origen, ref string) (string, string, LockEntry, func(), error) {
	noop := func() {}
	if info, err := os.Stat(origen); err == nil && info.IsDir() {
		abs, _ := filepath.Abs(origen)
		return abs, "", LockEntry{Origen: origen, Commit: gitHead(abs)}, noop, nil
	}
	repo, subdir := splitSource(origen)
	if subdir != "" && !isInside(subdir) {
		return "", "", LockEntry{}, noop, fmt.Errorf("origen: la subcarpeta '%s' debe ser relativa al repositorio", subdir)
	}
	tmp, err := os.MkdirTemp("", "qdd-blueprint-")
	if err != nil {
		return "", "", LockEntry{}, noop, err
	}
	cleanup := func() { os.RemoveAll(tmp) }
	args := []string{"-c", "protocol.ext.allow=never", "clone", "--quiet", "--depth", "1"}
	if ref != "" {
		args = append(args, "--branch", ref)
	}
	args = append(args, "--", repo, tmp)
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		return "", "", LockEntry{}, cleanup, fmt.Errorf("no se pudo clonar '%s' (¿tienes acceso al repositorio?): %s", repo, strings.TrimSpace(string(out)))
	}
	return tmp, subdir, LockEntry{Origen: repo, Ref: ref, Commit: gitHead(tmp)}, cleanup, nil
}

// splitSource separa "repo//subcarpeta" (convención de Terraform); el "//"
// de un esquema como https:// no cuenta.
func splitSource(origen string) (string, string) {
	start := 0
	if i := strings.Index(origen, "://"); i >= 0 {
		start = i + 3
	}
	i := strings.Index(origen[start:], "//")
	if i < 0 {
		return origen, ""
	}
	return origen[:start+i], strings.Trim(origen[start+i+2:], "/")
}

func gitHead(dir string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// blueprintFiles lista los archivos regulares de un blueprint, ordenados y
// relativos a su carpeta. Omite .git y los enlaces simbólicos: un blueprint
// nunca trae archivos de fuera de su carpeta.
func blueprintFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(files)
	return files, err
}

// HashDir calcula la huella de un blueprint: sha256 de cada ruta y contenido.
func HashDir(dir string) (string, error) {
	files, err := blueprintFiles(dir)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	for _, rel := range files {
		f, err := os.Open(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			return "", err
		}
		io.WriteString(h, rel+"\x00")
		_, err = io.Copy(h, f)
		f.Close()
		if err != nil {
			return "", err
		}
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func copyTree(src, dest string) error {
	files, err := blueprintFiles(src)
	if err != nil {
		return err
	}
	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(src, filepath.FromSlash(rel)))
		if err != nil {
			return err
		}
		target := filepath.Join(dest, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0644); err != nil {
			return err
		}
	}
	return nil
}

// LoadLock lee el lockfile; sin lockfile devuelve uno vacío.
func LoadLock(root string) (Lock, error) {
	data, err := os.ReadFile(filepath.Join(root, LockFile))
	if os.IsNotExist(err) {
		return Lock{}, nil
	}
	if err != nil {
		return Lock{}, err
	}
	var l Lock
	if err := yaml.Unmarshal(data, &l); err != nil {
		return Lock{}, fmt.Errorf("%s: YAML inválido: %w", LockFile, err)
	}
	return l, nil
}

func saveLockEntry(root string, entry LockEntry) error {
	l, err := LoadLock(root)
	if err != nil {
		return err
	}
	var out []LockEntry
	for _, e := range l.Blueprints {
		if e.ID == entry.ID {
			continue
		}
		out = append(out, e)
	}
	l.Blueprints = append(out, entry)
	sort.Slice(l.Blueprints, func(i, j int) bool { return l.Blueprints[i].ID < l.Blueprints[j].ID })
	data, err := yaml.Marshal(l)
	if err != nil {
		return err
	}
	header := "# Blueprints importados con 'qdd blueprint add'. No se edita a mano.\n"
	return os.WriteFile(filepath.Join(root, LockFile), append([]byte(header), data...), 0644)
}

// Find lee un blueprint importado al proyecto.
func Find(root, id string) (Blueprint, error) {
	if !idPattern.MatchString(id) {
		return Blueprint{}, fmt.Errorf("'%s' no es un id de blueprint válido", id)
	}
	b, err := ParseDir(filepath.Join(root, Dir, id))
	if err != nil {
		return Blueprint{}, fmt.Errorf("el blueprint '%s' no está importado; usa 'qdd blueprint add <origen>' (%v)", id, err)
	}
	return b, nil
}

// List lee los blueprints importados al proyecto.
func List(root string) ([]Blueprint, []error) {
	entries, _ := os.ReadDir(filepath.Join(root, Dir))
	var out []Blueprint
	var errs []error
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		b, err := ParseDir(filepath.Join(root, Dir, e.Name()))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		out = append(out, b)
	}
	return out, errs
}
