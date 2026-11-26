package baul

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
)

// Rule: Prohibido usar 'else'
// usar pone el token solo en el entorno del comando y no lo escribe en ningún archivo. La salida pasa por el ocultador,
// que tapa cualquier valor del baúl. No existe «qdd baul ver».

// Terminal es la terminal de la persona, sin sesión de Claude Code: allí se pregunta.
type Terminal struct {
	PID       int
	Preguntar func(pregunta string) bool
}

// OpcionesUsar es un uso: qué token, para qué, en qué proyecto, desde qué sesión o terminal, y el comando.
type OpcionesUsar struct {
	Root     string
	Nombre   string
	Fin      string
	Sesion   string
	Terminal *Terminal
	Comando  []string
	Entrada  io.Reader
	Salida   io.Writer
	Ahora    time.Time
}

// Usar corre el comando con el token si hay permiso. Sin permiso: deja el pedido y devuelve CodigoTodaviaNo con
// ErrSinPermiso.
func Usar(o OpcionesUsar) (int, error) {
	if strings.TrimSpace(o.Fin) == "" {
		return 1, fmt.Errorf("falta el fin: --para \"<para qué>\"")
	}
	if len(o.Comando) == 0 {
		return 1, fmt.Errorf("falta el comando: qdd baul usar <nombre> --para \"<fin>\" -- <comando>")
	}
	e, err := Buscar(o.Nombre)
	if err != nil {
		return 1, err
	}
	if err := autorizar(o, e); err != nil {
		if errors.Is(err, ErrSinPermiso) {
			return CodigoTodaviaNo, err
		}
		return 1, err
	}
	valor, err := leerValor(e)
	if err != nil {
		return 1, err
	}
	codigo := correr(o, e, valor)
	quien := o.Sesion
	if quien == "" {
		quien = "persona"
	}
	_ = cambiarEntrada(e.Nombre, func(x *Entrada) { t := o.Ahora.UTC(); x.Usado = &t })
	ocultos := tapadorDeTodos(valor)
	comando := ocultos.tapar([]byte(strings.Join(o.Comando, " ")))
	return codigo, Anotar(Linea{Quien: quien, Proyecto: o.Root, Accion: "usar", Token: e.Nombre, Fin: o.Fin, Comando: string(comando), Codigo: &codigo}, o.Ahora)
}

// autorizar: con permiso sigue; en la terminal pregunta; desde una sesión que late deja el pedido.
func autorizar(o OpcionesUsar, e Entrada) error {
	pid := 0
	if o.Sesion == "" && o.Terminal != nil {
		pid = o.Terminal.PID
	}
	hay, err := tienePermiso(o.Sesion, pid, o.Root, e.Nombre, o.Fin, o.Ahora)
	if err != nil || hay {
		return err
	}
	if o.Sesion == "" && o.Terminal == nil {
		return fmt.Errorf("sin sesión de Claude Code ni terminal no hay a quién pedirle permiso: corre qdd baul usar en tu terminal")
	}
	if o.Sesion == "" {
		pregunta := fmt.Sprintf("¿Permitir %s para «%s» en esta terminal, hasta cerrarla?", e.Nombre, o.Fin)
		if !o.Terminal.Preguntar(pregunta) {
			return fmt.Errorf("no se permitió")
		}
		return permitirTerminal(pid, o.Root, e.Nombre, o.Fin, o.Ahora)
	}
	if !consola.SesionLate(o.Root, o.Sesion, o.Ahora) {
		return fmt.Errorf("la sesión %s no late en este proyecto: el permiso se pide en el panel y dura lo que la consola (qdd consola activar)", o.Sesion)
	}
	if err := pedirPermiso(o.Root, o.Sesion, e.Nombre, o.Fin, o.Ahora); err != nil {
		return err
	}
	return fmt.Errorf("%w: quedó el pedido «🔑 %s» en el panel; corre de nuevo cuando la persona diga «Sí»", ErrSinPermiso, e.Nombre)
}

// tapadorDeTodos tapa el valor que se usa y los demás del baúl que se puedan leer sin preguntar (los del llavero).
func tapadorDeTodos(valor string) *Ocultador {
	valores := []string{valor}
	indice, _ := Indice()
	for _, x := range indice {
		if x.Donde != EnLlavero || x.Tipo == "aws-sso" {
			continue
		}
		if v, err := leerValor(x); err == nil {
			valores = append(valores, v)
		}
	}
	return NuevoOcultador(io.Discard, valores)
}

func correr(o OpcionesUsar, e Entrada, valor string) int {
	salida := o.Salida
	if salida == nil {
		salida = os.Stdout
	}
	tapa := tapadorDeTodos(valor)
	tapa.destino = salida
	defer tapa.Close()
	c := exec.Command(o.Comando[0], o.Comando[1:]...)
	c.Dir = o.Root
	c.Env = append(sinVariable(os.Environ(), e.Variable), e.Variable+"="+valor)
	c.Stdin, c.Stdout, c.Stderr = o.Entrada, tapa, tapa
	err := c.Run()
	var salidaErr *exec.ExitError
	if errors.As(err, &salidaErr) {
		return salidaErr.ExitCode()
	}
	if err != nil {
		fmt.Fprintf(tapa, "[!] no se pudo correr %s: %v\n", o.Comando[0], err)
		return 1
	}
	return 0
}

func sinVariable(env []string, variable string) []string {
	var out []string
	for _, kv := range env {
		if strings.HasPrefix(kv, variable+"=") {
			continue
		}
		out = append(out, kv)
	}
	return out
}
