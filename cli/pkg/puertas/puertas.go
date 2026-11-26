// Package puertas encamina a la IA por acciones (ficha puertas): ve en un comando si commitea o publica, o en una
// herramienta si edita, y decide, sin modelo y sin tokens, si la acción pasa. Puertas: audit (no commitear lo que qdd
// audit no aprobó), publicar (va como pedido de la persona), finding (un fix nombra su finding) y ficha (leer la ficha
// antes de editar lo que cubre). Decidir es puro: los hechos los trae quien llama.
package puertas

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// Las puertas.
const (
	PuertaAudit    = "audit"
	PuertaPublicar = "publicar"
	PuertaSalto    = "salto" // git commit --no-verify: la IA no se salta una puerta
	PuertaFinding  = "finding"
	PuertaFicha    = "ficha"
)

// Los resultados de una decisión. Inactiva: el proyecto no tiene puertas; Error: no se pudo decidir y la acción pasa.
const (
	Pasa     = "pasa"
	Rechaza  = "rechaza"
	Error    = "error"
	Inactiva = "inactiva"
	NoAplica = "no-aplica" // la puerta no tiene nada que revisar: un commit que no es fix, una zona sin ficha
)

// Los valores de puertas.publicar en .qdd/policies.yaml: pedido (por defecto) o ia.
const (
	PublicarPedido = "pedido"
	PublicarIA     = "ia"
)

// Accion es lo que un comando hace y una puerta revisa. Dir es la carpeta donde corre, relativa a la del comando ("" es
// la misma); Refs, solo en un git push sin --tags: lo que empuja, para ver si es un tag. Mensaje y ArchivoMensaje, solo
// en la puerta finding (-m o -F de git commit); Archivo, solo en la puerta ficha: lo que se edita, relativo a la raíz.
type Accion struct {
	Puerta         string
	Dir            string
	Que            string
	Refs           []string
	Mensaje        string
	ArchivoMensaje string
	Archivo        string
}

// Politica es puertas en .qdd/policies.yaml: sin la clave, el proyecto no tiene puertas.
type Politica struct {
	Activa   bool
	Publicar string
}

// Hechos es lo que la puerta necesita saber, ya leído. Marca es la huella del último qdd audit aprobado ("" sin
// ninguno); Huella, la del árbol de trabajo ahora.
// Mensaje es el del commit; Findings, los del proyecto (id → tiene test_result). Ficha es la que cubre el archivo que se
// edita ("" si ninguna) y Leida, si la sesión ya pidió su contexto.
type Hechos struct {
	Accion    Accion
	Politica  Politica
	Marca     string
	Huella    string
	ErrHuella error
	Mensaje   string
	Findings  map[string]bool
	Ficha     string
	Leida     bool
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
	msgSinFND   = "Puerta de QDD: un fix nombra su finding. Regístralo en .qdd/project/findings (FND-NNN, con test_result: su prueba de regresión) y nómbralo en el mensaje del commit."
	msgFNDMalo  = "Puerta de QDD: %s no está en .qdd/project/findings o le falta test_result (su prueba de regresión). Complétalo y vuelve a commitear."
	msgFicha    = "Puerta de QDD: antes de editar %s lee su ficha «%s»: qdd_project_context con esa ruta en paths (sin MCP: qdd modules context %s)."
)

var (
	esFix   = regexp.MustCompile(`^fix(\([^)]*\))?!?:`)
	idsFND  = regexp.MustCompile(`FND-[0-9]+`)
	maxRuta = 70
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
	if h.Accion.Puerta == PuertaFinding {
		return decidirFinding(h, d)
	}
	if h.Accion.Puerta == PuertaFicha {
		return decidirFicha(h, d)
	}
	return decidirAudit(h, d)
}

// decidirFinding: un commit fix(...) o fix: nombra al menos un FND-NNN que exista con su prueba de regresión.
func decidirFinding(h Hechos, d Decision) Decision {
	if !esFix.MatchString(strings.TrimSpace(h.Mensaje)) {
		d.Resultado = NoAplica
		return d
	}
	ids := idsFND.FindAllString(h.Mensaje, -1)
	if len(ids) == 0 {
		return rechazar(d, msgSinFND)
	}
	if slices.ContainsFunc(ids, func(id string) bool { return h.Findings[id] }) {
		d.Resultado = Pasa
		return d
	}
	return rechazar(d, fmt.Sprintf(msgFNDMalo, strings.Join(ids, ", ")))
}

// decidirFicha: una zona sin ficha no bloquea (el contexto ya le pide a la IA que proponga una).
func decidirFicha(h Hechos, d Decision) Decision {
	if h.Ficha == "" {
		d.Resultado = NoAplica
		return d
	}
	if h.Leida {
		d.Resultado = Pasa
		return d
	}
	ruta := acortar(h.Accion.Archivo)
	return rechazar(d, fmt.Sprintf(msgFicha, ruta, h.Ficha, ruta))
}

