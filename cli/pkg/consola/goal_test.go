package consola

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/modules"
)

// Rule: Prohibido usar 'else'
// Modo goal (decisión de la persona, 2026-11-04): en un proyecto con el modo activo, qdd elige solo lo recomendado y
// aprueba pedidos y fichas; con 2 min de silencio le recuerda a la IA lo que queda; tras 3 recordatorios sin avance real
// se detiene y avisa. Lo activa solo la persona. Las preguntas de diseño de un blueprint y los permisos de Claude Code
// siempre la esperan.

func conGoal(t *testing.T) string {
	t.Helper()
	root := proyectoConTareas(t)
	conSesionAbierta(t, root)
	if err := ActivarGoal(root, "panel", enCola); err != nil {
		t.Fatal(err)
	}
	return root
}

func silencio(sesion time.Duration) EntradaGoal {
	return EntradaGoal{Sesion: "s1", Silencio: sesion}
}

func cicloGoal(t *testing.T, root string, in EntradaGoal, now time.Time) ResultadoGoal {
	t.Helper()
	r, err := CicloGoal(root, "true", in, now)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func preguntaDe(t *testing.T, root string, opciones ...string) string {
	t.Helper()
	var o []string
	for _, x := range opciones {
		o = append(o, `{"label":"`+x+`","description":"d"}`)
	}
	entrada := `{"questions":[{"question":"¿Cuál va?","header":"H","multiSelect":false,"options":[` + strings.Join(o, ",") + `]}]}`
	p, err := RegistrarPregunta(root, NuevaPregunta{Sesion: "s1", Herramienta: herramientaPreguntar, Entrada: []byte(entrada)}, enCola)
	if err != nil {
		t.Fatal(err)
	}
	return p.ID
}

func TestGoal_SinActivarNoHaceNada(t *testing.T) {
	root := proyectoConTareas(t)
	id := preguntaDe(t, root, "Sí (Recommended)", "No")
	r := cicloGoal(t, root, silencio(time.Hour), enCola)
	if r.Mensaje != "" || len(r.Elegidas) != 0 {
		t.Fatalf("apagado no recuerda ni elige: %+v", r)
	}
	if p, _ := PreguntaDe(root, id, enCola); p.Estado != PreguntaPendiente {
		t.Fatalf("la pregunta sigue esperando: %+v", p)
	}
	if g := VistaGoal(root, enCola); g.Activo || g.Estado != "" {
		t.Fatalf("el panel dice que se ofrece y está apagado: %+v", g)
	}
}

func TestGoal_ActivarExigeTareasAbiertas(t *testing.T) {
	root := proyectoConCola(t)
	if err := ActivarGoal(root, "panel", enCola); err == nil {
		t.Fatal("sin tablero no hay nada que terminar")
	}
}

func TestGoal_EligeLaOpcionRecomendadaYLoRegistra(t *testing.T) {
	root := conGoal(t)
	id := preguntaDe(t, root, "Otra cosa", "Mejor (Recommended)")
	r := cicloGoal(t, root, silencio(0), enCola)
	p, _ := PreguntaDe(root, id, enCola)
	if p.Estado != PreguntaRespondida || p.Lugar != LugarGoal || p.Respuesta["q1"].Opciones[0] != p.Preguntas[0].Opciones[1].Valor {
		t.Fatalf("responde la recomendada, a nombre del modo: %+v", p)
	}
	if len(r.Elegidas) != 1 || r.Elegidas[0].Tipo != "pregunta" || r.Elegidas[0].Eleccion != "Mejor (Recommended)" {
		t.Fatalf("queda registrada: %+v", r.Elegidas)
	}
	if g := VistaGoal(root, enCola); len(g.Elegidas) != 1 || g.Elegidas[0].Eleccion != "Mejor (Recommended)" {
		t.Fatalf("y visible en el panel: %+v", g.Elegidas)
	}
}

func TestGoal_ReconoceLaRecomendadaEnEspanolYSoloComoSufijo(t *testing.T) {
	for etiqueta, recomendada := range map[string]bool{
		"Sí (Recommended)": true, "Sí (Recomendado)": true, "Sí (recomendada)": true,
		"(Recommended) por delante": false, "No Recommended": false, "Sí": false,
	} {
		if esRecomendada(etiqueta) != recomendada {
			t.Errorf("%q: recomendada=%v", etiqueta, !recomendada)
		}
	}
}

func TestGoal_NuncaEligeSinRecomendadaNiUnPermiso(t *testing.T) {
	root := conGoal(t)
	sin := preguntaDe(t, root, "Una", "Otra")
	r := cicloGoal(t, root, silencio(0), enCola)
	if p, _ := PreguntaDe(root, sin, enCola); p.Estado != PreguntaPendiente || len(r.Elegidas) != 0 {
		t.Fatalf("sin opción recomendada espera a la persona: %+v %+v", p, r)
	}
	if g := VistaGoal(root, enCola); g.Estado != GoalPausado || g.Motivo != MotivoEsperaPersona || r.Aviso == "" {
		t.Fatalf("se pausa y avisa: %+v %q", g, r.Aviso)
	}
	if r := cicloGoal(t, root, silencio(0), enCola); r.Aviso != "" {
		t.Fatalf("avisa una sola vez: %q", r.Aviso)
	}
	if _, err := CerrarPregunta(root, sin, PreguntaRespondida, "", enCola); err != nil {
		t.Fatal(err)
	}
	cicloGoal(t, root, silencio(0), enCola)
	if g := VistaGoal(root, enCola); g.Estado != GoalTrabajando {
		t.Fatalf("resuelta, sigue trabajando: %+v", g)
	}

	permiso, err := RegistrarPregunta(root, NuevaPregunta{Sesion: "s1", Herramienta: "Bash", Entrada: []byte(`{"command":"rm x (Recommended)"}`)}, enCola)
	if err != nil {
		t.Fatal(err)
	}
	cicloGoal(t, root, silencio(0), enCola)
	if p, _ := PreguntaDe(root, permiso.ID, enCola); p.Estado != PreguntaPendiente {
		t.Fatalf("un permiso de Claude Code nunca se elige solo: %+v", p)
	}
}

func TestGoal_UnaPreguntaDeDisenoDeBlueprintNuncaSeElige(t *testing.T) {
	root := conGoal(t)
	preguntasDeDiseno = func(string) map[string]bool { return map[string]bool{normalizarPregunta("¿Cuál va?"): true} }
	t.Cleanup(func() { preguntasDeDiseno = disenoReal })
	id := preguntaDe(t, root, "Sí (Recommended)", "No")
	cicloGoal(t, root, silencio(0), enCola)
	if p, _ := PreguntaDe(root, id, enCola); p.Estado != PreguntaPendiente {
		t.Fatalf("es de la persona, aunque traiga recomendada: %+v", p)
	}
}

func TestGoal_LasEsperasDelEstadoNoSeResponden(t *testing.T) {
	root := conGoal(t)
	conEspera(t, root, "¿Resend o SES?")
	r := cicloGoal(t, root, silencio(0), enCola)
	if len(r.Elegidas) != 0 {
		t.Fatalf("un ⏳ no tiene opciones: %+v", r.Elegidas)
	}
	if g := VistaGoal(root, enCola); g.Estado != GoalTrabajando {
		t.Fatalf("un ⏳ no pausa: la IA puede seguir con otra tarea: %+v", g)
	}
}

func TestGoal_CorreLosPedidosDeLaIAYNoLosDeUnPR(t *testing.T) {
	root := conGoal(t)
	marca := filepath.Join(t.TempDir(), "corrio")
	bin := os.Args[0]
	pd, err := Pedir(root, "touch "+marca, "Desplegar", root, enCola)
	if err != nil {
		t.Fatal(err)
	}
	if err := OfrecerPRs(root, []PR{{Numero: 7, Titulo: "de otra sesión"}}, enCola); err != nil {
		t.Fatal(err)
	}
	r, err := CicloGoal(root, bin, silencio(0), enCola)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Elegidas) != 1 || r.Elegidas[0].Tipo != "pedido" || r.Elegidas[0].Etiqueta != "Desplegar" {
		t.Fatalf("el pedido de la IA, registrado sin el comando: %+v", r.Elegidas)
	}
	for _, p := range PedidosA(root, enCola) {
		if p.ID == pd.ID && p.Estado == EstadoPendiente {
			t.Fatalf("quedó aceptado: %+v", p)
		}
		if p.PR != 0 && p.Estado != EstadoPendiente {
			t.Fatalf("un PR abierto no es un pedido de la IA y no se mergea solo: %+v", p)
		}
	}
}

