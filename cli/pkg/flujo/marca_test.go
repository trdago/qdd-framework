package flujo

import (
	"regexp"
	"slices"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// Pedido de la persona (2026-11-08): «lo que mandamos tiene que estar todo integrado en el html, el javascript + css +
// icons, siempre respetando el branding de qdd». La marca es la de qdd-movile docs/marca: Gruvbox Material solo en
// oscuro, JetBrains Mono 400/500/700 empaquetada, esquinas de 2 px, borde de 1 px, sin sombras, el isotipo «▶_», el
// ámbar solo para lo que espera a la persona y emojis de una lista cerrada.

// tokensDeMarca son los colores de qdd-movile docs/marca/tokens.json (semántico → valor).
var tokensDeMarca = map[string]string{
	"--fondo": "#282828", "--superficie": "#32302f", "--superficie-alta": "#3c3836", "--borde": "#504945",
	"--borde-campo": "#928374", "--texto": "#d4be98", "--secundario": "#a89984", "--ambar": "#d8a657",
	"--en-curso": "#7daea3", "--bien": "#a9b665", "--falla": "#ea6962", "--sobre-texto": "#282828",
	"--sobre-marca": "#282828", "--velo": "#28282899",
}

var (
	colorLiteral = regexp.MustCompile(`#[0-9a-fA-F]{3,8}\b|rgba?\(`)
	bloqueRaiz   = regexp.MustCompile(`(?s)^:root \{.*?\n\}\n`)
)

func archivoDePrueba(t *testing.T) string {
	t.Helper()
	html, err := ArchivoHTML(ArmarFoto(flujoConGrupos(), limpiezaDePrueba, ahoraDePrueba))
	if err != nil {
		t.Fatal(err)
	}
	return string(html)
}

func TestMarca_LosColoresSonLosTokensYSoloEnLaRaiz(t *testing.T) {
	raiz := bloqueRaiz.FindString(archivoCSS)
	if raiz == "" {
		t.Fatal("los tokens van al principio, en :root")
	}
	for nombre, valor := range tokensDeMarca {
		if !strings.Contains(raiz, nombre+": "+valor+";") {
			t.Fatalf("falta el token %s: %s", nombre, valor)
		}
	}
	if resto := strings.TrimPrefix(archivoCSS, raiz); colorLiteral.MatchString(resto) {
		t.Fatalf("fuera de :root no hay colores literales: %q", colorLiteral.FindString(resto))
	}
	if colorLiteral.MatchString(archivoJS) {
		t.Fatalf("el script no trae colores: %q", colorLiteral.FindString(archivoJS))
	}
}

func TestMarca_SoloOscuroSinSombrasYConEsquinasDeLaFirma(t *testing.T) {
	for _, prohibido := range []string{"prefers-color-scheme", "data-theme", "box-shadow: 0", "linear-gradient", "radial-gradient"} {
		if strings.Contains(archivoCSS, prohibido) {
			t.Fatalf("la marca es solo oscura, sin sombras ni gradientes: %q", prohibido)
		}
	}
	for _, r := range regexp.MustCompile(`border-radius: ([^;]+);`).FindAllStringSubmatch(archivoCSS, -1) {
		if r[1] != "var(--radio)" {
			t.Fatalf("todas las esquinas son la de la firma (2 px): %s", r[0])
		}
	}
	if !strings.Contains(archivoCSS, "--radio: 2px;") || !strings.Contains(archivoCSS, "color-scheme: dark;") {
		t.Fatal("la firma: radio de 2 px y esquema oscuro")
	}
}

func TestMarca_LaFuenteVaDentroDelArchivo(t *testing.T) {
	html := archivoDePrueba(t)
	for _, peso := range []string{"400", "500", "700"} {
		if !strings.Contains(html, "font-weight: "+peso+";") {
			t.Fatalf("JetBrains Mono %s va en el archivo", peso)
		}
	}
	if strings.Count(html, "src: url(data:font/woff2;base64,") != 3 || !strings.Contains(html, "font-src data:") {
		t.Fatal("las tres fuentes van como data: y la CSP las deja cargar")
	}
	if !strings.Contains(archivoCSS, `font-family: "JetBrains Mono", ui-monospace, monospace;`) {
		t.Fatal("una sola familia, monoespaciada")
	}
}

func TestMarca_IsotipoEIconosDeCadaTipo(t *testing.T) {
	html := archivoDePrueba(t)
	if !strings.Contains(html, `points="10,12 34,32 10,52"`) || !strings.Contains(html, `>QDD<`) {
		t.Fatal("la cabecera lleva el isotipo «▶_» y el nombre QDD")
	}
	for _, p := range Plantillas {
		for _, tn := range p.Tipos {
			if !strings.Contains(archivoJS, "'"+tn.ID+"': '") {
				t.Fatalf("el tipo %s (%s) tiene su ícono SVG en el archivo", tn.ID, p.ID)
			}
		}
	}
	if !strings.Contains(archivoJS, "grupo: '") {
		t.Fatal("y la caja de un grupo plegado también")
	}
}

// listaCerradaDeEmojis es la de qdd-movile docs/marca/voz.md.
var listaCerradaDeEmojis = []rune{'✅', '⏳', '❌', '✓', '✕', '●', '▶', '›', '·', '…', '└'}

func TestMarca_SinEmojisFueraDeLaListaCerrada(t *testing.T) {
	textos := []string{archivoJS, archivoHTML}
	for _, p := range Plantillas {
		for _, tn := range p.Tipos {
			textos = append(textos, tn.Icono, tn.Explica)
		}
	}
	for _, s := range textos {
		for _, r := range s {
			if r < 0x2600 || slices.Contains(listaCerradaDeEmojis, r) {
				continue
			}
			if r >= 0x1F000 || (r >= 0x2600 && r <= 0x27BF) {
				t.Fatalf("emoji fuera de la lista cerrada de la marca: %q", r)
			}
		}
	}
}
