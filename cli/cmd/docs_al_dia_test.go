package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Rule: Prohibido usar 'else'

// FND-307: la documentación pública quedó en la 2.9 mientras QDD llegaba a la 2.10: el README no nombraba la mitad de los
// comandos, no había índice de RFC y el Go que pedía no compilaba el proyecto. La referencia ya se comparaba con el
// binario; esto mira lo demás.

const enlacePublico = "https://github.com/trdago/qdd-framework/blob/main/"

// documentosPublicos: lo que lee quien llega por GitHub o npm.
func documentosPublicos(t *testing.T) []string {
	t.Helper()
	docs := []string{"../../README.md", "../../CONTRIBUTING.md", "../../SECURITY.md", "../../npm/README.md", "../../specification/qdd-spec.md"}
	for _, patron := range []string{"../../docs/*.md", "../../rfcs/*.md"} {
		rutas, err := filepath.Glob(patron)
		if err != nil {
			t.Fatal(err)
		}
		docs = append(docs, rutas...)
	}
	return docs
}

func TestDocs_ReadmeNombraCadaComandoDePrimerNivel_FND307(t *testing.T) {
	readme := leerDoc(t, "../../README.md")
	for _, c := range rootCmd.Commands() {
		if c.Hidden || c.Name() == "help" || c.Name() == "completion" {
			continue
		}
		if !strings.Contains(readme, "`qdd "+c.Name()) {
			t.Errorf("README.md no nombra `qdd %s` en su mapa de comandos", c.Name())
		}
	}
}

var estadoDeRFC = regexp.MustCompile(`(?mi)^[-* ]*(?:\*\*)?(?:Status|Estado):(?:\*\*)?\s*(.+)$`)

func TestDocs_ElIndiceDeRFCTieneTodosYCadaUnoDiceSuEstado_FND307(t *testing.T) {
	indice := leerDoc(t, "../../rfcs/README.md")
	rutas, err := filepath.Glob("../../rfcs/[0-9]*.md")
	if err != nil || len(rutas) == 0 {
		t.Fatalf("no hay RFC que revisar: %v", err)
	}
	for _, ruta := range rutas {
		nombre := filepath.Base(ruta)
		if !strings.Contains(indice, "("+nombre+")") {
			t.Errorf("rfcs/README.md no enlaza a %s", nombre)
		}
		if !estadoDeRFC.MatchString(leerDoc(t, ruta)) {
			t.Errorf("rfcs/%s no dice su estado", nombre)
		}
	}
}

func TestDocs_EnlacesYAnclasExisten_FND307(t *testing.T) {
	for _, archivo := range documentosPublicos(t) {
		for _, m := range enlaceMarkdown.FindAllStringSubmatch(sinCodigo(leerDoc(t, archivo)), -1) {
			if problema := problemaDeEnlace(archivo, m[1]); problema != "" {
				t.Errorf("%s: %s", strings.TrimPrefix(archivo, "../../"), problema)
			}
		}
	}
}

// problemaDeEnlace: "" si el destino (y su ancla) existe. Un enlace a blob/main se lee como el archivo del repositorio.
func problemaDeEnlace(archivo, enlace string) string {
	partes := strings.SplitN(enlace, "#", 2)
	destino := filepath.Join(filepath.Dir(archivo), partes[0])
	if strings.HasPrefix(enlace, enlacePublico) {
		destino = filepath.Join("../..", strings.TrimPrefix(partes[0], enlacePublico))
	}
	if partes[0] == "" {
		destino = archivo
	}
	if strings.Contains(enlace, "://") && !strings.HasPrefix(enlace, enlacePublico) {
		return ""
	}
	if strings.HasPrefix(enlace, "mailto:") {
		return ""
	}
	data, err := os.ReadFile(destino)
	if len(partes) == 2 && err == nil && !anclas(string(data))[partes[1]] {
		return "el ancla " + enlace + " no existe"
	}
	if len(partes) == 2 {
		return ""
	}
	if _, err := os.Stat(destino); err != nil {
		return "enlaza a " + enlace + ", que no existe"
	}
	return ""
}

var (
	bloqueDeCodigo = regexp.MustCompile("(?s)```.*?```")
	codigoEnLinea  = regexp.MustCompile("`[^`\n]*`")
	titulo         = regexp.MustCompile(`(?m)^#{1,6} +(.+)$`)
	enlaceEnTitulo = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)
)

func sinCodigo(doc string) string {
	return codigoEnLinea.ReplaceAllString(bloqueDeCodigo.ReplaceAllString(doc, ""), "")
}

// anclas: los ids que GitHub les da a los títulos (minúsculas, sin puntuación ni emojis, espacios como guiones).
func anclas(doc string) map[string]bool {
	ids := map[string]bool{}
	for _, m := range titulo.FindAllStringSubmatch(bloqueDeCodigo.ReplaceAllString(doc, ""), -1) {
		ids[anclaDe(enlaceEnTitulo.ReplaceAllString(m[1], "$1"))] = true
	}
	return ids
}

func anclaDe(texto string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(texto)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			b.WriteRune(r)
		}
		if r == ' ' {
			b.WriteRune('-')
		}
	}
	return b.String()
}

func TestDocs_ElAnclaEsLaDeGitHub(t *testing.T) {
	for texto, quiere := range map[string]string{
		"Instalación":                      "instalación",
		"🤖 Solo vía MCP — pedile a tu IDE": "-solo-vía-mcp--pedile-a-tu-ide",
		"`qdd audit` y más (RFC 0006)":     "qdd-audit-y-más-rfc-0006",
	} {
		if dio := anclaDe(texto); dio != quiere {
			t.Errorf("anclaDe(%q) = %q, quiere %q", texto, dio, quiere)
		}
	}
}

