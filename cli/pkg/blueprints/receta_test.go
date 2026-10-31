package blueprints

import (
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/modules"
)

// recetaConLimites es una receta completa: alcance, un parámetro de negocio, una constante técnica y pasos que
// llevan a cada escenario.
const recetaConLimites = `
id: panel
nombre: Panel
version: 1.0.0
requiere_qdd: 2.8.0
alcance:
  incluye: ["Ver las tareas y avisar cuando llega una"]
  excluye: ["Editar tareas desde el panel"]
vocabulario:
  - termino: tarea
    definicion: Una fila de la tabla.
parametros:
  - id: visibles
    pregunta: ¿Cuántas tareas se ven a la vez?
    tipo: numero
    por_defecto: "4"
    ejemplos: ["4", "10"]
constantes:
  - id: refresco
    valor: 2 s
    por_que: más seguido cuesta CPU; más lento, el panel se ve atrasado
preguntas:
  - id: avisos
    pregunta: ¿El panel avisa?
    por_defecto: "no"
    opciones:
      - valor: "no"
        descripcion: Sin avisos.
      - valor: si
        descripcion: Avisa cuando llega una tarea.
        invariantes: ["El aviso nunca quita el foco"]
        conformidad:
          - id: P-02
            titulo: El aviso
            dado: el panel cerrado
            cuando: llega una tarea
            entonces: el panel avisa sin quitar el foco
pasos:
  - id: leer
    accion: Leer las tareas cada {{refresco}} y mostrar las {{visibles}} más nuevas
    produce: la tabla con una tecla por tarea
    verifica: [P-01]
  - id: avisar
    si: {avisos: si}
    accion: Avisar cuando llega una tarea
    produce: el aviso, sin quitar el foco
    verifica: [P-02]
ficha:
  nombre: Panel
  objetivo_operativo:
    problema: Nadie ve las tareas.
    criterio_de_exito: Toda tarea se ve.
  objetivo_tecnologico:
    arquitectura: Un panel que lee un JSON.
    decisiones: ["Se ven las {{visibles}} tareas más nuevas"]
    invariantes: ["Una tarea nunca desaparece sin cierre"]
conformidad:
  - id: P-01
    titulo: Se ven las más nuevas
    dado: "{{visibles}} tareas o más"
    cuando: se dibuja el panel
    entonces: se ven las {{visibles}} más nuevas, cada una con su tecla
`

func TestReceta_CompletaEsValidaYApruebaCompleta(t *testing.T) {
	b := mustParse(t, recetaConLimites)
	if problems := b.Validate(); len(problems) > 0 {
		t.Fatal(problems)
	}
	if c := CalidadDe(b); c.Nota != 100 {
		t.Fatalf("una receta con límites, parámetros, constantes y pasos aprueba completa: %+v", c.Hallazgos)
	}
}

func TestReceta_ValidaQueCadaReferenciaExista(t *testing.T) {
	casos := map[string]string{
		"un {{x}} que no es pregunta, parámetro ni constante": strings.Replace(recetaConLimites, "cada {{refresco}}", "cada {{frecuencia}}", 1),
		"un paso que verifica un escenario que no existe":     strings.Replace(recetaConLimites, "verifica: [P-01]", "verifica: [P-09]", 1),
		"un parámetro sin valor por defecto":                  strings.Replace(recetaConLimites, "    por_defecto: \"4\"\n", "", 1),
		"un parámetro numérico con un valor que no es número": strings.Replace(recetaConLimites, "por_defecto: \"4\"", "por_defecto: cuatro", 1),
		"una constante sin su porqué":                         strings.Replace(recetaConLimites, "    por_que: más seguido cuesta CPU; más lento, el panel se ve atrasado\n", "", 1),
		"un parámetro con el id de una pregunta":              strings.Replace(recetaConLimites, "  - id: visibles\n", "  - id: avisos\n", 1),
		"un paso con una condición que no existe":             strings.Replace(recetaConLimites, "si: {avisos: si}", "si: {avisos: tal_vez}", 1),
	}
	for caso, yaml := range casos {
		if problems := mustParse(t, yaml).Validate(); len(problems) == 0 {
			t.Errorf("%s es inválido", caso)
		}
	}
}

