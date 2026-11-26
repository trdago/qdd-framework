package consola

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/blueprints"
)

// Rule: Prohibido usar 'else'
// Las preguntas de diseño de un blueprint son de la persona, también con el modo goal activo. Hasta aquí el modo solo
// se probaba con preguntasDeDiseno reemplazada; estas pruebas usan el catálogo real: un blueprint importado de verdad
// (repositorio git con su versión, lockfile y copia central en QDD_HOME) y blueprints.PreguntasDeDiseno sin stub.

const blueprintConDiseno = `id: auth
nombre: Usuarios y acceso
version: 1.0.0
organizacion: ACME
preguntas:
  - id: identificador
    pregunta: ¿Con qué inicia sesión el usuario?
    opciones:
      - valor: correo
      - valor: rut
  - id: recuperacion
    pregunta: ¿Por qué canal se recupera la contraseña?
    opciones:
      - valor: correo
      - valor: whatsapp
ficha:
  nombre: Usuarios, roles y permisos
  rutas: [internal/auth]
  objetivo_operativo:
    problema: Controlar quién entra y qué puede hacer.
    criterio_de_exito: Ningún acceso sin permiso explícito.
  objetivo_tecnologico:
    arquitectura: JWT de acceso corto + refresh rotativo.
    invariantes: ["Autorización por permiso, nunca por nombre de rol"]
conformidad:
  - id: AUTH-01
    titulo: Credenciales inválidas
    cuando: se inicia sesión con una contraseña incorrecta
    entonces: la respuesta no revela si la cuenta existe
`

// conBlueprintImportado publica el blueprint en un repositorio git temporal y lo importa al proyecto (lockfile y copia
// central en el QDD_HOME de la prueba).
func conBlueprintImportado(t *testing.T, root string) {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "auth")
	escribirArchivo(t, filepath.Join(repo, blueprints.ManifestName), blueprintConDiseno)
	gitEn(t, repo, "init", "-q")
	gitEn(t, repo, "add", ".")
	gitEn(t, repo, "commit", "-q", "-m", "v1")
	gitEn(t, repo, "tag", "v1.0.0")
	if _, problemas := blueprints.Add(blueprints.AddInput{Root: root, Origen: repo}); len(problemas) > 0 {
		t.Fatalf("no se pudo importar el blueprint: %v", problemas)
	}
}

// preguntaConTexto registra un AskUserQuestion con ese texto y una opción recomendada.
func preguntaConTexto(t *testing.T, root, texto string) string {
	t.Helper()
	entrada, _ := json.Marshal(map[string]any{"questions": []map[string]any{{
		"question": texto, "header": "Diseño", "multiSelect": false,
		"options": []map[string]string{{"label": "WhatsApp (Recommended)", "description": "d"}, {"label": "Correo", "description": "d"}},
	}}})
	p, err := RegistrarPregunta(root, NuevaPregunta{Sesion: "s1", Herramienta: herramientaPreguntar, Entrada: entrada}, enCola)
	if err != nil {
		t.Fatal(err)
	}
	return p.ID
}

// contrato: goal#bb2a2f53
// contrato: blueprints#7b4242ab
func TestGoal_UnaPreguntaDeDisenoDeUnBlueprintImportadoEsperaALaPersona(t *testing.T) {
	root := conGoal(t)
	conBlueprintImportado(t, root)
	for _, texto := range []string{
		"¿Por qué canal se recupera la contraseña?",
		"¿POR QUÉ CANAL se recupera   la contraseña?",
		"  ¿Por qué canal se recupera la\ncontraseña?  ",
	} {
		id := preguntaConTexto(t, root, texto)
		r := cicloGoal(t, root, silencio(0), enCola)
		if p, _ := PreguntaDe(root, id, enCola); p.Estado != PreguntaPendiente || len(r.Elegidas) != 0 {
			t.Fatalf("%q es una pregunta de diseño: la responde la persona aunque traiga recomendada: %+v %+v", texto, p, r.Elegidas)
		}
		if g := VistaGoal(root, enCola); g.Estado != GoalPausado || g.Motivo != MotivoEsperaPersona {
			t.Fatalf("el modo queda en pausa esperando a la persona: %+v", g)
		}
		if _, err := CerrarPregunta(root, id, PreguntaRespondida, "", enCola); err != nil {
			t.Fatal(err)
		}
	}
	otra := preguntaConTexto(t, root, "¿Uso WhatsApp para el aviso de prueba?")
	cicloGoal(t, root, silencio(0), enCola)
	if p, _ := PreguntaDe(root, otra, enCola); p.Estado != PreguntaRespondida || p.Lugar != LugarGoal {
		t.Fatalf("una pregunta que no es del blueprint sí la elige el modo: %+v", p)
	}
}

func TestGoal_ElCatalogoRealTraeLasPreguntasDelBlueprint(t *testing.T) {
	root := proyectoConCola(t)
	if got := disenoReal(root); len(got) != 0 {
		t.Fatalf("sin blueprints no hay preguntas de diseño: %v", got)
	}
	conBlueprintImportado(t, root)
	got := disenoReal(root)
	if !got[normalizarPregunta("¿Por qué canal se recupera la contraseña?")] || !got[normalizarPregunta("¿Con qué inicia sesión el usuario?")] || len(got) != 2 {
		t.Fatalf("las dos preguntas del blueprint importado: %v", got)
	}
	if _, err := os.Stat(filepath.Join(root, blueprints.LockFile)); err != nil {
		t.Fatalf("el proyecto solo versiona el lockfile: %v", err)
	}
}