var goPedido = regexp.MustCompile(`Go \(?(\d+\.\d+)\+`)

func TestDocs_ElGoQuePidenEsElDeGoMod_FND307(t *testing.T) {
	gomod := regexp.MustCompile(`(?m)^go (\d+\.\d+)`).FindStringSubmatch(leerDoc(t, "../go.mod"))
	if gomod == nil {
		t.Fatal("go.mod no dice su versión de Go")
	}
	for _, archivo := range documentosPublicos(t) {
		for _, m := range goPedido.FindAllStringSubmatch(leerDoc(t, archivo), -1) {
			if m[1] != gomod[1] {
				t.Errorf("%s pide Go %s+, pero go.mod pide %s", strings.TrimPrefix(archivo, "../../"), m[1], gomod[1])
			}
		}
	}
}

// Afirmaciones que la documentación hacía y el código no sostiene. Una que vuelva a ser cierta se saca de aquí con la
// prueba de que lo es.
var afirmacionesSinRespaldo = []string{
	"9 pilares", "nivel faang", "horas ahorradas", "cumplimiento auditable", "missing-project-cert", "certificado firmado",
	"bloqueará el código libre", "read-only 100%", "qdd_pr_certificate", "dashboard_demo", "goreportcard",
}

func TestDocs_NoAfirmanLoQueElCodigoNoHace_FND307(t *testing.T) {
	for _, ruta := range documentosPublicos(t) {
		if strings.Contains(ruta, "/rfcs/0") {
			continue
		}
		for _, frase := range afirmacionesEn(leerDoc(t, ruta)) {
			t.Errorf("%s: afirma «%s», que el código no sostiene", ruta, frase)
		}
	}
}

func TestDocs_LaAfirmacionSeReconoceSinImportarMayusculas(t *testing.T) {
	if got := afirmacionesEn("Equipo resolutor Nivel FAANG y 9 pilares."); len(got) != 2 {
		t.Fatalf("esperaba dos afirmaciones, hay %v", got)
	}
	if got := afirmacionesEn("Corre las pruebas y las reglas."); len(got) != 0 {
		t.Fatalf("no hay ninguna, y encontró %v", got)
	}
}

func afirmacionesEn(texto string) []string {
	minusculas := strings.ToLower(texto)
	var halladas []string
	for _, frase := range afirmacionesSinRespaldo {
		if strings.Contains(minusculas, frase) {
			halladas = append(halladas, frase)
		}
	}
	return halladas
}

// Cada opción de cada comando está en una fila de la referencia que nombra ese comando.
func TestDocs_LaReferenciaNombraCadaOpcion_FND307(t *testing.T) {
	ref := strings.Split(leerDoc(t, "../../docs/command-reference.md"), "\n")
	revisadas := 0
	recorrerComandos(rootCmd, func(c *cobra.Command) {
		filas := filasDe(ref, c.CommandPath())
		c.LocalNonPersistentFlags().VisitAll(func(f *pflag.Flag) {
			if f.Hidden || f.Name == "help" {
				return
			}
			revisadas++
			if !strings.Contains(filas, "--"+f.Name) {
				t.Errorf("la referencia no nombra %s --%s: %s", c.CommandPath(), f.Name, f.Usage)
			}
		})
	})
	if revisadas < 50 {
		t.Fatalf("se revisaron %d opciones: el recorrido no está mirando los comandos", revisadas)
	}
}

func recorrerComandos(c *cobra.Command, mira func(*cobra.Command)) {
	if c.Hidden || c.Name() == "help" || c.Name() == "completion" {
		return
	}
	mira(c)
	for _, hijo := range c.Commands() {
		recorrerComandos(hijo, mira)
	}
}

func filasDe(ref []string, comando string) string {
	var filas strings.Builder
	for _, l := range ref {
		if strings.Contains(l, "`"+comando) {
			filas.WriteString(l)
		}
	}
	return filas.String()
}

// FND-321: el RFC 0013 y la referencia decían que el relay no guarda las acciones ni los textos, y lo que manda el
// teléfono por el relay (realtime.send) queda unos días en realtime.messages. La comunicación lo dice y nadie lo niega.
var niegaQueElRelayGuarda = regexp.MustCompile(`(?i)relay no (guarda|almacena|conserva)|no se (guardan|almacenan) en supabase`)

func TestDocs_NadieDiceQueElRelayNoGuardaLoDelTelefono_FND321(t *testing.T) {
	for _, archivo := range documentosPublicos(t) {
		for _, m := range niegaQueElRelayGuarda.FindAllString(leerDoc(t, archivo), -1) {
			t.Errorf("%s dice «%s»: lo que pasa por realtime.send queda de 72 horas a 4 días (FND-321)", archivo, m)
		}
	}
	comunicacion := leerDoc(t, "../../docs/qdd-movil-comunicacion.md")
	_, seccion, hay := strings.Cut(comunicacion, "## Qué queda guardado en el relay")
	if !hay {
		t.Fatal("docs/qdd-movil-comunicacion.md no dice qué queda guardado en el relay")
	}
	for _, evento := range []string{"`accion`", "`responder`", "`ejecutar`", "`foto`", "realtime.messages"} {
		if !strings.Contains(seccion, evento) {
			t.Errorf("la sección «Qué queda guardado en el relay» no nombra %s", evento)
		}
	}
}
