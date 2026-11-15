package consola

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// Sprint 8 (MEJ-057, MEJ-026): toda pregunta a la persona, de cualquier tipo, aparece en «▶ Para ti» mientras sigue
// abierta y sale cuando se responde o deja de esperarse. «Para ti» dibuja lo que trae turno en 'qdd consola panel
// --json --con pregunta' (RFC 0019), y QDD Móvil lee el mismo JSON: si aquí lleva turno, está en los dos.

// conPreguntasDelPanel es lo que pide el panel de Claude Code (mod qdd-tareas, OPCIONALES).
var conPreguntasDelPanel = []string{TipoAtender, TipoPregunta}

// tipoDePregunta deja abierta una pregunta de un tipo y devuelve el id de su acción en el panel.
type tipoDePregunta struct {
	nombre string
	dejar  func(t *testing.T, root string, now time.Time) string
}

func registrada(t *testing.T, root string, n NuevaPregunta, now time.Time) string {
	t.Helper()
	p, err := RegistrarPregunta(root, n, now)
	if err != nil {
		t.Fatal(err)
	}
	return prefijoPregunta + p.ID
}

func askDePrueba(texto string) json.RawMessage {
	return json.RawMessage(`{"questions":[{"question":"` + texto + `","header":"Base","multiSelect":false,"options":[{"label":"Postgres","description":""},{"label":"SQLite","description":""}]}]}`)
}

// tiposDePregunta son todas las formas en que qdd o la IA le preguntan algo a la persona. Un tipo u origen nuevo
// tiene que sumarse aquí (TestParaTi_ContratoCubreCadaTipoYOrigen).
func tiposDePregunta() []tipoDePregunta {
	return []tipoDePregunta{
		{"AskUserQuestion de la IA", func(t *testing.T, root string, now time.Time) string {
			return registrada(t, root, NuevaPregunta{Sesion: "s1", Herramienta: herramientaPreguntar, Entrada: askDePrueba("¿Qué base usamos?")}, now)
		}},
		{"permiso de herramienta", func(t *testing.T, root string, now time.Time) string {
			return registrada(t, root, NuevaPregunta{Sesion: "s1", Herramienta: "Bash", Entrada: json.RawMessage(`{"command":"git push"}`)}, now)
		}},
		{"pregunta de un comando qdd (origen qdd)", func(t *testing.T, root string, now time.Time) string {
			return registrada(t, root, NuevaPregunta{Origen: OrigenQDD, Etiqueta: "qdd blueprint apply auth", Preguntas: []ItemPregunta{{
				ID: "q1", Encabezado: "auth", Texto: "¿Multi-tenant?",
				Opciones: []OpcionPregunta{{Valor: "a", Etiqueta: "Sí"}, {Valor: "b", Etiqueta: "No"}},
			}}}, now)
		}},
		{"⏳ de la persona escrito como pregunta", func(t *testing.T, root string, now time.Time) string {
			escribirArchivo(t, filepath.Join(root, estado.Ruta), estadoBaseEncima)
			if err := estado.Set(root, estado.Cambio{Frente: "base", Etapa: "Código", Estado: "⏳", Bloqueo: "¿Sigo con la migración?", DeQuien: "la persona"}, now); err != nil {
				t.Fatal(err)
			}
			for _, p := range PreguntasA(root, now) {
				if p.Origen == OrigenEstado {
					return prefijoPregunta + p.ID
				}
			}
			t.Fatal("el ⏳ de la persona no llegó a la cola de preguntas")
			return ""
		}},
		{"pedido de qdd_consola_pedir", func(t *testing.T, root string, now time.Time) string {
			p, err := Pedir(root, "gh pr merge 1", "Mergear el PR 1", "", now)
			if err != nil {
				t.Fatal(err)
			}
			return prefijoPedido + p.ID
		}},
		{"ficha en borrador", func(t *testing.T, root string, now time.Time) string {
			return prefijoAprobar + fichaEnBorrador(t, root)
		}},
	}
}

// fichaEnBorrador deja una ficha en borrador en un proyecto que ya existe.
func fichaEnBorrador(t *testing.T, root string) string {
	t.Helper()
	otro := proyectoConFichas(t, nil, []string{"pagos"})
	if err := os.CopyFS(filepath.Join(root, ".qdd"), os.DirFS(filepath.Join(otro, ".qdd"))); err != nil {
		t.Fatal(err)
	}
	return "pagos"
}