// LeidaCubre dice si pedir el contexto de leida mostró la ficha del archivo: es la misma ruta o la ficha la cubre.
func LeidaCubre(leida, archivo, ficha string, cubiertas func(string) []string) bool {
	if leida == "" {
		return false
	}
	return leida == archivo || slices.Contains(cubiertas(leida), ficha)
}

// acortar deja una ruta larga en su final, para que el rechazo quepa en 280 caracteres.
func acortar(ruta string) string {
	r := []rune(ruta)
	if len(r) <= maxRuta {
		return ruta
	}
	return "…" + string(r[len(r)-maxRuta+1:])
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
		for _, a := range accionesDe(programa, palabras[1:]) {
			a.Dir = unir(dir, a.Dir)
			out = append(out, a)
		}
	}
	return out
}

func accionesDe(programa string, args []string) []Accion {
	if programa == "git" {
		return accionesGit(args)
	}
	a, ok := accionDe(programa, args)
	if !ok {
		return nil
	}
	return []Accion{a}
}

func accionDe(programa string, args []string) (Accion, bool) {
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

func accionesGit(args []string) []Accion {
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
		return nil
	}
	if args[0] == "commit" {
		acciones := accionesCommit(args[1:])
		for i := range acciones {
			acciones[i].Dir = dir
		}
		return acciones
	}
	if args[0] != "push" {
		return nil
	}
	a, ok := accionPush(args[1:])
	if !ok {
		return nil
	}
	a.Dir = dir
	return []Accion{a}
}

// Opciones de commit cuyo valor va en la palabra siguiente.
var conValorCommit = map[string]bool{"--message": true, "--file": true, "--author": true, "--date": true, "--reuse-message": true,
	"--reedit-message": true, "--fixup": true, "--squash": true, "--cleanup": true, "--template": true, "--trailer": true}

// mensajeCommit junta lo que dicen -m y -F de un commit.
type mensajeCommit struct {
	partes  []string
	archivo string
}

func (m *mensajeCommit) tomar(opcion, valor string) {
	if opcion == "m" || opcion == "--message" {
		m.partes = append(m.partes, textoDeMensaje(valor))
	}
	if opcion == "F" || opcion == "--file" {
		m.archivo = valor
	}
}

// accionesCommit: un commit pasa por la puerta audit y por la finding (con su mensaje); con --no-verify, solo por la
// que lo rechaza.
func accionesCommit(args []string) []Accion {
	salto := []Accion{{Puerta: PuertaSalto, Que: "git commit --no-verify"}}
	var m mensajeCommit
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}
		if arg == "--no-verify" {
			return salto
		}
		if opcion, valor, ok := strings.Cut(arg, "="); ok && conValorCommit[opcion] {
			m.tomar(opcion, valor)
			continue
		}
		if conValorCommit[arg] {
			m.tomar(arg, siguiente(args, i))
			i++
			continue
		}
		if strings.HasPrefix(arg, "--") || !strings.HasPrefix(arg, "-") {
			continue
		}
		c := cortas(arg[1:])
		if c.sinVerificar {
			return salto
		}
		if c.opcion != "" && c.valor == "" {
			c.valor = siguiente(args, i)
			i++
		}
		m.tomar(c.opcion, c.valor)
	}
	return []Accion{
		{Puerta: PuertaAudit, Que: "git commit"},
		{Puerta: PuertaFinding, Que: "git commit", Mensaje: strings.Join(m.partes, "\n\n"), ArchivoMensaje: m.archivo},
	}
}

func siguiente(args []string, i int) string {
	if i+1 < len(args) {
		return args[i+1]
	}
	return ""
}

// grupoCorto es un grupo de opciones cortas de commit (-qm): si alguna es -n (--no-verify) y la que lleva valor, con
// lo que le sigue en el mismo grupo (-mfix: → «fix:»); valor vacío es que va en la palabra siguiente.
type grupoCorto struct {
	sinVerificar bool
	opcion       string
	valor        string
}

func cortas(grupo string) grupoCorto {
	for i, r := range grupo {
		if r == 'n' {
			return grupoCorto{sinVerificar: true}
		}
		if strings.ContainsRune("mFCct", r) {
			return grupoCorto{opcion: string(r), valor: grupo[i+1:]}
		}
	}
	return grupoCorto{}
}

// textoDeMensaje saca el texto de un -m "$(cat <<'EOF' … EOF)", la forma en que la IA escribe un mensaje largo.
func textoDeMensaje(valor string) string {
	if !strings.HasPrefix(valor, "$(cat <<") {
		return valor
	}
	primera, resto, ok := strings.Cut(valor, "\n")
	if !ok {
		return valor
	}
	delim := strings.Trim(strings.TrimPrefix(strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(primera, "$(cat <<"), ")")), "-"), `'"`)
	lineas := strings.Split(resto, "\n")
	fin := slices.IndexFunc(lineas, func(l string) bool { return strings.TrimSpace(l) == delim })
	if fin < 0 {
		return strings.TrimSpace(resto)
	}
	return strings.TrimSpace(strings.Join(lineas[:fin], "\n"))
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
