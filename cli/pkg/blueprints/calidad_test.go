package blueprints

import (
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// recetaConDefectos tiene un defecto de cada tipo; recetaLimpia es la misma receta corregida, sin perder nada.
var recetaConDefectos = `
id: panel
nombre: Panel
version: 1.0.0
vocabulario:
  - termino: tarea
    definicion: Una fila de la tabla.
  - termino: farol
    definicion: Una palabra que nadie usa.
preguntas:
  - id: avisos
    pregunta: ¿El panel avisa?
    por_defecto: "no"
    opciones:
      - valor: "no"
        descripcion: Sin avisos.
      - valor: si
        descripcion: Avisa cuando llega una tarea.
        depende_de: ["qdd ≥ 2.3.0"]
        decisiones: ["El aviso usa la cola de pedidos (qdd ≥ 2.6.0)", "Cada tarea tiene su tecla"]
        invariantes: ["El aviso nunca quita el foco"]
  - id: color
    pregunta: ¿Con color?
    opciones:
      - valor: si
      - valor: "no"
        descripcion: Monocromo.
ficha:
  nombre: Panel
  objetivo_operativo:
    problema: Nadie ve las tareas.
    criterio_de_exito: Toda tarea se ve.
  objetivo_tecnologico:
    arquitectura: Un panel que lee un JSON.
    decisiones: ["Cada tarea tiene su tecla"]
    invariantes:
      - "Una tarea nunca desaparece sin cierre"
      - "` + strings.Repeat("Una regla que dice demasiado en una sola línea, ", 7) + `"
conformidad:
  - id: P-01
    titulo: Se ve la tarea
    cuando: llega una tarea
    entonces: el panel la muestra con su tecla
  - id: P-02
    titulo: El aviso
    dado: avisos=si y el panel cerrado
    cuando: llega una tarea
    entonces: "el panel avisa sin quitar el foco: ` + "```js\\nfocus(false)\\n```" + `"
`

const recetaLimpia = `
id: panel
nombre: Panel
version: 1.0.0
vocabulario:
  - termino: tarea
    definicion: Una fila de la tabla.
preguntas:
  - id: avisos
    pregunta: ¿El panel avisa?
    por_defecto: "no"
    opciones:
      - valor: "no"
        descripcion: Sin avisos.
      - valor: si
        descripcion: Avisa cuando llega una tarea.
        depende_de: ["qdd ≥ 2.6.0"]
        decisiones: ["El aviso usa la cola de pedidos (qdd ≥ 2.6.0)"]
        invariantes: ["El aviso nunca quita el foco"]
        conformidad:
          - id: P-02
            titulo: El aviso
            dado: el panel cerrado
            cuando: llega una tarea
            entonces: el panel avisa sin quitar el foco
  - id: color
    pregunta: ¿Con color?
    opciones:
      - valor: si
        descripcion: Con color.
      - valor: "no"
        descripcion: Monocromo.
ficha:
  nombre: Panel
  objetivo_operativo:
    problema: Nadie ve las tareas.
    criterio_de_exito: Toda tarea se ve.
  objetivo_tecnologico:
    arquitectura: Un panel que lee un JSON.
    decisiones: ["Cada tarea tiene su tecla"]
    invariantes:
      - "Una tarea nunca desaparece sin cierre"
      - "Una regla corta"
conformidad:
  - id: P-01
    titulo: Se ve la tarea
    cuando: llega una tarea
    entonces: el panel la muestra con su tecla
`

func codigos(c Calidad) map[string][]string {
	out := map[string][]string{}
	for _, h := range c.Hallazgos {
		out[h.Codigo] = append(out[h.Codigo], h.Donde)
	}
	return out
}

func TestCalidad_EncuentraCadaDefectoDeLaReceta(t *testing.T) {
	c := CalidadDe(mustParse(t, recetaConDefectos))
	got := codigos(c)
	want := map[string]string{
		CalEscenarioDeOpcion: "P-02",        // solo vale con avisos=si, pero se aplica siempre
		CalVersionMinima:     "avisos=si",   // pide qdd ≥ 2.3.0 y usa algo de 2.6.0
		CalOpcionVacia:       "color=si",    // no dice qué implica elegirla
		CalCodigo:            "P-02",        // una receta no trae código
		CalLargo:             "invariantes", // una regla por línea
		CalRepetido:          "avisos=si",   // repite una decisión de la base
		CalTerminoSinUso:     "farol",       // vocabulario que nadie usa
	}
	for codigo, donde := range want {
		if !strings.Contains(strings.Join(got[codigo], " "), donde) {
			t.Errorf("%s debe señalar %q; hallazgos: %v", codigo, donde, got)
		}
	}
	if c.Nota >= 70 || c.Defectos < 4 {
		t.Fatalf("una receta con defectos no aprueba: nota %d, %d defectos", c.Nota, c.Defectos)
	}
}

func TestCalidad_LaRecetaCorregidaNoPierdeNadaYApruebaCompleta(t *testing.T) {
	antes, despues := mustParse(t, recetaConDefectos), mustParse(t, recetaLimpia)
	c := CalidadDe(despues)
	if c.Nota != 100 || len(c.Hallazgos) != 0 {
		t.Fatalf("la receta corregida no tiene hallazgos: %d %+v", c.Nota, c.Hallazgos)
	}
	ia, id := IndiceDe(antes), IndiceDe(despues)
	if !mismos(ia.Escenarios, id.Escenarios) {
		t.Fatalf("corregir no quita escenarios: %v → %v", ia.Escenarios, id.Escenarios)
	}
	if CalidadDe(antes).Tokens.PorDefecto <= c.Tokens.PorDefecto {
		t.Fatalf("el escenario que va en su opción ya no se lee con la respuesta por defecto: %d → %d", CalidadDe(antes).Tokens.PorDefecto, c.Tokens.PorDefecto)
	}
}

func mismos(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for _, x := range a {
		if !inList(b, x) {
			return false
		}
	}
	return true
}

func TestCalidad_MideLosTokensDeLaReceta(t *testing.T) {
	c := CalidadDe(mustParse(t, recetaLimpia))
	if c.Tokens.Indice <= 0 || c.Tokens.PorDefecto <= 0 || c.Tokens.Maxima < c.Tokens.PorDefecto {
		t.Fatalf("índice, receta por defecto y receta más grande: %+v", c.Tokens)
	}
	if c.Tokens.Indice >= c.Tokens.Maxima {
		t.Fatalf("el índice cuesta menos que la receta: %+v", c.Tokens)
	}
}

func TestCalidad_UnaLineaParaListar(t *testing.T) {
	l := LineaCalidad(CalidadDe(mustParse(t, recetaConDefectos)))
	if strings.Contains(l, "\n") || utf8.RuneCountInString(l) > MaxLineaIndice {
		t.Fatalf("una línea corta: %q", l)
	}
	for _, want := range []string{"panel@1.0.0", "/100", "⚠", "✂", "tokens"} {
		if !strings.Contains(l, want) {
			t.Fatalf("falta %q en %q", want, l)
		}
	}
}

func TestCalidad_CadaHallazgoDiceComoArreglarloSinPerderNada(t *testing.T) {
	for _, h := range CalidadDe(mustParse(t, recetaConDefectos)).Hallazgos {
		if h.Codigo == "" || h.Donde == "" || h.Arreglo == "" {
			t.Fatalf("código, dónde y cómo arreglarlo: %+v", h)
		}
		if utf8.RuneCountInString(h.Texto()) > 200 {
			t.Fatalf("estilo chat, una línea: %q", h.Texto())
		}
	}
}

// FND-093: el índice contaba dos veces una invariante que comparten dos opciones (anclas YAML): 🔒 mentía.
func TestIndice_NoCuentaDosVecesLaMismaInvariante(t *testing.T) {
	b := mustParse(t, `
id: x
nombre: X
version: 1.0.0
preguntas:
  - id: modo
    pregunta: ¿Modo?
    opciones:
      - valor: a
        invariantes: &comunes ["Regla uno", "Regla dos"]
      - valor: b
        invariantes: *comunes
ficha:
  nombre: X
  objetivo_operativo: {problema: p, criterio_de_exito: c}
  objetivo_tecnologico:
    arquitectura: a
    invariantes: ["Regla base", "Regla uno"]
`)
	if n := IndiceDe(b).Invariantes; n != 3 {
		t.Fatalf("3 invariantes distintas (base, uno, dos), no %d", n)
	}
}

func TestMedir_UnaCarpetaDeAporteOUnImportado(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ManifestName), recetaLimpia)
	c, _, err := Medir(t.TempDir(), []string{dir})
	if err != nil || len(c) != 1 || c[0].Ref != "panel@1.0.0" || c[0].Nota != 100 {
		t.Fatalf("mide la carpeta de un aporte en curso: %+v %v", c, err)
	}
	if _, _, err := Medir(t.TempDir(), nil); err == nil || !strings.Contains(err.Error(), "qdd blueprint add") {
		t.Fatalf("sin blueprints ni fuente, dice cómo importarlos: %v", err)
	}
}