func proyectoParaTi(t *testing.T) string {
	t.Helper()
	root := proyectoConCola(t)
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func accionDelPanel(root, id string, now time.Time) (Accion, bool) {
	for _, a := range Panel(root, now, conPreguntasDelPanel...).Acciones {
		if a.ID == id {
			return a, true
		}
	}
	return Accion{}, false
}

func TestParaTi_CadaTipoDePreguntaLlevaTurnoEnElPanel(t *testing.T) {
	for _, tipo := range tiposDePregunta() {
		t.Run(tipo.nombre, func(t *testing.T) {
			root := proyectoParaTi(t)
			now := time.Now()
			id := tipo.dejar(t, root, now)
			a, ok := accionDelPanel(root, id, now)
			if !ok {
				t.Fatalf("%s no llega al panel: %s", tipo.nombre, id)
			}
			if a.Turno == 0 {
				t.Fatalf("%s llega sin turno: «Para ti» no la dibuja ni QDD Móvil la cuenta: %+v", tipo.nombre, a)
			}
		})
	}
}

func TestParaTi_LosTiposJuntosLlevanTurnosDistintos(t *testing.T) {
	root := proyectoParaTi(t)
	now := time.Now()
	turnos := map[int]string{}
	for _, tipo := range tiposDePregunta() {
		id := tipo.dejar(t, root, now)
		a, ok := accionDelPanel(root, id, now)
		if !ok || a.Turno == 0 {
			t.Fatalf("%s no lleva turno junto a los demás: %+v", tipo.nombre, a)
		}
		if otro, repetido := turnos[a.Turno]; repetido {
			t.Fatalf("%s y %s comparten el turno %d", tipo.nombre, otro, a.Turno)
		}
		turnos[a.Turno] = tipo.nombre
	}
}

// QDD Móvil lee el mismo panel que el PC, con los opcionales del teléfono: ve las mismas preguntas con los mismos números.
func TestParaTi_ElTelefonoVeLasMismasPreguntasConElMismoTurno(t *testing.T) {
	root := proyectoParaTi(t)
	now := time.Now()
	var ids []string
	for _, tipo := range tiposDePregunta() {
		ids = append(ids, tipo.dejar(t, root, now))
	}
	telefono := map[string]int{}
	for _, a := range Panel(root, now, TipoAtender, TipoPregunta, ConTelefono).Acciones {
		telefono[a.ID] = a.Turno
	}
	for _, id := range ids {
		a, _ := accionDelPanel(root, id, now)
		if telefono[id] == 0 || telefono[id] != a.Turno {
			t.Fatalf("%s: el panel le da %d y el teléfono %d", id, a.Turno, telefono[id])
		}
	}
}

// noPreguntanALaPersona son los tipos y orígenes de acción que no le preguntan nada a la persona, con el motivo.
var noPreguntanALaPersona = map[string]string{
	"TipoAbrir":      "abrir la carpeta: lo ofrece qdd, no espera respuesta",
	"TipoComando":    "herramientas del proyecto (auditar, revisar): las ofrece qdd",
	"TipoDescartar":  "quitar una tarea: la ofrece qdd en su fila",
	"TipoYaLoVi":     "«✓ Ya lo vi»: lo ofrece qdd en su fila",
	"TipoAtender":    "llevar un ⏳ al chat: es el botón de la pregunta, no otra pregunta",
	"OrigenPanel":    "un logro que salió del panel: no espera a nadie",
	"OrigenTelefono": "lo mandado desde el teléfono: ya lo decidió la persona",
}

// cubiertosPorTiposDePregunta son los tipos y orígenes que tiposDePregunta() deja abiertos.
var cubiertosPorTiposDePregunta = map[string]string{
	"TipoPregunta":  "AskUserQuestion (también las de diseño de un blueprint), permiso, comando qdd y ⏳",
	"TipoPedido":    "pedido de qdd_consola_pedir",
	"TipoAprobar":   "ficha en borrador",
	"OrigenIA":      "AskUserQuestion de la IA",
	"OrigenPermiso": "permiso de herramienta",
	"OrigenQDD":     "pregunta de un comando qdd",
	"OrigenEstado":  "⏳ de la persona",
}

// El contrato: cada constante Tipo… u Origen… del paquete está cubierta por una prueba de «Para ti» o dice por qué no
// pregunta nada. Un tipo nuevo sin su fila hace fallar la suite.
func TestParaTi_ContratoCubreCadaTipoYOrigen(t *testing.T) {
	var sinCubrir []string
	for _, nombre := range constantesDeTipoYOrigen(t) {
		_, cubierto := cubiertosPorTiposDePregunta[nombre]
		_, explicado := noPreguntanALaPersona[nombre]
		if cubierto || explicado {
			continue
		}
		sinCubrir = append(sinCubrir, nombre)
	}
	if len(sinCubrir) > 0 {
		t.Fatalf("tipos u orígenes sin prueba de «Para ti» ni motivo para no tenerla: %v. Súmalos a tiposDePregunta() o a noPreguntanALaPersona", sinCubrir)
	}
}

func constantesDeTipoYOrigen(t *testing.T) []string {
	t.Helper()
	fset := token.NewFileSet()
	archivos, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, ruta := range archivos {
		if strings.HasSuffix(ruta, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, ruta, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, constantesDe(f)...)
	}
	sort.Strings(out)
	return out
}

func constantesDe(f *ast.File) []string {
	var out []string
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok || g.Tok != token.CONST {
			continue
		}
		for _, s := range g.Specs {
			for _, n := range s.(*ast.ValueSpec).Names {
				if !esTipoUOrigen(n.Name) {
					continue
				}
				out = append(out, n.Name)
			}
		}
	}
	return out
}