func TestGoal_ApruebaLasFichasEnBorrador(t *testing.T) {
	root := conGoal(t)
	m := modules.Module{
		ID: "nueva", Nombre: "Nueva", Estado: modules.EstadoBorrador, Rutas: []string{"."},
		ObjetivoOperativo:   modules.ObjetivoOperativo{Problema: "p", CriterioDeExito: "c"},
		ObjetivoTecnologico: modules.ObjetivoTecnologico{Arquitectura: "a"},
		Pruebas:             modules.Pruebas{Comandos: []string{"go test ./..."}},
	}
	if _, problemas := modules.Save(root, m); len(problemas) > 0 {
		t.Fatal(problemas)
	}
	r := cicloGoal(t, root, silencio(0), enCola)
	if len(r.Elegidas) != 1 || r.Elegidas[0].Tipo != "ficha" {
		t.Fatalf("la aprueba y lo registra: %+v", r.Elegidas)
	}
	if len(borradores(root)) != 0 {
		t.Fatal("no queda en borrador")
	}
}

func TestGoal_ConSilencioRecuerdaYSinSilencioNo(t *testing.T) {
	root := conGoal(t)
	if r := cicloGoal(t, root, silencio(SilencioGoal-time.Second), enCola); r.Mensaje != "" {
		t.Fatalf("antes de 2 min no molesta: %q", r.Mensaje)
	}
	if r := cicloGoal(t, root, EntradaGoal{Sesion: "s1", Silencio: time.Hour, Ocupada: true}, enCola); r.Mensaje != "" {
		t.Fatalf("si está haciendo algo no recuerda: %q", r.Mensaje)
	}
	r := cicloGoal(t, root, silencio(SilencioGoal), enCola)
	if !strings.HasPrefix(r.Mensaje, "🎯") || !strings.Contains(r.Mensaje, "Barra") {
		t.Fatalf("recuerda lo que queda, con 🎯 (nunca «!» ni «/»): %q", r.Mensaje)
	}
	if otra := cicloGoal(t, root, silencio(SilencioGoal), enCola.Add(time.Second)); otra.Mensaje != "" {
		t.Fatalf("otra terminal de la misma sesión no lo repite: %q", otra.Mensaje)
	}
}

