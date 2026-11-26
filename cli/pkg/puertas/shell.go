package puertas

import "strings"

// lector parte un comando de shell en comandos simples (sus palabras), lo justo para ver qué programa corre cada uno:
// separa por && || ; | & ( ) y salto de línea, junta lo que va entre comillas y salta el cuerpo de un heredoc y el
// destino de una redirección. No expande nada.
type lector struct {
	texto       []rune
	i           int
	comandos    [][]string
	actual      []string
	palabra     strings.Builder
	hay         bool     // la palabra en curso existe (también "" entre comillas)
	heredocs    []string // delimitadores que esperan el próximo salto de línea
	redireccion bool     // la próxima palabra es el destino de una redirección
}

func comandosDe(comando string) [][]string {
	l := &lector{texto: []rune(comando)}
	for l.i < len(l.texto) {
		l.paso()
	}
	l.cerrarComando()
	return l.comandos
}

func (l *lector) paso() {
	r := l.texto[l.i]
	if r == '\n' {
		l.cerrarComando()
		l.i++
		l.saltarHeredocs()
		return
	}
	if r == ' ' || r == '\t' {
		l.cerrarPalabra()
		l.i++
		return
	}
	if strings.ContainsRune(";&|()", r) {
		l.cerrarComando()
		l.i++
		return
	}
	if r == '<' || r == '>' {
		l.redirigir()
		return
	}
	l.leerPalabra()
}

// redirigir: <<DELIM abre un heredoc; < > >> 2> llevan un destino que no es parte del comando.
func (l *lector) redirigir() {
	if l.palabra.Len() > 0 && strings.Trim(l.palabra.String(), "0123456789") == "" {
		l.palabra.Reset()
		l.hay = false
	}
	l.cerrarPalabra()
	heredoc := l.viene("<<")
	for l.i < len(l.texto) && strings.ContainsRune("<>&-", l.texto[l.i]) {
		l.i++
	}
	if !heredoc {
		l.redireccion = true
		return
	}
	for l.i < len(l.texto) && (l.texto[l.i] == ' ' || l.texto[l.i] == '\t') {
		l.i++
	}
	l.leerPalabra()
	l.heredocs = append(l.heredocs, l.palabra.String())
	l.palabra.Reset()
	l.hay = false
}

// saltarHeredocs salta las líneas de cada heredoc pendiente hasta su delimitador.
func (l *lector) saltarHeredocs() {
	for _, delim := range l.heredocs {
		for l.i < len(l.texto) {
			fin := l.i
			for fin < len(l.texto) && l.texto[fin] != '\n' {
				fin++
			}
			linea := strings.TrimLeft(string(l.texto[l.i:fin]), "\t")
			l.i = min(fin+1, len(l.texto))
			if linea == delim {
				break
			}
		}
	}
	l.heredocs = nil
}

// leerPalabra lee una palabra hasta un espacio u operador: las comillas y $( ) se juntan enteras.
func (l *lector) leerPalabra() {
	for l.i < len(l.texto) {
		r := l.texto[l.i]
		if r == ' ' || r == '\t' || r == '\n' || strings.ContainsRune(";&|()<>", r) {
			return
		}
		l.hay = true
		if r == '\'' {
			l.hasta('\'', false)
			continue
		}
		if r == '"' {
			l.hasta('"', true)
			continue
		}
		if r == '\\' && l.i+1 < len(l.texto) {
			l.palabra.WriteRune(l.texto[l.i+1])
			l.i += 2
			continue
		}
		if r == '$' && l.i+1 < len(l.texto) && l.texto[l.i+1] == '(' {
			l.subcomando()
			continue
		}
		l.palabra.WriteRune(r)
		l.i++
	}
}

// hasta junta lo que va entre comillas, sin ellas; entre comillas dobles \ escapa la siguiente.
func (l *lector) hasta(cierre rune, escapes bool) {
	l.i++
	for l.i < len(l.texto) && l.texto[l.i] != cierre {
		if escapes && l.texto[l.i] == '\\' && l.i+1 < len(l.texto) {
			l.i++
		}
		l.palabra.WriteRune(l.texto[l.i])
		l.i++
	}
	l.i++
}

// subcomando junta $( ... ) entero en la palabra: su salida es texto del comando, no otro comando.
func (l *lector) subcomando() {
	nivel := 0
	for l.i < len(l.texto) {
		r := l.texto[l.i]
		l.palabra.WriteRune(r)
		l.i++
		if r == '(' {
			nivel++
		}
		if r == ')' {
			nivel--
		}
		if nivel == 0 && r == ')' {
			return
		}
	}
}

func (l *lector) viene(s string) bool {
	return strings.HasPrefix(string(l.texto[l.i:min(l.i+len(s), len(l.texto))]), s)
}

func (l *lector) cerrarPalabra() {
	if !l.hay {
		return
	}
	if !l.redireccion {
		l.actual = append(l.actual, l.palabra.String())
	}
	l.redireccion = false
	l.palabra.Reset()
	l.hay = false
}

func (l *lector) cerrarComando() {
	l.cerrarPalabra()
	if len(l.actual) > 0 {
		l.comandos = append(l.comandos, l.actual)
	}
	l.actual = nil
}