func TestCalidad_LaRecetaDiceSusLimitesYComoConstruirla(t *testing.T) {
	got := codigos(CalidadDe(mustParse(t, recetaLimpia))) // una receta v1: sin alcance ni pasos
	for _, codigo := range []string{CalSinAlcance, CalSinPasos} {
		if _, ok := got[codigo]; !ok {
			t.Errorf("falta %s: %v", codigo, got)
		}
	}
	defectos := strings.NewReplacer(
		"verifica: [P-01]", "verifica: []",
		`invariantes: ["Una tarea nunca desaparece sin cierre"]`, `invariantes: ["Una tarea nunca desaparece sin cierre", "El panel se apaga a los 120 s sin uso", "La tabla se arma en tabla.go con armar()"]`,
		"    ejemplos: [\"4\", \"10\"]\n", "    ejemplos: [\"4\", \"10\"]\n  - id: idioma\n    pregunta: ¿En qué idioma?\n    por_defecto: es\n",
		"requiere_qdd: 2.8.0\n", "",
	).Replace(recetaConLimites)
	got = codigos(CalidadDe(mustParse(t, defectos)))
	want := map[string]string{
		CalPasoSinEscenario: "leer",
		CalEscenarioSinPaso: "P-01",
		CalValorFijo:        "invariantes",
		CalImplementacion:   "invariantes",
		CalSinUso:           "idioma",
		CalRequiere:         "requiere_qdd",
	}
	for codigo, donde := range want {
		if !strings.Contains(strings.Join(got[codigo], " "), donde) {
			t.Errorf("%s debe señalar %q; hallazgos: %v", codigo, donde, got)
		}
	}
}

func TestCalidad_UnValorFijoEnUnEscenarioEsUnEjemploNoUnaRegla(t *testing.T) {
	b := mustParse(t, strings.Replace(recetaConLimites, "cuando: se dibuja el panel", "cuando: se dibuja el panel a los 3 s", 1))
	if _, ok := codigos(CalidadDe(b))[CalValorFijo]; ok {
		t.Fatal("los escenarios traen datos de ejemplo; el valor fijo se busca en reglas y pasos")
	}
}

func TestApply_PreguntaLosParametrosDeNegocioALaPersona(t *testing.T) {
	root := t.TempDir()
	if _, problems := Add(AddInput{Root: root, Origen: sourceBlueprint(t, recetaConLimites)}); len(problems) > 0 {
		t.Fatal(problems)
	}
	res, problems := Apply(ApplyInput{Root: root, Blueprint: "panel", Rutas: []string{"internal/panel"}, Respuestas: map[string]string{"avisos": "si"}})
	if len(problems) > 0 || len(res.Pendientes) != 1 || res.Pendientes[0].ID != "visibles" || res.Pendientes[0].PorDefecto != "4" {
		t.Fatalf("el valor de negocio lo decide la persona, con su valor sugerido: %+v %v", res.Pendientes, problems)
	}
	if d := DescribePregunta(res.Pendientes[0]); !strings.Contains(d, "«4»") || !strings.Contains(d, "10") {
		t.Fatalf("se pregunta con su sugerido y sus ejemplos: %q", d)
	}
	if _, problems := Apply(ApplyInput{Root: root, Blueprint: "panel", Rutas: []string{"internal/panel"}, Respuestas: map[string]string{"avisos": "si", "visibles": "muchas"}}); len(problems) == 0 {
		t.Fatal("un parámetro numérico rechaza lo que no es número")
	}
	res, problems = Apply(ApplyInput{Root: root, Blueprint: "panel", Rutas: []string{"internal/panel"}, Respuestas: map[string]string{"avisos": "si", "visibles": "10"}})
	if len(problems) > 0 || len(res.Pendientes) > 0 {
		t.Fatal(problems, res.Pendientes)
	}
	m, _ := modules.Find(root, "panel")
	decisiones := strings.Join(m.ObjetivoTecnologico.Decisiones, "\n")
	for _, want := range []string{"Se ven las 10 tareas más nuevas", "Paso 1 · Leer las tareas cada 2 s y mostrar las 10 más nuevas → la tabla con una tecla por tarea (verifica P-01)", "Paso 2 · Avisar"} {
		if !strings.Contains(decisiones, want) {
			t.Fatalf("la ficha trae el valor elegido, la constante y los pasos en orden: falta %q en\n%s", want, decisiones)
		}
	}
	if strings.Contains(decisiones, "{{") {
		t.Fatalf("no queda ningún {{x}} sin reemplazar:\n%s", decisiones)
	}
}