func TestGoal_NoLeRecuerdaALaTerminalQueMiraMientrasOtraTrabaja(t *testing.T) {
	root := conGoal(t)
	cicloGoal(t, root, EntradaGoal{Sesion: "trabaja", Ocupada: true}, enCola)
	r := cicloGoal(t, root, EntradaGoal{Sesion: "mira", Silencio: time.Hour}, enCola.Add(time.Minute))
	if r.Mensaje != "" {
		t.Fatalf("otra terminal está trabajando: %q", r.Mensaje)
	}
	r = cicloGoal(t, root, EntradaGoal{Sesion: "mira", Silencio: time.Hour}, enCola.Add(3*time.Minute))
	if r.Mensaje == "" {
		t.Fatal("la otra lleva rato callada: ahora sí se recuerda")
	}
}

func TestGoal_TresRecordatoriosSinAvanceLoDetienen(t *testing.T) {
	root := conGoal(t)
	for i := 0; i < MaxRecordatoriosGoal; i++ {
		now := enCola.Add(time.Duration(i+1) * 3 * time.Minute)
		if r := cicloGoal(t, root, silencio(SilencioGoal), now); r.Mensaje == "" {
			t.Fatalf("recordatorio %d: %+v", i+1, r)
		}
	}
	if g := VistaGoal(root, enCola); g.Recordatorios != MaxRecordatoriosGoal {
		t.Fatalf("cuenta los recordatorios: %+v", g)
	}
	r := cicloGoal(t, root, silencio(SilencioGoal), enCola.Add(30*time.Minute))
	if r.Mensaje != "" || r.Aviso == "" {
		t.Fatalf("el cuarto no va: se detiene y avisa: %+v", r)
	}
	if g := VistaGoal(root, enCola); g.Estado != GoalPausado || g.Motivo != MotivoSinAvance {
		t.Fatalf("pausado sin avance: %+v", g)
	}
	if r := cicloGoal(t, root, silencio(SilencioGoal), enCola.Add(time.Hour)); r.Mensaje != "" || r.Aviso != "" {
		t.Fatalf("sigue detenido hasta que la persona lo reactive: %+v", r)
	}
}

