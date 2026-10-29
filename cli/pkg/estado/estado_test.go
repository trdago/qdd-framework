package estado

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// RFC 0006: el archivo .qdd/project/estado.yaml y su lector automático (escenarios EPF-C01…C07 y EPF-A01 del blueprint
// estado-por-frente).

var ahora = time.Date(2026, 10, 29, 14, 0, 0, 0, time.UTC)

const valido = `formato: 1
etapas: [Código, Pruebas, Producción, Validación real]
actualizado: 2026-10-29T13:00:00Z
frentes:
  - id: gemini
    nombre: Gemini retirado
    estado: abierto
    celdas:
      Código: {estado: "✅", evidencia: "commit e29393e6"}
      Pruebas: {estado: "✅", evidencia: "gate 90/100"}
      Producción: {estado: "✅", evidencia: "run 40828808431"}
      Validación real: {estado: "⏳", bloqueo: "ver un análisis con imagen", de_quien: "uso real"}
  - id: ticket-logs
    nombre: Ticket MP en logs
    estado: abierto
    celdas:
      Código: {estado: "❌", falla: "la URL con el ticket se imprime"}
      Pruebas: {estado: "—"}
      Producción: {estado: "—"}
      Validación real: {estado: "—"}
  - id: viejo
    nombre: Frente cerrado hace tiempo
    estado: cerrado
    cerrado_el: 2026-10-01T00:00:00Z
    motivo_cierre: validado en uso real
    celdas:
      Código: {estado: "✅", evidencia: "v1.0.0"}
      Pruebas: {estado: "✅", evidencia: "12/12 pruebas"}
      Producción: {estado: "✅", evidencia: "v1.0.0"}
      Validación real: {estado: "✅", evidencia: "uso real del 2026-10-02"}
`

func mustParse(t *testing.T, s string) Estado {
	t.Helper()
	e, err := Parse([]byte(s))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return e
}

func ids(vs []Violacion) string {
	var out []string
	for _, v := range vs {
		out = append(out, v.ID)
	}
	return strings.Join(out, ",")
}

func tieneViolacion(vs []Violacion, id string) bool {
	for _, v := range vs {
		if v.ID == id {
			return true
		}
	}
	return false
}

func TestCheck_ArchivoValidoNoTieneViolaciones(t *testing.T) {
	if vs := Check(mustParse(t, valido), nil); len(vs) != 0 {
		t.Fatalf("archivo válido: %v", vs)
	}
}

