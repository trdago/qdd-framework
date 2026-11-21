// Package puertas encamina a la IA por acciones (ficha puertas): ve en un comando si commitea o publica y decide, sin
// modelo y sin tokens, si la acción pasa. Etapa 1: la puerta audit (no commitear lo que qdd audit no aprobó) y la
// puerta publicar (publicar va como pedido de la persona). Decidir es puro: los hechos los trae quien llama.
package puertas

import (
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Las puertas.
const (
	PuertaAudit    = "audit"
	PuertaPublicar = "publicar"
	PuertaSalto    = "salto" // git commit --no-verify: la IA no se salta una puerta
)

// Los resultados de una decisión. Inactiva: el proyecto no tiene puertas; Error: no se pudo decidir y la acción pasa.
const (
	Pasa     = "pasa"
	Rechaza  = "rechaza"
	Error    = "error"
	Inactiva = "inactiva"
)

// Los valores de puertas.publicar en .qdd/policies.yaml: pedido (por defecto) o ia.
const (
	PublicarPedido = "pedido"
	PublicarIA     = "ia"
)

// Accion es lo que un comando hace y una puerta revisa. Dir es la carpeta donde corre, relativa a la del comando ("" es
// la misma); Refs, solo en un git push sin --tags: lo que empuja, para ver si es un tag.
type Accion struct {
	Puerta string
	Dir    string
	Que    string
	Refs   []string
}

// Politica es puertas en .qdd/policies.yaml: sin la clave, el proyecto no tiene puertas.
type Politica struct {
	Activa   bool
	Publicar string
}

// Hechos es lo que la puerta necesita saber, ya leído. Marca es la huella del último qdd audit aprobado ("" sin
// ninguno); Huella, la del árbol de trabajo ahora.
type Hechos struct {
	Accion    Accion
	Politica  Politica
	Marca     string
	Huella    string
	ErrHuella error
}

// Decision es lo que responde una puerta. Mensaje solo cuando rechaza: es lo único que lee la IA.
type Decision struct {
	Puerta    string
	Resultado string
	Mensaje   string
}

const (
	msgSalto    = "Puerta de QDD: la IA no se salta las puertas (--no-verify). Corre qdd audit, arregla lo que diga y commitea sin --no-verify."
	msgSinAudit = "Puerta de QDD: no hay un qdd audit aprobado para este árbol. Corre qdd audit en esta carpeta y commitea sin cambiar nada después."
	msgCambio   = "Puerta de QDD: el árbol cambió desde el último qdd audit aprobado. Vuelve a correr qdd audit y commitea sin tocar nada después."
	msgPublicar = "Puerta de QDD: publicar (%s) lo confirma la persona. Déjalo como pedido: qdd consola pedir -m '<por qué>' '<comando>'; corre cuando ella lo confirma en el panel."
)

// Decidir aplica la puerta de la acción a sus hechos: la misma acción con los mismos hechos da siempre lo mismo.
func Decidir(h Hechos) Decision {
	d := Decision{Puerta: h.Accion.Puerta}
	if !h.Politica.Activa {
		d.Resultado = Inactiva
		return d
	}
	if h.Accion.Puerta == PuertaSalto {
		return rechazar(d, msgSalto)
	}
	if h.Accion.Puerta == PuertaPublicar {
		return decidirPublicar(h, d)
	}
	return decidirAudit(h, d)
}

func decidirPublicar(h Hechos, d Decision) Decision {
	if h.Politica.Publicar == PublicarIA {
		d.Resultado = Pasa
		return d
	}
	return rechazar(d, fmt.Sprintf(msgPublicar, h.Accion.Que))
}

// decidirAudit: al día es que nada cambió desde el audit aprobado, así que commitear una parte de lo auditado pasa.
func decidirAudit(h Hechos, d Decision) Decision {
	if h.ErrHuella != nil {
		d.Resultado = Error
		return d
	}
	if h.Marca == "" {
		return rechazar(d, msgSinAudit)
	}
	if h.Marca != h.Huella {
		return rechazar(d, msgCambio)
	}
	d.Resultado = Pasa
	return d
}

func rechazar(d Decision, mensaje string) Decision {
	d.Resultado, d.Mensaje = Rechaza, mensaje
	return d
}

// PoliticaDe lee la política de .qdd/policies.yaml; uno que no se entiende no activa nada.
func PoliticaDe(datos []byte) Politica {
	var p struct {
		Puertas *struct {
			Publicar string `yaml:"publicar"`
		} `yaml:"puertas"`
	}
	if yaml.Unmarshal(datos, &p) != nil || p.Puertas == nil {
		return Politica{}
	}
	return Politica{Activa: true, Publicar: p.Puertas.Publicar}
}

// Acciones son los commits y publicaciones de un comando de shell, en orden. Lo que va entre comillas o en un heredoc
// es texto, no un comando; cd y git -C cambian la carpeta de lo que sigue.
func Acciones(comando string) []Accion {
	var out []Accion
	dir := ""
	for _, palabras := range comandosDe(comando) {
		palabras = sinPrefijos(palabras)
		if len(palabras) == 0 {
			continue
		}
		programa := filepath.Base(palabras[0])
		if programa == "cd" && len(palabras) > 1 {
			dir = unir(dir, palabras[1])
			continue
		}
		a, ok := accionDe(programa, palabras[1:])
		if !ok {
			continue
		}
		a.Dir = unir(dir, a.Dir)
		out = append(out, a)
	}
	return out
}

func accionDe(programa string, args []string) (Accion, bool) {
	if programa == "git" {
		return accionGit(args)
	}
	if programa == "gh" && len(args) > 1 && args[1] == "merge" && args[0] == "pr" {
		return Accion{Puerta: PuertaPublicar, Que: "gh pr merge"}, true
	}
	if programa == "gh" && len(args) > 1 && args[1] == "create" && args[0] == "release" {
		return Accion{Puerta: PuertaPublicar, Que: "gh release create"}, true
	}
	if (programa == "npm" || programa == "pnpm" || programa == "yarn") && primeroSinGuion(args) == "publish" {
		return Accion{Puerta: PuertaPublicar, Que: programa + " publish"}, true
	}
	return Accion{}, false
}

func accionGit(args []string) (Accion, bool) {
	dir := ""
	for len(args) > 0 && strings.HasPrefix(args[0], "-") {
		if (args[0] == "-C" || args[0] == "-c") && len(args) > 1 {
			if args[0] == "-C" {
				dir = unir(dir, args[1])
			}
			args = args[1:]
		}
		args = args[1:]
	}
	if len(args) == 0 {
		return Accion{}, false
	}
	if args[0] == "commit" {
		a := accionCommit(args[1:])
		a.Dir = dir
		return a, true
	}
	if args[0] != "push" {
		return Accion{}, false
	}
	a, ok := accionPush(args[1:])
	a.Dir = dir
	return a, ok
}

// Opciones de commit cuyo valor va en la palabra siguiente.
var conValorCommit = map[string]bool{"--message": true, "--file": true, "--author": true, "--date": true, "--reuse-message": true,
	"--reedit-message": true, "--fixup": true, "--squash": true, "--cleanup": true, "--template": true, "--trailer": true}

func accionCommit(args []string) Accion {
	audit := Accion{Puerta: PuertaAudit, Que: "git commit"}
	salto := Accion{Puerta: PuertaSalto, Que: "git commit --no-verify"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return audit
		}
		if arg == "--no-verify" {
			return salto
		}
		if conValorCommit[arg] {
			i++
			continue
		}
		if strings.HasPrefix(arg, "--") || !strings.HasPrefix(arg, "-") {
			continue
		}
		saltar, sinVerificar := cortas(arg[1:])
		if sinVerificar {
			return salto
		}
		if saltar {
			i++
		}
	}
	return audit
}