func TestGoal_UnAvanceRealReiniciaLaCuenta(t *testing.T) {
	root := conGoal(t)
	cicloGoal(t, root, silencio(SilencioGoal), enCola.Add(3*time.Minute))
	cicloGoal(t, root, silencio(SilencioGoal), enCola.Add(6*time.Minute))
	if err := estado.Set(root, estado.Cambio{Frente: "barra", Etapa: "PR", Estado: "✅", Evidencia: "PR #9"}, enCola); err != nil {
		t.Fatal(err)
	}
	cicloGoal(t, root, silencio(SilencioGoal), enCola.Add(9*time.Minute))
	if g := VistaGoal(root, enCola); g.Recordatorios > 1 {
		t.Fatalf("cambió el tablero: la cuenta vuelve a empezar: %+v", g)
	}
}

func TestGoal_LaFechaDelTableroSolaNoEsAvance(t *testing.T) {
	root := conGoal(t)
	antes := huellaGoal(root)
	e, err := estado.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	e.Actualizado = "2030-01-01T00:00:00Z"
	if err := estado.Save(root, e); err != nil {
		t.Fatal(err)
	}
	if huellaGoal(root) != antes {
		t.Fatal("reescribir el tablero sin cambiar nada no es avance")
	}
}

func TestGoal_LoQueEscribeQDDSoloNoEsAvance(t *testing.T) {
	root := conGoal(t)
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}, {"-c", "user.email=a@b", "-c", "user.name=a", "commit", "-qm", "x"}} {
		c := exec.Command("git", args...)
		c.Dir = root
		if b, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, b)
		}
	}
	antes := huellaGoal(root)
	metricas := filepath.Join(root, ".qdd", "project", "metrics")
	if err := os.MkdirAll(metricas, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(metricas, "m.json"), []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if huellaGoal(root) != antes {
		t.Fatal("las métricas que escribe qdd no son avance")
	}
	if err := os.WriteFile(filepath.Join(root, "codigo.go"), []byte("package x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if huellaGoal(root) == antes {
		t.Fatal("un archivo nuevo del proyecto sí es avance")
	}
}

func TestGoal_SinTareasTerminaYAvisa(t *testing.T) {
	root := conGoal(t)
	if err := estado.Cerrar(root, "barra", "listo", false, enCola); err != nil {
		t.Fatal(err)
	}
	if err := estado.Cerrar(root, "otra", "listo", false, enCola); err != nil {
		t.Fatal(err)
	}
	if r := cicloGoal(t, root, silencio(0), enCola); r.Aviso != "" {
		t.Fatalf("con la IA aún activa no da por terminado: %+v", r)
	}
	r := cicloGoal(t, root, silencio(SilencioGoal), enCola)
	if r.Mensaje != "" || r.Aviso == "" {
		t.Fatalf("termina y avisa, sin recordar: %+v", r)
	}
	if g := VistaGoal(root, enCola); g.Estado != GoalTerminado || g.Motivo != "" {
		t.Fatalf("terminado: %+v", g)
	}
}

func TestGoal_SinConsolaSePausaEnLaVistaYVuelveSola(t *testing.T) {
	root := conGoal(t)
	if g := VistaGoal(root, enCola); g.Estado != GoalTrabajando {
		t.Fatalf("con sesión, trabajando: %+v", g)
	}
	despues := enCola.Add(VidaSesion + time.Second)
	if g := VistaGoal(root, despues); g.Estado != GoalPausado || g.Motivo != MotivoSinConsola {
		t.Fatalf("sin sesión, pausado: %+v", g)
	}
	if err := LatirSesion(root, "s1", despues); err != nil {
		t.Fatal(err)
	}
	if g := VistaGoal(root, despues); g.Estado != GoalTrabajando {
		t.Fatalf("vuelve la sesión, sigue: %+v", g)
	}
}

func TestGoal_ElLimiteAgotadoLoPausaYAlReiniciarseSigue(t *testing.T) {
	root := conGoal(t)
	reinicia := enCola.Add(time.Hour)
	GuardarLimites([]byte(`{"rate_limits":{"five_hour":{"used_percentage":100,"resets_at":"`+reinicia.Format(time.RFC3339)+`"}}}`), enCola)
	preguntaDe(t, root, "Sí (Recommended)", "No")
	r := cicloGoal(t, root, silencio(time.Hour), enCola)
	if r.Mensaje != "" || r.Aviso == "" || len(r.Elegidas) != 0 {
		t.Fatalf("sin límite no recuerda ni elige, y avisa: %+v", r)
	}
	if g := VistaGoal(root, enCola); g.Motivo != MotivoSinLimite {
		t.Fatalf("pausado sin límite: %+v", g)
	}
	luego := reinicia.Add(time.Minute)
	if err := LatirSesion(root, "s1", luego); err != nil {
		t.Fatal(err)
	}
	cicloGoal(t, root, silencio(0), luego)
	if g := VistaGoal(root, luego); g.Estado != GoalTrabajando {
		t.Fatalf("reiniciado el límite, sigue: %+v", g)
	}
}

func TestGoal_ReactivarEmpiezaDeCeroYDesactivarLoApaga(t *testing.T) {
	root := conGoal(t)
	cicloGoal(t, root, silencio(SilencioGoal), enCola.Add(3*time.Minute))
	if err := ActivarGoal(root, "panel", enCola.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if g := VistaGoal(root, enCola); g.Recordatorios != 0 || g.Estado != GoalTrabajando {
		t.Fatalf("reactivarlo reinicia la cuenta: %+v", g)
	}
	if err := DesactivarGoal(root, enCola); err != nil {
		t.Fatal(err)
	}
	if g := VistaGoal(root, enCola); g.Activo {
		t.Fatalf("apagado: %+v", g)
	}
}

func TestGoal_GuardaSoloLasUltimasElegidas(t *testing.T) {
	root := conGoal(t)
	for i := 0; i < MaxElegidasGoal+5; i++ {
		preguntaDe(t, root, "Sí (Recommended)", "No")
		cicloGoal(t, root, silencio(0), enCola.Add(time.Duration(i)*time.Second))
	}
	g := VistaGoal(root, enCola)
	if len(g.Elegidas) != MaxElegidasGoal || !g.Elegidas[0].En.After(g.Elegidas[1].En) {
		t.Fatalf("las últimas, la más reciente primero: %d", len(g.Elegidas))
	}
}

func TestGoal_ElPanelSiempreDiceSiSeOfrece(t *testing.T) {
	root := proyectoConTareas(t)
	d := panel(root, enCola)
	if d.Goal == nil || d.Goal.Activo {
		t.Fatalf("el panel trae goal con activo false: %+v", d.Goal)
	}
	if _, err := BuscarAccion(d.Acciones, AccionGoalActivar); err != nil {
		t.Fatalf("con tablero se ofrece activar: %v", err)
	}
	if err := ActivarGoal(root, "panel", enCola); err != nil {
		t.Fatal(err)
	}
	d = panel(root, enCola)
	if _, err := BuscarAccion(d.Acciones, AccionGoalDesactivar); err != nil || d.Goal == nil || !d.Goal.Activo {
		t.Fatalf("activo se ofrece desactivar: %v %+v", err, d.Goal)
	}
	if a, err := BuscarAccion(panel(proyectoConTareas(t), enCola).Acciones, AccionGoalActivar); err != nil || !a.Confirmar {
		t.Fatalf("activar pasa por la confirmación de la persona: %+v %v", a, err)
	}
}

func TestGoal_UnArchivoIlegibleLoVeDoctorYSeAparta(t *testing.T) {
	root := conGoal(t)
	ruta, err := rutaGoal(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte("{roto"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := RevisarGoal(root); err == nil {
		t.Fatal("doctor tiene que verlo")
	}
	if v := VistaGoal(root, enCola); v.Activo {
		t.Fatalf("ilegible es apagado, nunca activo: %+v", v)
	}
	if err := ApartarGoal(root); err != nil {
		t.Fatal(err)
	}
	if activo, err := RevisarGoal(root); err != nil || activo {
		t.Fatalf("apartado queda apagado y legible: %v %v", activo, err)
	}
}
