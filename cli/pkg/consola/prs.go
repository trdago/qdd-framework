package consola

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/gitenv"
)

// Rule: Prohibido usar 'else'
// Los PRs abiertos del repositorio a un clic en el panel, siempre opcionales (pedido del usuario, 2026-11-02). qdd
// los trae con gh cada IntervaloPRs, aparte y sin demorar al panel (PRsAlDia), y deja un pedido «🔀 Mergear #N» por
// cada uno. Confirmado, «qdd consola mergear-pr N» mergea solo con el CI verde; con el CI corriendo dice «todavía no»
// y el pedido queda armado. Son opcionales: no abren el panel, no son esperas de la persona, ceden su lugar a un
// pedido de la IA y uno descartado no vuelve.

const (
	// MaxPRs es cuántos lugares de la cola pueden ocupar los PRs.
	MaxPRs = 3
	// IntervaloPRs es cada cuánto se preguntan los PRs abiertos.
	IntervaloPRs = time.Minute
	// topeGH es cuánto puede tardar una llamada a gh.
	topeGH = 20 * time.Second
)

// PR es un pull request abierto (gh pr list --json number,title,isDraft).
type PR struct {
	Numero   int    `json:"number"`
	Titulo   string `json:"title"`
	Borrador bool   `json:"isDraft"`
}

// ComandoMergearPR es el comando del pedido de un PR.
func ComandoMergearPR(n int) string {
	return fmt.Sprintf("qdd consola mergear-pr %d", n)
}

func motivoPR(p PR) string {
	return recortar(fmt.Sprintf("🔀 Mergear #%d · %s", p.Numero, p.Titulo), MaxMotivo)
}

// OfrecerPRs deja en la cola un pedido por cada PR abierto (no borrador) que no lo tenga, hasta MaxPRs y sin pasar de
// MaxPedidos, y quita los pendientes de PRs que ya no están abiertos o que un pedido abierto de la IA ya nombra (FND-132).
// No ofrece un PR que ese pedido nombra («#69» en su motivo o comando) ni uno que la persona descartó.
func OfrecerPRs(root string, abiertos []PR, now time.Time) error {
	descartados := leerDescartados(root)
	abierto := map[int]PR{}
	for _, p := range abiertos {
		if !p.Borrador {
			abierto[p.Numero] = p
		}
	}
	err := conCola(root, now, func(cola []Pedido) ([]Pedido, error) {
		var queda []Pedido
		for _, p := range cola {
			if _, sigue := abierto[p.PR]; p.PR != 0 && p.Estado == EstadoPendiente && !sigue {
				continue
			}
			if p.PR != 0 && p.Estado == EstadoPendiente && loNombraOtroPedido(cola, p.PR) {
				continue
			}
			queda = append(queda, p)
		}
		usadas, dePRs := map[string]bool{}, 0
		for _, p := range queda {
			if p.Abierto() {
				usadas[p.Tecla] = true
			}
			if p.Abierto() && p.PR != 0 {
				dePRs++
			}
		}
		for _, n := range numerosEnOrden(abierto) {
			if descartados[n] || yaOfrecido(queda, n) || dePRs >= MaxPRs || len(usadas) >= MaxPedidos {
				continue
			}
			tecla := teclaLibre(usadas)
			usadas[tecla] = true
			dePRs++
			queda = append(queda, Pedido{ID: siguienteID(queda), Comando: ComandoMergearPR(n), Motivo: motivoPR(abierto[n]), Carpeta: root, Creado: now, Tecla: tecla, Estado: EstadoPendiente, PR: n})
		}
		return queda, nil
	})
	if err != nil {
		return err
	}
	return guardarDescartados(root, soloAbiertos(descartados, abierto))
}

func numerosEnOrden(abierto map[int]PR) []int {
	var ns []int
	for n := range abierto {
		ns = append(ns, n)
	}
	sort.Ints(ns)
	return ns
}

// yaOfrecido: un pedido del PR n (en cualquier estado: uno hecho tampoco se repite) o uno abierto que lo nombra.
func yaOfrecido(cola []Pedido, n int) bool {
	for _, p := range cola {
		if p.PR == n {
			return true
		}
	}
	return loNombraOtroPedido(cola, n)
}

// loNombraOtroPedido: hay un pedido abierto de la IA que habla del PR n. El de otro PR no cuenta: que su título mencione
// el #n no mergea el #n.
func loNombraOtroPedido(cola []Pedido, n int) bool {
	for _, p := range cola {
		if p.PR == 0 && p.Abierto() && nombraPR(n, p.Motivo, p.Comando) {
			return true
		}
	}
	return false
}

