package incidentes

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Rule: Prohibido usar 'else'

const moduloQDD = "github.com/qdd-framework/qdd/"

// Traza deja de una pila de Go (debug.Stack) solo las funciones de QDD, con su archivo relativo al módulo
// (pkg/estado/estado.go:123): nunca una ruta de la máquina. Con un pánico, empieza después de «panic(»: lo de antes es
// la maquinaria que lo atrapó.
func Traza(pila []byte) []string {
	lineas := strings.Split(string(pila), "\n")
	desde := 0
	for n, l := range lineas {
		if strings.HasPrefix(l, "panic(") {
			desde = n + 1
		}
	}
	var out []string
	for n := desde; n+1 < len(lineas) && len(out) < maxTraza; n++ {
		funcion, ok := strings.CutPrefix(lineas[n], moduloQDD)
		if !ok || strings.HasPrefix(funcion, "pkg/incidentes.") {
			continue
		}
		out = append(out, fmt.Sprintf("%s (%s)", sinArgumentos(funcion), archivoDelModulo(lineas[n+1])))
	}
	return out
}

func sinArgumentos(f string) string {
	if i := strings.LastIndex(f, "("); i > 0 && strings.HasSuffix(f, ")") {
		return f[:i]
	}
	return f
}

// archivoDelModulo toma «\t/ruta/.../cli/pkg/x/y.go:12 +0x1f» y deja «pkg/x/y.go:12».
func archivoDelModulo(l string) string {
	l = strings.TrimSpace(l)
	l, _, _ = strings.Cut(l, " +0x")
	for _, ancla := range []string{"/cli/", "/qdd-framework/qdd@"} {
		i := strings.LastIndex(l, ancla)
		if i < 0 {
			continue
		}
		resto := l[i+len(ancla):]
		if ancla != "/cli/" {
			_, resto, _ = strings.Cut(resto, "/")
		}
		return resto
	}
	return l[strings.LastIndex(l, "/")+1:]
}

// Titulo es el del issue: el tipo, dónde y el comienzo del mensaje.
func Titulo(i Incidente) string {
	t := fmt.Sprintf("[incidente] %s en %s: %s", nombreDelTipo(i.Tipo), i.Origen, i.Mensaje)
	if r := []rune(t); len(r) > 100 {
		return string(r[:99]) + "…"
	}
	return t
}

func nombreDelTipo(tipo string) string {
	if tipo == TipoAutorreparacion {
		return "autorreparación"
	}
	return "pánico"
}

// Cuerpo es el issue en el formato estándar qdd-incidente/1: lo que pasó, dónde en el código de QDD, cómo reproducirlo
// y un bloque YAML con todo, para que una IA lo lea sin interpretar prosa. La firma va en un comentario: así se
// encuentra el issue de un defecto que ya se publicó.
func Cuerpo(i Incidente) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<!-- qdd-incidente firma=%s formato=%s -->\n\n## Qué pasó\n\n", i.Firma, Formato)
	fmt.Fprintf(&b, "%s en `%s` (qdd %s, %s): %d ocurrencia(s) desde %s.\n\n> %s\n\n", strings.ToUpper(nombreDelTipo(i.Tipo)[:1])+nombreDelTipo(i.Tipo)[1:], i.Origen, i.Version, i.SO, i.Ocurrencias, dia(i.Primera), i.Mensaje)
	if i.Reparacion != "" {
		fmt.Fprintf(&b, "QDD se reparó solo: %s. La reparación funciona, pero el estado que la pidió no debió darse.\n\n", i.Reparacion)
	}
	b.WriteString("## Dónde (código de QDD)\n\n")
	if len(i.Traza) == 0 {
		b.WriteString("Sin traza: el origen de arriba es el punto de entrada.\n")
	}
	for n, f := range i.Traza {
		fmt.Fprintf(&b, "%d. `%s`\n", n+1, f)
	}
	b.WriteString("\n## Cómo reproducir\n\n")
	b.WriteString(reproducir(i))
	b.WriteString("\n## Para la IA que lo repara\n\n")
	b.WriteString("Arma un escenario sintético mínimo que llegue a la primera línea de «Dónde» (los datos del proyecto no viajan: rutas, nombres y valores van como `<marcador>`). Regístralo como finding con su prueba de regresión, que falla antes del arreglo (método QDD), y cierra este issue desde el PR.\n\n")
	b.WriteString("```yaml\n")
	b.WriteString(bloque(i))
	b.WriteString("```\n\n<sub>Publicado por QDD (qdd-incidente/1). Solo datos de QDD; se apaga con QDD_INCIDENTES=off.</sub>\n")
	return b.String()
}

func reproducir(i Incidente) string {
	if i.Tipo == TipoAutorreparacion {
		return fmt.Sprintf("1. Con qdd %s, deja en un proyecto de prueba el estado que describe «Qué pasó».\n2. Corre `qdd doctor`: falla en «%s»; `qdd doctor --fix` lo repara.\n", i.Version, i.Origen)
	}
	comando := i.Comando
	if comando == "" {
		comando = i.Origen
	}
	return fmt.Sprintf("1. Instala qdd %s (%s).\n2. En un proyecto de prueba: `qdd init`.\n3. Corre `%s` con valores sintéticos en cada `<marcador>`.\n", i.Version, i.SO, comando)
}

func bloque(i Incidente) string {
	data, _ := yaml.Marshal(struct {
		Formato     string   `yaml:"formato"`
		Firma       string   `yaml:"firma"`
		Tipo        string   `yaml:"tipo"`
		Origen      string   `yaml:"origen"`
		Comando     string   `yaml:"comando,omitempty"`
		Mensaje     string   `yaml:"mensaje"`
		Traza       []string `yaml:"traza,omitempty"`
		Reparacion  string   `yaml:"reparacion,omitempty"`
		Version     string   `yaml:"version"`
		SO          string   `yaml:"so"`
		Ocurrencias int      `yaml:"ocurrencias"`
		Primera     string   `yaml:"primera"`
		Ultima      string   `yaml:"ultima"`
	}{Formato, i.Firma, i.Tipo, i.Origen, i.Comando, i.Mensaje, i.Traza, i.Reparacion, i.Version, i.SO, i.Ocurrencias, i.Primera, i.Ultima})
	return string(data)
}

// Ocurrencia es el comentario cuando el defecto ya tiene issue: cuántas veces más y con qué versión.
func Ocurrencia(i Incidente) string {
	return fmt.Sprintf("<!-- qdd-incidente firma=%s -->\nOtra vez: %d ocurrencia(s) en total (+%d), la última el %s con qdd %s (%s).\n", i.Firma, i.Ocurrencias, i.Ocurrencias-i.Publicadas, dia(i.Ultima), i.Version, i.SO)
}

func dia(rfc string) string {
	d, _, _ := strings.Cut(rfc, "T")
	return d
}
