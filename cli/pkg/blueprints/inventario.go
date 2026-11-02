package blueprints

import (
	"path"
	"sort"
	"strings"
)

// Estado de un blueprint en el catálogo del panel (pedido del usuario, 2026-11-01). Todo se calcula sin red y sin IA:
// con el espejo central (lo trae 'qdd blueprint sync') y el lockfile del proyecto.
const (
	CatAlDia      = "al-dia"      // instalado en la última versión publicada
	CatSoloLock   = "solo-lock"   // hay un tag nuevo con la misma receta (o solo redacción): se actualiza sin la IA
	CatConCambios = "con-cambios" // la receta cambió: actualizar es determinista, adaptar el módulo es de la IA
	CatDisponible = "disponible"  // publicado y sin instalar
	CatSinMedir   = "sin-medir"   // instalado, pero no hay copia central para comparar
)

// EntradaInventario es un blueprint del catálogo.
type EntradaInventario struct {
	ID        string   `json:"id"`
	Nombre    string   `json:"nombre,omitempty"`
	Origen    string   `json:"origen"`
	Subdir    string   `json:"subdir,omitempty"`
	Instalada string   `json:"instalada,omitempty"`
	Publicada string   `json:"publicada,omitempty"`
	Ref       string   `json:"ref,omitempty"` // el tag de la última versión publicada
	Estado    string   `json:"estado"`
	Cambio    string   `json:"cambio,omitempty"` // patch, minor o major
	Razones   []string `json:"razones,omitempty"`
}

// Inventario es lo que el panel muestra de los blueprints, con sus cuentas.
type Inventario struct {
	Entradas    []EntradaInventario `json:"entradas"`
	Instalados  int                 `json:"instalados"`
	Disponibles int                 `json:"disponibles"`
	SinIA       int                 `json:"sin_ia"` // con versión nueva que se toma sin la IA
	ConIA       int                 `json:"con_ia"` // con versión nueva que cambia lo que hay que construir
	Problemas   []string            `json:"problemas,omitempty"`
}

// maxRazones son las razones de un cambio que entran en el panel.
const maxRazones = 3

// InventarioDe lista los blueprints de las fuentes del proyecto (su organización y las de su lockfile): los instalados
// con su estado y los demás como disponibles. Sin copia central de una fuente, sus instalados quedan sin medir y el
// problema dice cómo traerla; nunca se inventa una versión.
func InventarioDe(root string) (Inventario, error) {
	lock, err := LoadLock(root)
	if err != nil {
		return Inventario{}, err
	}
	var c Inventario
	fuentes, err := fuentesDe(root, lock)
	if err != nil {
		c.Problemas = append(c.Problemas, err.Error())
		return c, nil
	}
	vistos := map[string]bool{}
	for _, repo := range fuentes {
		c.Entradas = append(c.Entradas, entradasDe(repo, lock, vistos, &c)...)
	}
	for _, e := range lock.Blueprints {
		if vistos[e.ID] {
			continue
		}
		c.Entradas = append(c.Entradas, EntradaInventario{ID: e.ID, Origen: e.Origen, Subdir: e.Subdir, Instalada: e.Version, Estado: CatSinMedir})
	}
	sort.SliceStable(c.Entradas, func(i, j int) bool { return orden(c.Entradas[i]) < orden(c.Entradas[j]) })
	contar(&c)
	return c, nil
}

func entradasDe(repo string, lock Lock, vistos map[string]bool, c *Inventario) []EntradaInventario {
	mirror, err := mirrorPath(repo)
	if err != nil || !exists(mirror) {
		c.Problemas = append(c.Problemas, "sin copia central de "+repo+": ↻ Actualizar catálogo (qdd blueprint sync)")
		return nil
	}
	tags, err := versionTags(mirror)
	if err != nil || len(tags) == 0 {
		c.Problemas = append(c.Problemas, repo+" no tiene versiones publicadas (tags vX.Y.Z)")
		return nil
	}
	ultimo := tags[0]
	var out []EntradaInventario
	for _, subdir := range subdirsCon(mirror, ultimo) {
		b, ok := blueprintEn(mirror, ultimo, subdir)
		if !ok || vistos[b.ID] {
			continue
		}
		vistos[b.ID] = true
		e := EntradaInventario{ID: b.ID, Nombre: b.Nombre, Origen: repo, Subdir: subdir, Publicada: b.Version, Ref: ultimo, Estado: CatDisponible}
		out = append(out, comparar(e, instaladoDe(lock, repo, b.ID), mirror, b))
	}
	return out
}

// comparar dice qué pasa con un blueprint instalado frente a su última versión publicada.
func comparar(e EntradaInventario, l *LockEntry, mirror string, nuevo Blueprint) EntradaInventario {
	if l == nil {
		return e
	}
	e.Instalada = l.Version
	if l.Ref == e.Ref || compararVersion(l.Version, nuevo.Version) > 0 {
		e.Estado = CatAlDia
		return e
	}
	base, ok := blueprintEn(mirror, l.Ref, l.Subdir)
	if !ok {
		e.Estado = CatConCambios
		e.Razones = []string{"no se encontró " + l.Ref + " en la copia central"}
		return e
	}
	cambio, razones := Clasificar(base, nuevo)
	e.Cambio = cambio
	if cambio == "" || cambio == CambioPatch {
		e.Estado = CatSoloLock
		return e
	}
	e.Estado = CatConCambios
	e.Razones = razones[:min(len(razones), maxRazones)]
	return e
}

func instaladoDe(lock Lock, repo, id string) *LockEntry {
	for i, e := range lock.Blueprints {
		if e.ID == id && repoCanonico(e.Origen) == repoCanonico(repo) {
			return &lock.Blueprints[i]
		}
	}
	return nil
}

// subdirsCon lista las carpetas que tienen un blueprint.yaml en ref.
func subdirsCon(mirror, ref string) []string {
	out, err := git(mirror, "ls-tree", "-r", "--name-only", ref)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, f := range strings.Fields(out) {
		if path.Base(f) == ManifestName && path.Dir(f) != "." {
			dirs = append(dirs, path.Dir(f))
		}
	}
	return dirs
}

// orden: primero lo que pide atención (con cambios, solo lock), luego lo instalado y al final lo disponible.
func orden(e EntradaInventario) int {
	return map[string]int{CatConCambios: 0, CatSoloLock: 1, CatSinMedir: 2, CatAlDia: 3, CatDisponible: 4}[e.Estado]
}

func contar(c *Inventario) {
	for _, e := range c.Entradas {
		if e.Estado == CatDisponible {
			c.Disponibles++
			continue
		}
		c.Instalados++
		if e.Estado == CatSoloLock {
			c.SinIA++
		}
		if e.Estado == CatConCambios {
			c.ConIA++
		}
	}
}