// nombraPR: el motivo o el comando hablan del PR n («#69», «gh pr merge 69», «gh pr view 69», «mergear-pr 69»).
func nombraPR(n int, motivo, comando string) bool {
	num := strconv.Itoa(n)
	nombra := regexp.MustCompile(`#` + num + `\b|\b(pr merge|pr view|mergear-pr) ` + num + `\b`)
	return nombra.MatchString(motivo) || nombra.MatchString(comando)
}

// sinAutomaticosQueNombra quita los pedidos automáticos pendientes de los PRs que nombra un pedido nuevo de la IA
// (FND-132). Uno ya confirmado sigue su curso: la persona lo decidió.
func sinAutomaticosQueNombra(cola []Pedido, motivo, comando string) []Pedido {
	queda := make([]Pedido, 0, len(cola))
	for _, p := range cola {
		if p.PR != 0 && p.Estado == EstadoPendiente && nombraPR(p.PR, motivo, comando) {
			continue
		}
		queda = append(queda, p)
	}
	return queda
}

// cederUnPR quita el pedido opcional pendiente más viejo (un PR o una versión de qdd) para que entre uno de la IA; false
// si no hay ninguno.
func cederUnPR(cola []Pedido) ([]Pedido, bool) {
	for i, p := range cola {
		if p.Opcional() && p.Estado == EstadoPendiente {
			return append(append([]Pedido{}, cola[:i]...), cola[i+1:]...), true
		}
	}
	return cola, false
}

// ── Descartados: un PR que la persona descartó no vuelve mientras siga abierto ────────────────────────────────────

func rutaDescartados(root string) (string, error) {
	return RutaLocal(root, "prs-descartados")
}

func leerDescartados(root string) map[int]bool {
	out := map[int]bool{}
	ruta, err := rutaDescartados(root)
	if err != nil {
		return out
	}
	b, err := os.ReadFile(ruta)
	if err != nil {
		return out
	}
	var ns []int
	if json.Unmarshal(b, &ns) != nil {
		return out
	}
	for _, n := range ns {
		out[n] = true
	}
	return out
}

func guardarDescartados(root string, descartados map[int]bool) error {
	ruta, err := rutaDescartados(root)
	if err != nil {
		return err
	}
	ns := []int{}
	for n := range descartados {
		ns = append(ns, n)
	}
	sort.Ints(ns)
	if len(ns) == 0 {
		os.Remove(ruta)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0700); err != nil {
		return err
	}
	b, _ := json.Marshal(ns)
	return os.WriteFile(ruta, b, 0600)
}

func anotarDescartado(root string, n int) {
	d := leerDescartados(root)
	d[n] = true
	guardarDescartados(root, d)
}

func soloAbiertos(descartados map[int]bool, abierto map[int]PR) map[int]bool {
	out := map[int]bool{}
	for n := range descartados {
		if _, ok := abierto[n]; ok {
			out[n] = true
		}
	}
	return out
}

// ── Traerlos: aparte, cada IntervaloPRs ───────────────────────────────────────────────────────────────────────────

// gh corre gh en dir con un tope de tiempo.
type gh func(dir string, args ...string) ([]byte, error)

func ghReal(dir string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), topeGH)
	defer cancel()
	c := exec.CommandContext(ctx, "gh", args...)
	c.Dir = dir
	return c.Output()
}