func TestMedir_SinImportadosMideLoPublicadoEnLaFuente(t *testing.T) {
	repo := gitOrgRepo(t)
	root := t.TempDir()
	writeFile(t, filepath.Join(root, configFile), "blueprints:\n  origen: "+repo+"\n")
	c, fuente, err := Medir(root, nil)
	if err != nil || len(c) == 0 || !strings.Contains(fuente, "v1.0.0") {
		t.Fatalf("mide cada blueprint publicado, con su versión: %+v %q %v", c, fuente, err)
	}
}

func TestDefectosNuevos_SoloLosQueAgregaElCambio(t *testing.T) {
	antes := CalidadDe(mustParse(t, authBlueprint))
	if len(DefectosNuevos(antes, antes)) != 0 {
		t.Fatal("los defectos que ya tenía no bloquean un cambio")
	}
	peor := strings.Replace(authBlueprint, "conformidad:\n  - id: AUTH-01", "conformidad:\n  - id: AUTH-02\n    titulo: Por WhatsApp\n    dado: recuperacion=whatsapp\n    cuando: se pide recuperar\n    entonces: llega un código\n  - id: AUTH-01", 1)
	nuevos := DefectosNuevos(antes, CalidadDe(mustParse(t, peor)))
	if len(nuevos) != 1 || nuevos[0].Codigo != CalEscenarioDeOpcion || nuevos[0].Donde != "AUTH-02" {
		t.Fatalf("solo el defecto que agrega el cambio: %+v", nuevos)
	}
}