func esTipoUOrigen(nombre string) bool {
	for _, prefijo := range []string{"Tipo", "Origen"} {
		resto := strings.TrimPrefix(nombre, prefijo)
		if resto != nombre && resto != "" && strings.ToUpper(resto[:1]) == resto[:1] {
			return true
		}
	}
	return false
}

// Respondida en la terminal (el mod la cierra en el PC), sale de «Para ti» en la lectura siguiente: no queda fantasma.
func TestParaTi_RespondidaEnLaTerminalSaleDeParaTi(t *testing.T) {
	root := proyectoParaTi(t)
	now := time.Now()
	id := tiposDePregunta()[0].dejar(t, root, now)
	if _, err := CerrarPregunta(root, id, PreguntaRespondida, "", now); err != nil {
		t.Fatal(err)
	}
	a, _ := accionDelPanel(root, id, now.Add(time.Second))
	if a.Turno != 0 {
		t.Fatalf("respondida en la terminal sigue en «Para ti»: %+v", a)
	}
}

// Si la sesión se cierra con la pregunta abierta, vence y sale de «Para ti».
func TestParaTi_LaSesionCerradaVenceSusPreguntas(t *testing.T) {
	root := proyectoParaTi(t)
	now := time.Now()
	id := tiposDePregunta()[0].dejar(t, root, now)
	if n, err := VencerSesion(root, "s1", "se cerró Claude Code", now); err != nil || n != 1 {
		t.Fatalf("vence la de su sesión: %d %v", n, err)
	}
	a, _ := accionDelPanel(root, id, now.Add(time.Second))
	if a.Turno != 0 {
		t.Fatalf("vencida sigue en «Para ti»: %+v", a)
	}
}

// Mientras su sesión la sigue esperando (latido antes de LatidoPregunta), no vence aunque pase más que el latido.
func TestParaTi_ConLatidoNoVenceAunquePaseElTiempo(t *testing.T) {
	root := proyectoParaTi(t)
	now := time.Now()
	id := tiposDePregunta()[0].dejar(t, root, now)
	for paso := 1; paso <= 3; paso++ {
		if _, err := LatirPregunta(root, id, now.Add(time.Duration(paso)*LatidoPregunta*2/3)); err != nil {
			t.Fatal(err)
		}
	}
	a, _ := accionDelPanel(root, id, now.Add(2*LatidoPregunta+time.Second))
	if a.Turno == 0 {
		t.Fatalf("una pregunta que su sesión sigue esperando no vence: %+v", a)
	}
}

// Una cola de preguntas ilegible se aparta y la pregunta nueva aparece igual.
func TestParaTi_UnaColaIlegibleNoTapaLaPreguntaNueva(t *testing.T) {
	root := proyectoParaTi(t)
	ruta, err := rutaPreguntas(root)
	if err != nil {
		t.Fatal(err)
	}
	escribirArchivo(t, ruta, "{esto no es json")
	now := time.Now()
	id := tiposDePregunta()[0].dejar(t, root, now)
	if a, _ := accionDelPanel(root, id, now); a.Turno == 0 {
		t.Fatalf("con la cola ilegible la pregunta nueva no aparece: %+v", a)
	}
	if _, err := os.Stat(ruta + ".ilegible"); err != nil {
		t.Fatalf("la cola ilegible se aparta, no se pierde: %v", err)
	}
}

// Dos sesiones del mismo proyecto preguntan a la vez: las dos aparecen, cada una con su turno.
func TestParaTi_DosSesionesPreguntanALaVez(t *testing.T) {
	root := proyectoParaTi(t)
	now := time.Now()
	uno := registrada(t, root, NuevaPregunta{Sesion: "s1", Herramienta: herramientaPreguntar, Entrada: askDePrueba("¿Uno?")}, now)
	dos := registrada(t, root, NuevaPregunta{Sesion: "s2", Herramienta: herramientaPreguntar, Entrada: askDePrueba("¿Dos?")}, now)
	a, _ := accionDelPanel(root, uno, now)
	b, _ := accionDelPanel(root, dos, now)
	if a.Turno == 0 || b.Turno == 0 || a.Turno == b.Turno {
		t.Fatalf("cada sesión con su turno: %d y %d", a.Turno, b.Turno)
	}
}

// Una pendiente nunca sale de la cola por el tope mientras haya cerradas que puedan salir antes.
func TestParaTi_ElTopeDeLaColaNoSacaUnaPendiente(t *testing.T) {
	root := proyectoParaTi(t)
	now := time.Now()
	abierta := tiposDePregunta()[0].dejar(t, root, now)
	for i := range MaxPreguntasCola + 5 {
		id := registrada(t, root, NuevaPregunta{Sesion: "s2", Herramienta: "Bash", Entrada: json.RawMessage(`{"command":"ls"}`)}, now.Add(time.Duration(i+1)*time.Millisecond))
		if _, err := CerrarPregunta(root, id, PreguntaRespondida, "", now); err != nil {
			t.Fatal(err)
		}
	}
	if a, _ := accionDelPanel(root, abierta, now); a.Turno == 0 {
		t.Fatalf("las cerradas sacaron a la pendiente de la cola: %+v", a)
	}
}