func TestApply_UnPasoSoloConSuRespuesta(t *testing.T) {
	root := t.TempDir()
	if _, problems := Add(AddInput{Root: root, Origen: sourceBlueprint(t, recetaConLimites)}); len(problems) > 0 {
		t.Fatal(problems)
	}
	if _, problems := Apply(ApplyInput{Root: root, Blueprint: "panel", Rutas: []string{"internal/panel"}, Respuestas: map[string]string{"avisos": "no", "visibles": "4"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	m, _ := modules.Find(root, "panel")
	if d := strings.Join(m.ObjetivoTecnologico.Decisiones, "\n"); strings.Contains(d, "Avisar") || !strings.Contains(d, "Paso 1 · Leer") {
		t.Fatalf("sin avisos no hay paso de avisar:\n%s", d)
	}
}

func TestApply_UnBlueprintQuePideUnQDDMasNuevo(t *testing.T) {
	root := t.TempDir()
	if _, problems := Add(AddInput{Root: root, Origen: sourceBlueprint(t, strings.Replace(recetaConLimites, "requiere_qdd: 2.8.0", "requiere_qdd: 9.0.0", 1))}); len(problems) > 0 {
		t.Fatal(problems)
	}
	defer func(v string) { VersionQDD = v }(VersionQDD)
	VersionQDD = "v2.8.0-rc.9"
	if _, problems := Apply(ApplyInput{Root: root, Blueprint: "panel", Rutas: []string{"internal/panel"}, Respuestas: map[string]string{"avisos": "no", "visibles": "4"}}); len(problems) == 0 || !strings.Contains(problems[0], "9.0.0") {
		t.Fatalf("un qdd anterior no aplica a medias una receta que no entiende: %v", problems)
	}
	if !menorVersion("2.7.9", "2.8.0") || menorVersion("v2.8.0-rc.9", "2.8.0") {
		t.Fatal("una preview de 2.8.0 cumple con 2.8.0")
	}
}

func TestClasificar_ParametrosConstantesYPasos(t *testing.T) {
	base := mustParse(t, recetaConLimites)
	casos := map[string]struct {
		nuevo  string
		cambio string
	}{
		"otro valor sugerido":      {strings.Replace(recetaConLimites, `por_defecto: "4"`, `por_defecto: "6"`, 1), CambioPatch},
		"otra constante técnica":   {strings.Replace(recetaConLimites, "valor: 2 s", "valor: 5 s", 1), CambioMajor},
		"otro paso":                {strings.Replace(recetaConLimites, "produce: la tabla con una tecla por tarea", "produce: la tabla", 1), CambioMajor},
		"algo más que queda fuera": {strings.Replace(recetaConLimites, `excluye: ["Editar tareas desde el panel"]`, `excluye: ["Editar tareas desde el panel", "Borrar tareas"]`, 1), CambioMinor},
	}
	for caso, c := range casos {
		if got, _ := Clasificar(base, mustParse(t, c.nuevo)); got != c.cambio {
			t.Errorf("%s: %s, se esperaba %s", caso, got, c.cambio)
		}
	}
}

func TestIndice_ParametrosYPasos(t *testing.T) {
	i := IndiceDe(mustParse(t, recetaConLimites))
	if i.Pasos != 2 || len(i.Parametros) != 1 || i.Parametros[0].PorDefecto != "4" {
		t.Fatalf("el índice dice cuántos pasos y qué se adapta: %+v", i)
	}
	if l := LineaIndice(i); !strings.Contains(l, "▶2") || !strings.Contains(l, "⚙1") {
		t.Fatalf("la línea los cuenta: %q", l)
	}
	if s, err := Seccion(mustParse(t, recetaConLimites), "pasos"); err != nil || !strings.Contains(s, "1. Leer") || !strings.Contains(s, "si avisos = si") {
		t.Fatalf("sección pasos, en orden y con su condición: %q %v", s, err)
	}
	if s, _ := Seccion(mustParse(t, recetaConLimites), "alcance"); !strings.Contains(s, "✗ Editar tareas") {
		t.Fatalf("sección alcance: %q", s)
	}
}

func TestCalidad_CadaValorFijoSuPreguntaYLosEjemplosNoCuentan(t *testing.T) {
	b := mustParse(t, strings.Replace(recetaConLimites, `invariantes: ["Una tarea nunca desaparece sin cierre"]`,
		`invariantes: ["La vida es de 120 s; el panel late cada 5 s", "Un campo que falta se omite (p. ej. 0 %)", "El resumen dice «4 de 9 tareas · ⏳ 3 esperan»"]`, 1))
	var valores []string
	for _, h := range CalidadDe(b).Hallazgos {
		if h.Codigo == CalValorFijo {
			valores = append(valores, h.Detalle)
		}
	}
	if len(valores) != 2 || !strings.Contains(valores[0], "120 s") || !strings.Contains(valores[1], "5 s") {
		t.Fatalf("una pregunta por valor; un ejemplo entre «» o tras «p. ej.» no es una regla: %v", valores)
	}
}

func TestCalidad_UnValorDeclaradoSeCitaEnTodaLaReceta(t *testing.T) {
	b := mustParse(t, strings.NewReplacer(
		"definicion: Una fila de la tabla.", "definicion: Una fila de la tabla; se ven 4 a la vez, cada 2 s.",
		"entonces: se ven las {{visibles}} más nuevas, cada una con su tecla", "entonces: se ven las {{visibles}} más nuevas cada 2 s",
	).Replace(recetaConLimites))
	var donde []string
	for _, h := range CalidadDe(b).Hallazgos {
		if h.Codigo == CalValorFijo {
			donde = append(donde, h.Donde+": "+h.Arreglo)
		}
	}
	got := strings.Join(donde, "\n")
	if !strings.Contains(got, "vocabulario tarea") || !strings.Contains(got, "P-01: escribe {{refresco}}") {
		t.Fatalf("el vocabulario es regla, y un valor ya declarado se cita con {{id}} también en los escenarios:\n%s", got)
	}
}

func TestParametro_SoloSePreguntaCuandoAplica(t *testing.T) {
	conSi := strings.Replace(recetaConLimites, "  - id: visibles\n", "  - id: visibles\n    si: {avisos: si}\n", 1)
	b := mustParse(t, conSi)
	_, pendientes, _ := b.Resolver(map[string]string{"avisos": "no"})
	if len(pendientes) != 0 {
		t.Fatalf("sin avisos, no se pregunta cuántas tareas: %+v", pendientes)
	}
	if _, ok := codigos(CalidadDe(b))[CalCitaSinValor]; !ok {
		t.Fatal("con avisos=no, la decisión base quedaría con {{visibles}} sin reemplazar: es un defecto")
	}
}

func TestCalidad_CadaLimiteSeMideSolo(t *testing.T) {
	largo := strings.Repeat("Ver las tareas una por una ", 6) // ≈160 caracteres cada límite: juntos pasan de 280
	b := mustParse(t, strings.Replace(recetaConLimites, `incluye: ["Ver las tareas y avisar cuando llega una"]`, `incluye: ["`+largo+`", "`+largo+`avisar"]`, 1))
	if got := codigos(CalidadDe(b)); len(got[CalLargo]) > 0 {
		t.Fatalf("cada límite del alcance es una regla aparte: no se suman sus largos: %v", got)
	}
}