// remoteOrigin es la URL del remote origin de root; vacía si no hay.
func remoteOrigin(root string) string {
	ctx, cancel := context.WithTimeout(context.Background(), topeGH)
	defer cancel()
	c := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	c.Dir = root
	c.Env = gitenv.Limpio(os.Environ())
	b, err := c.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// PRsAlDia lanza aparte «bin consola prs --raiz root» si hace más de IntervaloPRs que no se preguntan; vuelve en el
// acto. Sin gh no hace nada.
func PRsAlDia(root, bin string, now time.Time) {
	if _, err := exec.LookPath("gh"); err != nil {
		return
	}
	marca, err := RutaLocal(root, "prs-revisados")
	if err != nil {
		return
	}
	if info, err := os.Stat(marca); err == nil && now.Sub(info.ModTime()) < IntervaloPRs {
		return
	}
	if os.MkdirAll(filepath.Dir(marca), 0700) != nil || os.WriteFile(marca, nil, 0600) != nil {
		return
	}
	c := exec.Command(bin, "consola", "prs", "--raiz", root)
	c.Dir = root
	separar(c)
	if c.Start() == nil {
		_ = c.Process.Release()
	}
}

// TraerPRs pregunta a gh los PRs abiertos del repositorio de root y los ofrece en el panel.
func TraerPRs(root string, now time.Time) error {
	return traerPRsCon(ghReal, remoteOrigin, root, now)
}

// traerPRsCon: sin un remote de GitHub no hay PRs que preguntar, y gh no se llama (FND-122: con Azure DevOps, cada
// minuto lanzaba un gh que no podía devolver nada).
func traerPRsCon(correr gh, remote func(string) string, root string, now time.Time) error {
	if !strings.Contains(remote(root), "github.com") {
		return nil
	}
	b, err := correr(root, "pr", "list", "--state", "open", "--json", "number,title,isDraft", "--limit", "20")
	if err != nil {
		return fmt.Errorf("gh pr list: %v", err)
	}
	var abiertos []PR
	if err := json.Unmarshal(b, &abiertos); err != nil {
		return fmt.Errorf("gh pr list: %v", err)
	}
	return OfrecerPRs(root, abiertos, now)
}

// ── Mergear: solo con el CI verde ─────────────────────────────────────────────────────────────────────────────────

// CheckPR es un check del PR: un check run (Estado y Conclusion) o un status (Contexto).
type CheckPR struct {
	Estado     string `json:"status"`
	Conclusion string `json:"conclusion"`
	Contexto   string `json:"state"`
}

// VistaPR es lo que hace falta para decidir si un PR se puede mergear.
type VistaPR struct {
	Estado       string    `json:"state"`
	Borrador     bool      `json:"isDraft"`
	Checks       []CheckPR `json:"statusCheckRollup"`
	ConWorkflows bool      `json:"-"`
}

// veredictoPR: -1 si se puede mergear; si no, el código de salida y por qué.
func veredictoPR(n int, v VistaPR) (int, string) {
	if v.Estado == "MERGED" {
		return 0, fmt.Sprintf("El #%d ya estaba mergeado.", n)
	}
	if v.Estado != "OPEN" {
		return 1, fmt.Sprintf("El #%d está cerrado sin mergear.", n)
	}
	if v.Borrador {
		return CodigoTodaviaNo, fmt.Sprintf("Todavía no: el #%d es un borrador.", n)
	}
	for _, c := range v.Checks {
		if c.Conclusion == "FAILURE" || c.Conclusion == "CANCELLED" || c.Conclusion == "TIMED_OUT" || c.Contexto == "FAILURE" || c.Contexto == "ERROR" {
			return 1, fmt.Sprintf("El CI del #%d falló: no se mergea.", n)
		}
	}
	for _, c := range v.Checks {
		if (c.Estado != "" && c.Estado != "COMPLETED") || c.Contexto == "PENDING" || c.Contexto == "EXPECTED" {
			return CodigoTodaviaNo, fmt.Sprintf("Todavía no: el CI del #%d sigue corriendo.", n)
		}
	}
	if len(v.Checks) == 0 && v.ConWorkflows {
		return CodigoTodaviaNo, fmt.Sprintf("Todavía no: el CI del #%d no arrancó.", n)
	}
	return -1, ""
}

// MergearPR es «qdd consola mergear-pr N»: mergea el PR con un commit de merge si su CI está verde.
func MergearPR(root string, n int, salida io.Writer) int {
	return mergearPRCon(ghReal, root, n, salida)
}

func mergearPRCon(correr gh, root string, n int, salida io.Writer) int {
	num := strconv.Itoa(n)
	b, err := correr(root, "pr", "view", num, "--json", "state,isDraft,statusCheckRollup")
	if err != nil {
		fmt.Fprintf(salida, "No se pudo leer el #%d con gh: %v\n", n, err)
		return 1
	}
	var v VistaPR
	if err := json.Unmarshal(b, &v); err != nil {
		fmt.Fprintf(salida, "gh respondió algo ilegible sobre el #%d: %v\n", n, err)
		return 1
	}
	v.ConWorkflows = exists(filepath.Join(root, ".github", "workflows"))
	if codigo, dice := veredictoPR(n, v); codigo >= 0 {
		fmt.Fprintln(salida, dice)
		return codigo
	}
	if _, err := correr(root, "pr", "merge", num, "--merge"); err != nil {
		fmt.Fprintf(salida, "No se pudo mergear el #%d: %v\n", n, err)
		return 1
	}
	fmt.Fprintf(salida, "#%d mergeado.\n", n)
	return 0
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