// Criterio de éxito del modo goal, de punta a punta: con el modo activo el proyecto avanza mientras puede y se detiene
// diciendo por qué, sin tomar lo que es de la persona. En un proyecto con dos tareas, una pregunta con recomendada, una
// pregunta de diseño de un blueprint importado, un pedido de la IA y uno de la persona: elige la pregunta y corre el de
// la IA (cada elección queda en Elegidas), se pausa por la de diseño, y cuando la persona la responde sigue hasta 3
// recordatorios sin avance y se detiene. La pregunta de diseño y el pedido de la persona nunca los toca.
func TestGoal_DePuntaAPuntaAvanzaYSeDetieneSinTomarLoDeLaPersona(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("comando de prueba en sh")
	}
	antes := probarScope
	t.Cleanup(func() { probarScope = antes })
	probarScope = func() bool { return false }
	root := conGoal(t)
	conBlueprintImportado(t, root)
	normal := preguntaDe(t, root, "Sigo con la barra (Recommended)", "Paro")
	diseno := preguntaConTexto(t, root, "¿Por qué canal se recupera la contraseña?")
	marca := filepath.Join(t.TempDir(), "corrio")
	deIA := mustPedir(t, root, "touch "+marca, "Desplegar", "")
	marcaPersona := filepath.Join(t.TempDir(), "admitida")
	dePersona, err := PedirEnOrden(root, "touch "+marcaPersona, "📮 Admitir MEJ-090", "", Orden{DePersona: true}, enCola)
	if err != nil {
		t.Fatal(err)
	}
	bin := os.Args[0]
	// La sesión sigue abierta todo el rato: late en cada vuelta, como la consola de verdad.
	ciclo := func(now time.Time, in EntradaGoal) ResultadoGoal {
		t.Helper()
		if err := LatirSesion(root, "s1", now); err != nil {
			t.Fatal(err)
		}
		r, err := CicloGoal(root, bin, in, now)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	loDeLaPersonaIntacto := func(paso string) {
		t.Helper()
		if p := pedidoDe(t, root, dePersona.ID, enCola); p.Estado != EstadoPendiente || p.Confirmado != nil {
			t.Fatalf("%s: el pedido de la persona sigue esperando su «s  Sí»: %+v", paso, p)
		}
	}

	r := ciclo(enCola, silencio(0))
	if len(r.Elegidas) != 2 || r.Elegidas[0].Tipo != "pregunta" || r.Elegidas[1].Tipo != "pedido" || r.Elegidas[1].Etiqueta != "Desplegar" {
		t.Fatalf("elige la pregunta con recomendada y corre el pedido de la IA, nada más: %+v", r.Elegidas)
	}
	esperarEstado(t, root, deIA.ID, EstadoHecho)
	if p, _ := PreguntaDe(root, normal, enCola); p.Estado != PreguntaRespondida || p.Lugar != LugarGoal {
		t.Fatalf("la pregunta con recomendada la respondió el modo, a su nombre: %+v", p)
	}
	if g := VistaGoal(root, enCola); g.Estado != GoalPausado || g.Motivo != MotivoEsperaPersona || r.Aviso == "" {
		t.Fatalf("se pausa diciendo que espera a la persona: %+v %q", g, r.Aviso)
	}
	for i := 1; i <= 5; i++ {
		now := enCola.Add(time.Duration(i) * 50 * time.Second)
		// La sesión sigue esperando la respuesta (qdd consola pregunta esperar, menos de LatidoPregunta entre una y otra).
		if _, err := LatirPregunta(root, diseno, now); err != nil {
			t.Fatal(err)
		}
		if r := ciclo(now, silencio(SilencioGoal)); r.Mensaje != "" || len(r.Elegidas) != 0 {
			t.Fatalf("vuelta %d: en pausa no recuerda ni elige: %+v", i, r)
		}
		if g := VistaGoal(root, now); g.Estado != GoalPausado || g.Motivo != MotivoEsperaPersona {
			t.Fatalf("vuelta %d: sigue en pausa por la persona: %+v", i, g)
		}
		if p, _ := PreguntaDe(root, diseno, now); p.Estado != PreguntaPendiente {
			t.Fatalf("vuelta %d: la pregunta de diseño es de la persona: %+v", i, p)
		}
		loDeLaPersonaIntacto("en pausa")
	}

	if _, err := CerrarPregunta(root, diseno, PreguntaRespondida, "", enCola.Add(5*time.Minute)); err != nil {
		t.Fatal(err)
	}
	recordatorios := 0
	var ultimo ResultadoGoal
	for i := 0; i < 10; i++ {
		ultimo = ciclo(enCola.Add(time.Hour+time.Duration(i)*3*time.Minute), silencio(SilencioGoal))
		if ultimo.Mensaje != "" {
			recordatorios++
		}
		loDeLaPersonaIntacto("trabajando")
	}
	g := VistaGoal(root, enCola)
	if recordatorios != MaxRecordatoriosGoal || g.Estado != GoalPausado || g.Motivo != MotivoSinAvance {
		t.Fatalf("sin avance se detiene tras %d recordatorios, diciendo por qué: %d %+v %q", MaxRecordatoriosGoal, recordatorios, g, ultimo.Aviso)
	}
	tipos := map[string]int{}
	for _, e := range g.Elegidas {
		tipos[e.Tipo]++
	}
	if len(g.Elegidas) != 2 || tipos["pregunta"] != 1 || tipos["pedido"] != 1 {
		t.Fatalf("cada elección queda registrada, y solo esas: %+v", g.Elegidas)
	}
	if _, err := os.Stat(marca); err != nil {
		t.Fatal("el pedido de la IA corrió")
	}
	if _, err := os.Stat(marcaPersona); err == nil {
		t.Fatal("el pedido de la persona no corrió")
	}
}