// cortas lee un grupo de opciones cortas de commit (-qm): si alguna es -n (--no-verify) y si la última espera su
// valor en la palabra siguiente. Lo que sigue a una que lleva valor es ese valor.
func cortas(grupo string) (saltar, sinVerificar bool) {
	for i, r := range grupo {
		if r == 'n' {
			return false, true
		}
		if strings.ContainsRune("mFCct", r) {
			return i == len(grupo)-1, false
		}
	}
	return false, false
}

var conValorPush = map[string]bool{"-o": true, "--push-option": true, "--repo": true, "--receive-pack": true, "--exec": true}

var empujaTags = map[string]bool{"--tags": true, "--follow-tags": true, "--mirror": true}

// accionPush: con --tags publica; si no, lo que empuja (después del remoto) puede ser un tag, y eso lo ve quien llama.
func accionPush(args []string) (Accion, bool) {
	var posicionales []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if empujaTags[arg] {
			return Accion{Puerta: PuertaPublicar, Que: "git push " + arg}, true
		}
		if conValorPush[arg] {
			i++
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		posicionales = append(posicionales, arg)
	}
	if len(posicionales) < 2 {
		return Accion{}, false
	}
	return Accion{Puerta: PuertaPublicar, Que: "git push", Refs: posicionales[1:]}, true
}

// RefDeTag es el tag que empuja un refspec (+refs/tags/v1:refs/tags/v1 → refs/tags/v1), sin el + ni el destino.
func RefDeTag(refspec string) string {
	origen, _, _ := strings.Cut(strings.TrimPrefix(refspec, "+"), ":")
	return origen
}

func primeroSinGuion(args []string) string {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			return a
		}
	}
	return ""
}

// sinPrefijos quita lo que va antes del programa: VAR=x, env, command, time, nohup.
func sinPrefijos(palabras []string) []string {
	for len(palabras) > 0 && (esAsignacion(palabras[0]) || prefijos[palabras[0]]) {
		palabras = palabras[1:]
	}
	return palabras
}

var prefijos = map[string]bool{"env": true, "command": true, "time": true, "nohup": true, "exec": true}

func esAsignacion(p string) bool {
	nombre, _, ok := strings.Cut(p, "=")
	if !ok || nombre == "" {
		return false
	}
	for i, r := range nombre {
		valido := r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (i > 0 && r >= '0' && r <= '9')
		if !valido {
			return false
		}
	}
	return true
}

func unir(base, dir string) string {
	if dir == "" {
		return base
	}
	if filepath.IsAbs(dir) || base == "" {
		return dir
	}
	return filepath.Join(base, dir)
}