func TestCheck_EPFC01_UnEstadoPorCelda(t *testing.T) {
	s := strings.Replace(valido, `Pruebas: {estado: "—"}`, `Pruebas: {estado: "listo"}`, 1)
	s = strings.Replace(s, `      Producción: {estado: "—"}`+"\n", "", 1)
	vs := Check(mustParse(t, s), nil)
	if !tieneViolacion(vs, "EPF-C01") {
		t.Fatalf("un estado fuera de ✅ ⏳ ❌ — y una celda faltante son EPF-C01: %s", ids(vs))
	}
	n := 0
	for _, v := range vs {
		if v.ID == "EPF-C01" {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("2 celdas inválidas (estado desconocido y celda faltante), obtuve %d: %v", n, vs)
	}
}

func TestCheck_EPFC02_HechoSinEvidencia(t *testing.T) {
	s := strings.Replace(valido, `{estado: "✅", evidencia: "gate 90/100"}`, `{estado: "✅"}`, 1)
	if vs := Check(mustParse(t, s), nil); !tieneViolacion(vs, "EPF-C02") {
		t.Fatalf("✅ sin evidencia: %s", ids(vs))
	}
}

func TestCheck_EPFC03_EsperaSinBloqueoODeQuien(t *testing.T) {
	s := strings.Replace(valido, `de_quien: "uso real"`, `de_quien: ""`, 1)
	if vs := Check(mustParse(t, s), nil); !tieneViolacion(vs, "EPF-C03") {
		t.Fatalf("⏳ sin de quién: %s", ids(vs))
	}
}

func TestCheck_FallaSinNombrar(t *testing.T) {
	s := strings.Replace(valido, `falla: "la URL con el ticket se imprime"`, `falla: ""`, 1)
	if vs := Check(mustParse(t, s), nil); !tieneViolacion(vs, "EPF-C01") {
		t.Fatalf("❌ sin nombrar la falla: %s", ids(vs))
	}
}

func TestCheck_EPFC04_DesplegadoNoEsValidado(t *testing.T) {
	s := strings.Replace(valido, `Validación real: {estado: "⏳", bloqueo: "ver un análisis con imagen", de_quien: "uso real"}`,
		`Validación real: {estado: "✅", evidencia: "run 40828808431"}`, 1)
	if vs := Check(mustParse(t, s), nil); !tieneViolacion(vs, "EPF-C04") {
		t.Fatalf("Validación real ✅ con la misma evidencia que Producción: %s", ids(vs))
	}
}

func TestCheck_EPFC04_ModelosProduccionSinPuerta(t *testing.T) {
	s := `formato: 1
etapas: [Datos, Entreno, Puerta, Producción]
frentes:
  - id: m1
    nombre: Modelo 1
    estado: abierto
    celdas:
      Datos: {estado: "✅", evidencia: "v1"}
      Entreno: {estado: "✅", evidencia: "run 7"}
      Puerta: {estado: "⏳", bloqueo: "medir F1", de_quien: "QA"}
      Producción: {estado: "✅", evidencia: "v1.2.0"}
`
	if vs := Check(mustParse(t, s), nil); !tieneViolacion(vs, "EPF-C04") {
		t.Fatalf("Producción ✅ sin la Puerta en ✅: %s", ids(vs))
	}
}

func TestCheck_EPFC05_FrenteQueDesapareceSinCierre(t *testing.T) {
	prev := mustParse(t, valido)
	cur := mustParse(t, valido)
	cur.Frentes = cur.Frentes[:1]
	vs := Check(cur, &prev)
	if !tieneViolacion(vs, "EPF-C05") {
		t.Fatalf("frente borrado en vez de cerrado: %s", ids(vs))
	}
	cerrado := mustParse(t, strings.Replace(valido, "    estado: cerrado\n    cerrado_el: 2026-10-01T00:00:00Z\n    motivo_cierre: validado en uso real\n",
		"    estado: cerrado\n    cerrado_el: 2026-10-01T00:00:00Z\n", 1))
	if vs := Check(cerrado, nil); !tieneViolacion(vs, "EPF-C05") {
		t.Fatalf("frente cerrado sin motivo: %s", ids(vs))
	}
}

func TestCheck_EPFC06_AprobacionPendienteNoPasaAHecho(t *testing.T) {
	s := strings.Replace(valido, `{estado: "✅", evidencia: "run 40828808431"}`,
		`{estado: "✅", evidencia: "run 40828808431", requiere_aprobacion: "usuario"}`, 1)
	if vs := Check(mustParse(t, s), nil); !tieneViolacion(vs, "EPF-C06") {
		t.Fatalf("✅ con aprobación pendiente: %s", ids(vs))
	}
	aprobado := strings.Replace(s, `requiere_aprobacion: "usuario"}`, `requiere_aprobacion: "usuario", aprobado: true}`, 1)
	if vs := Check(mustParse(t, aprobado), nil); len(vs) != 0 {
		t.Fatalf("con la aprobación dada no hay violación: %v", vs)
	}
}

func TestCheck_EPFC07_SinSecretosNiDatosPersonales(t *testing.T) {
	for _, malo := range []string{"sk-abcdefghijklmnopqrstuvwx", "ghp_abcdefghijklmnopqrstuv", "AKIAABCDEFGHIJKLMNOP", "password=hunter2", "juan@empresa.cl", "RUT 12.345.678-5"} {
		s := strings.Replace(valido, `bloqueo: "ver un análisis con imagen"`, `bloqueo: "`+malo+`"`, 1)
		if vs := Check(mustParse(t, s), nil); !tieneViolacion(vs, "EPF-C07") {
			t.Fatalf("%q debe ser EPF-C07: %s", malo, ids(vs))
		}
	}
}

func TestStatusline(t *testing.T) {
	got := Statusline(mustParse(t, valido), ahora)
	if got != "◧ 2 frentes · ⏳1 ❌1" {
		t.Fatalf("resumen: %q", got)
	}
	sinNada := mustParse(t, "formato: 1\netapas: [Código]\nfrentes: []\n")
	if got := Statusline(sinNada, ahora); got != "" {
		t.Fatalf("sin frentes abiertos no se muestra nada: %q", got)
	}
}

func TestStatuslineDir_SinArchivoODanadoNoMuestraNada(t *testing.T) {
	dir := t.TempDir()
	if got := StatuslineDir(dir, ahora); got != "" {
		t.Fatalf("sin archivo: %q", got)
	}
	os.MkdirAll(filepath.Join(dir, ".qdd", "project"), 0o755)
	os.WriteFile(filepath.Join(dir, Ruta), []byte("{{{ no es yaml"), 0o644)
	if got := StatuslineDir(dir, ahora); got != "" {
		t.Fatalf("archivo dañado: %q", got)
	}
}

func TestMarkdown_AlcanceAbiertosYRecienCerrados(t *testing.T) {
	e := mustParse(t, valido)
	md := Markdown(e, ahora, AlcanceAbiertos)
	if !strings.Contains(md, "| Frente | Código | Pruebas | Producción | Validación real |") {
		t.Fatalf("encabezado: %s", md)
	}
	if !strings.Contains(md, "✅ commit e29393e6") || !strings.Contains(md, "⏳ ver un análisis con imagen (uso real)") {
		t.Fatalf("celdas con evidencia y bloqueo: %s", md)
	}
	if strings.Contains(md, "Frente cerrado hace tiempo") {
		t.Fatalf("un cerrado hace más de 7 días no entra en el alcance «abiertos»: %s", md)
	}
	if !strings.Contains(Markdown(e, ahora, AlcanceTodos), "Frente cerrado hace tiempo") {
		t.Fatal("alcance «todos» muestra los cerrados")
	}
}

func TestCheckRespuesta_EPFA01(t *testing.T) {
	e := mustParse(t, valido)
	md := Markdown(e, ahora, AlcanceAbiertos)
	if vs := CheckRespuesta(e, md, ahora); len(vs) != 0 {
		t.Fatalf("la tabla generada coincide con el archivo: %v", vs)
	}
	distinta := strings.Replace(md, "⏳ ver un análisis con imagen (uso real)", "✅ listo", 1)
	if vs := CheckRespuesta(e, distinta, ahora); !tieneViolacion(vs, "EPF-A01") {
		t.Fatalf("la respuesta dice ✅ y el archivo ⏳: %s", ids(vs))
	}
	if vs := CheckRespuesta(e, "Sin tabla en esta respuesta.", ahora); !tieneViolacion(vs, "EPF-A01") {
		t.Fatalf("una respuesta sin tabla no coincide: %s", ids(vs))
	}
}

func TestSet_ValidaAntesDeEscribirYEsAtomico(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".qdd", "project"), 0o755)
	os.WriteFile(filepath.Join(dir, Ruta), []byte(valido), 0o644)

	err := Set(dir, Cambio{Frente: "gemini", Etapa: "Validación real", Estado: "ok", Evidencia: "análisis 1234-5-LP26 con imagen leída"}, ahora)
	if err != nil {
		t.Fatalf("set válido: %v", err)
	}
	e, _ := Load(dir)
	c := e.Frentes[0].Celdas["Validación real"]
	if c.Estado != Hecho || c.Bloqueo != "" || c.DeQuien != "" {
		t.Fatalf("el alias «ok» es ✅ y limpia el bloqueo anterior: %+v", c)
	}

	antes, _ := os.ReadFile(filepath.Join(dir, Ruta))
	err = Set(dir, Cambio{Frente: "gemini", Etapa: "Pruebas", Estado: "✅"}, ahora)
	if err == nil {
		t.Fatal("✅ sin evidencia se rechaza")
	}
	despues, _ := os.ReadFile(filepath.Join(dir, Ruta))
	if string(antes) != string(despues) {
		t.Fatal("un set inválido no toca el archivo")
	}
	if err := Set(dir, Cambio{Frente: "gemini", Etapa: "Inventada", Estado: "—"}, ahora); err == nil {
		t.Fatal("una etapa que no está en el archivo se rechaza")
	}
}