func TestProponer_NoPublicaUnaRecetaPeor(t *testing.T) {
	pedidos := sinGH(t)
	repo := gitOrgRepo(t)
	root := t.TempDir()
	if _, problems := AddManaged(AddInput{Root: root, Origen: repo, Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	ap, err := Aportar(root, "auth", "")
	if err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(ap.Carpeta, ManifestName)
	writeFile(t, manifest, strings.Replace(authBlueprint, "    entonces: la respuesta no revela si la cuenta existe\n", "    entonces: la respuesta no revela si la cuenta existe\n  - id: AUTH-02\n    titulo: Por WhatsApp\n    dado: recuperacion=whatsapp\n    cuando: se pide recuperar\n    entonces: llega un código\n", 1))
	if _, err := Proponer(root, "auth", "recupera por whatsapp", ""); err == nil || !strings.Contains(err.Error(), "CAL-01 AUTH-02") {
		t.Fatalf("un aporte que empeora la receta no se propone, y dice qué arreglar: %v", err)
	}
	writeFile(t, manifest, strings.Replace(authBlueprint, "      - valor: correo\n      - valor: whatsapp", "      - valor: correo\n        descripcion: Un enlace al correo.\n      - valor: whatsapp", 1))
	if _, err := Proponer(root, "auth", "describe la recuperación por correo", ""); err != nil {
		t.Fatal(err)
	}
	if len(*pedidos) != 1 || !strings.Contains((*pedidos)[0], "Calidad") || !strings.Contains((*pedidos)[0], "calidad 60/100") || !strings.Contains((*pedidos)[0], "` → `") {
		t.Fatalf("el PR dice cómo cambia la calidad: %v", *pedidos)
	}
}

func TestProponer_UnBlueprintNuevoNaceSinDefectos(t *testing.T) {
	sinGH(t)
	repo := gitOrgRepo(t)
	root := t.TempDir()
	if _, problems := AddManaged(AddInput{Root: root, Origen: repo, Blueprints: []string{"auth"}}); len(problems) > 0 {
		t.Fatal(problems)
	}
	ap, err := Aportar(root, "otro", "")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(ap.Carpeta, ManifestName), strings.Replace(recetaConDefectos, "id: panel", "id: otro", 1))
	if _, err := Proponer(root, "otro", "nuevo", ""); err == nil || !strings.Contains(err.Error(), CalOpcionVacia) {
		t.Fatalf("una receta nueva incompleta no se propone: %v", err)
	}
}

func TestCalidad_UnTerminoEnPluralSeUsa(t *testing.T) {
	if !seUsa("acción del panel", "acciones del panel: aprobar cada ficha") || seUsa("farol", "una tabla con tareas") {
		t.Fatal("el término se reconoce por la raíz de sus palabras")
	}
}

func TestCalidad_UnaOpcionQueDependeDeOtraRespuestaVaConSi(t *testing.T) {
	b := mustParse(t, strings.Replace(recetaLimpia, "  - id: color\n    pregunta: ¿Con color?\n    opciones:\n      - valor: si\n        descripcion: Con color.",
		"  - id: color\n    pregunta: ¿Con color?\n    opciones:\n      - valor: si\n        descripcion: Con color.\n        depende_de: [\"avisos=si\"]", 1))
	got := codigos(CalidadDe(b))[CalCondicion]
	if len(got) != 1 || got[0] != "color=si" {
		t.Fatalf("con avisos=no se podría elegir color=si: la pregunta va con si: {avisos: si}: %v", got)
	}
	condicional := strings.Replace(recetaLimpia, "  - id: color\n    pregunta: ¿Con color?\n", "  - id: color\n    pregunta: ¿Con color?\n    si: {avisos: si}\n", 1)
	if n := len(codigos(CalidadDe(mustParse(t, condicional)))[CalCondicion]); n != 0 {
		t.Fatalf("una pregunta con si ya no se hace sin esa respuesta: %d", n)
	}
}