func TestSet_FrenteNuevoYCerrar(t *testing.T) {
	dir := t.TempDir()
	err := Set(dir, Cambio{Frente: "nuevo", Nombre: "Frente nuevo", Etapa: "Código", Estado: "espera", Bloqueo: "revisión", DeQuien: "usuario"}, ahora)
	if err == nil {
		t.Fatal("sin archivo no se sabe qué etapas usar: se pide Init")
	}
	if err := Init(dir, []string{"Código", "Pruebas"}, ahora); err != nil {
		t.Fatal(err)
	}
	if err := Set(dir, Cambio{Frente: "nuevo", Nombre: "Frente nuevo", Etapa: "Código", Estado: "espera", Bloqueo: "revisión", DeQuien: "usuario"}, ahora); err != nil {
		t.Fatalf("frente nuevo: %v", err)
	}
	e, _ := Load(dir)
	if len(e.Frentes) != 1 || e.Frentes[0].Celdas["Pruebas"].Estado != NoAplica {
		t.Fatalf("un frente nuevo completa sus otras celdas con —: %+v", e.Frentes)
	}
	if err := Cerrar(dir, "nuevo", "", false, ahora); err == nil {
		t.Fatal("cerrar sin motivo se rechaza")
	}
	if err := Cerrar(dir, "nuevo", "duplicado de otro frente", true, ahora); err != nil {
		t.Fatal(err)
	}
	e, _ = Load(dir)
	if e.Frentes[0].Estado != EstadoDescartado || e.Frentes[0].MotivoCierre == "" {
		t.Fatalf("descartado con motivo: %+v", e.Frentes[0])
	}
}

func TestPanel_AjustaAlAnchoYListaEsperasYFallas(t *testing.T) {
	e := mustParse(t, valido)
	lineas := Panel(e, "notificador", 40, ahora)
	for _, l := range lineas {
		if AnchoVisible(l) > 40 {
			t.Fatalf("línea más ancha que la terminal (%d): %q", AnchoVisible(l), l)
		}
	}
	todo := strings.Join(lineas, "\n")
	if !strings.Contains(todo, "Esperando") || !strings.Contains(todo, "uso real") {
		t.Fatalf("lista lo que espera y de quién: %s", todo)
	}
	if !strings.Contains(todo, "Fallas") || !strings.Contains(todo, "ticket") {
		t.Fatalf("lista las fallas: %s", todo)
	}
}

func TestOpcionesYEtapasDeLaFicha(t *testing.T) {
	o := Opciones([]string{"Los estados son cerrados: ✅ ⏳ ❌ —", "etapas → software", "destino → archivo"})
	if o["etapas"] != "software" || o["destino"] != "archivo" || len(o) != 2 {
		t.Fatalf("opciones: %v", o)
	}
	if got := strings.Join(EtapasDe("software"), ","); got != "Código,Pruebas,Producción,Validación real" {
		t.Fatalf("etapas software: %s", got)
	}
	if EtapasDe("propias") != nil {
		t.Fatal("«propias» se leen del archivo, no se inventan")
	}
}

func TestAbreviaturas_SonUnicasAunqueEmpiecenIgual(t *testing.T) {
	got := abreviaturas([]string{"Código", "Pruebas", "Producción", "Validación real"})
	vistas := map[string]bool{}
	for _, a := range got {
		if vistas[a] {
			t.Fatalf("abreviatura repetida %q en %v", a, got)
		}
		vistas[a] = true
		if AnchoVisible(a) != 2 {
			t.Fatalf("abreviatura de 2 columnas: %q", a)
		}
	}
	if got[0] != "Có" || got[3] != "Va" {
		t.Fatalf("sin choque se usan las 2 primeras letras: %v", got)
	}
}

func TestPanel_ColumnaFrenteMideLoQueElNombreMasLargo(t *testing.T) {
	e := mustParse(t, valido)
	lineas := Panel(e, "notificador", 200, ahora)
	largo := 0
	for _, f := range e.Frentes {
		if n := AnchoVisible(f.Nombre); n > largo {
			largo = n
		}
	}
	cab := lineas[2]
	if AnchoVisible(cab) > largo+1+3*len(e.Etapas) {
		t.Fatalf("la columna Frente no debe rellenar hasta el ancho de la terminal: %d columnas en %q", AnchoVisible(cab), cab)
	}
}

func TestDatos_LoQueDibujaElPanelDeLaConsola(t *testing.T) {
	e := mustParse(t, valido)
	d := Datos(e, ahora)
	if d.Resumen != Statusline(e, ahora) || len(d.Frentes) != 2 {
		t.Fatalf("resumen y solo frentes abiertos (o recién cerrados): %+v", d)
	}
	if strings.Join(d.Abreviaturas, ",") != "Có,Pu,Po,Va" {
		t.Fatalf("abreviaturas únicas: %v", d.Abreviaturas)
	}
	if strings.Join(d.Frentes[0].Simbolos, "") != "✅✅✅⏳" {
		t.Fatalf("un símbolo por etapa, en orden: %v", d.Frentes[0].Simbolos)
	}
	if len(d.Esperas) != 1 || d.Esperas[0].DeQuien != "uso real" || d.Esperas[0].Texto == "" {
		t.Fatalf("cada ⏳ con qué espera y de quién: %+v", d.Esperas)
	}
	if len(d.Fallas) != 1 || !strings.Contains(d.Fallas[0].Texto, "ticket") {
		t.Fatalf("cada ❌ con su falla: %+v", d.Fallas)
	}
	if _, err := json.Marshal(d); err != nil {
		t.Fatal(err)
	}
}
