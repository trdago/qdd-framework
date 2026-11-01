package herramientas

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'

const (
	topePrueba = 10 * time.Second // la prueba de humo
	topeAyuda  = 5 * time.Second  // cada intento de leer la ayuda
	topeTexto  = 64 << 10         // lo que se guarda de la ayuda o de una prueba que falla
)

// Opciones de Correr.
type Opciones struct {
	Confirmado bool // la persona dijo «sí» a una acción con confirmar
	Salida     io.Writer
	Errores    io.Writer
}

// Correr ejecuta una acción sin shell, en la raíz del proyecto y con el entorno de la máquina. Devuelve su código de
// salida; el error es que no corrió (falta confirmar, un campo o una variable, o no se pudo lanzar).
func Correr(ctx context.Context, root string, h Herramienta, accion string, valores map[string]string, o Opciones) (int, error) {
	a, err := accionDe(h, accion)
	if err != nil {
		return -1, err
	}
	if a.Confirmar && !o.Confirmado {
		return -1, fmt.Errorf("«%s» pide confirmación antes de correr (en la CLI, --si)", a.Nombre)
	}
	args, err := Argumentos(a, valores)
	if err != nil {
		return -1, err
	}
	if err := entornoCompleto(h); err != nil {
		return -1, err
	}
	cmd := comando(ctx, root, append(append([]string{}, h.Comando...), args...))
	cmd.Stdout, cmd.Stderr = o.Salida, o.Errores
	return salida(cmd.Run())
}

func accionDe(h Herramienta, id string) (Accion, error) {
	var ids []string
	for _, a := range h.Acciones {
		if a.ID == id {
			return a, nil
		}
		ids = append(ids, a.ID)
	}
	return Accion{}, fmt.Errorf("%s no tiene la acción «%s» (tiene: %s)", h.ID, id, strings.Join(ids, ", "))
}

// Argumentos arma los argumentos de una acción con lo que completó la persona. Cada {{campo}} es un argumento entero;
// un grupo va solo si todos sus campos tienen valor (un si-no, si dice «si»), y el si-no no se pasa como argumento.
func Argumentos(a Accion, valores map[string]string) ([]string, error) {
	campos := map[string]Campo{}
	for _, c := range a.Campos {
		campos[c.ID] = c
	}
	for id := range valores {
		if _, ok := campos[id]; !ok {
			return nil, fmt.Errorf("«%s» no es un campo de %s", id, a.ID)
		}
	}
	finales := map[string]string{}
	for _, c := range a.Campos {
		v, err := valorDe(c, valores[c.ID])
		if err != nil {
			return nil, fmt.Errorf("%s: %v", a.ID, err)
		}
		finales[c.ID] = v
	}
	var out []string
	for _, arg := range a.Args {
		partes, va := expandir(arg, campos, finales)
		if va {
			out = append(out, partes...)
		}
	}
	return out, nil
}

func valorDe(c Campo, v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		v = c.PorDefecto
	}
	if v == "" && c.Requerido {
		return "", fmt.Errorf("falta «%s»", nombreDe(c))
	}
	if v == "" {
		return "", nil
	}
	return v, valorValido(c, v)
}

func valorValido(c Campo, v string) error {
	if c.Tipo == "numero" {
		if _, err := strconv.ParseFloat(v, 64); err != nil {
			return fmt.Errorf("«%s» no es un número", nombreDe(c))
		}
	}
	if c.Tipo == "opcion" && !inList(c.Opciones, v) {
		return fmt.Errorf("«%s» no es una opción de %s (%s)", v, nombreDe(c), strings.Join(c.Opciones, ", "))
	}
	if c.Tipo == "si-no" && v != "si" && v != "no" {
		return fmt.Errorf("%s es «si» o «no»", nombreDe(c))
	}
	return nil
}

func nombreDe(c Campo) string {
	if c.Nombre != "" {
		return c.Nombre
	}
	return c.ID
}

// expandir reemplaza los campos de un argumento; va es si el argumento (o el grupo) se pasa.
func expandir(arg Arg, campos map[string]Campo, valores map[string]string) ([]string, bool) {
	var out []string
	for _, parte := range arg.Partes {
		m := campoEntero.FindStringSubmatch(parte)
		if m == nil {
			out = append(out, parte)
			continue
		}
		v := valores[m[1]]
		if v == "" || (v == "no" && campos[m[1]].Tipo == "si-no") {
			return nil, false
		}
		if campos[m[1]].Tipo != "si-no" {
			out = append(out, v)
		}
	}
	return out, true
}

// entornoCompleto dice qué variables de entorno faltan, sin mostrar ningún valor.
func entornoCompleto(h Herramienta) error {
	var faltan []string
	for _, v := range h.Entorno {
		if os.Getenv(v) == "" {
			faltan = append(faltan, v)
		}
	}
	if len(faltan) > 0 {
		return fmt.Errorf("falta la variable de entorno %s: defínela en esta máquina (su valor nunca va en la definición)", strings.Join(faltan, ", "))
	}
	return nil
}

// validarComando revisa que el ejecutable exista: dentro del proyecto si es relativo, en su ruta si es absoluto o en
// el PATH si es un nombre (npm, make, go).
func validarComando(root string, argv []string) []string {
	if len(argv) == 0 {
		return nil
	}
	c := argv[0]
	if !strings.ContainsAny(c, `/\`) {
		if _, err := exec.LookPath(c); err != nil {
			return []string{fmt.Sprintf("«%s» no está en el PATH de esta máquina", c)}
		}
		return nil
	}
	path := rutaDe(root, c)
	if !filepath.IsAbs(c) && !dentro(root, path) {
		return []string{fmt.Sprintf("«%s» queda fuera del proyecto: usa una ruta dentro de él o la ruta absoluta que elegiste", c)}
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return []string{fmt.Sprintf("«%s» no existe", c)}
	}
	if !ejecutable(info) {
		return []string{fmt.Sprintf("«%s» no es ejecutable", c)}
	}
	return nil
}

func rutaDe(root, c string) string {
	if filepath.IsAbs(c) || !strings.ContainsAny(c, `/\`) {
		return c
	}
	return filepath.Join(root, filepath.FromSlash(c))
}

func dentro(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// comando arma la ejecución: sin shell, sin entrada, en la raíz del proyecto.
func comando(ctx context.Context, root string, argv []string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, rutaDe(root, argv[0]), argv[1:]...)
	cmd.Dir = root
	cmd.WaitDelay = time.Second
	enGrupo(cmd)
	return cmd
}

func salida(err error) (int, error) {
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), nil
	}
	if err != nil {
		return -1, err
	}
	return 0, nil
}

// PruebaDeHumo corre el comando con los argumentos de su prueba: debe salir con 0.
func PruebaDeHumo(root string, h Herramienta) error {
	if h.Prueba == nil {
		return fmt.Errorf("falta la prueba de humo")
	}
	if err := entornoCompleto(h); err != nil {
		return fmt.Errorf("no se puede probar: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), topePrueba)
	defer cancel()
	argv := append(append([]string{}, h.Comando...), h.Prueba.Args...)
	var out bytes.Buffer
	cmd := comando(ctx, root, argv)
	cmd.Stdout, cmd.Stderr = &out, &out
	code, err := salida(cmd.Run())
	if ctx.Err() != nil {
		return fmt.Errorf("su prueba de humo (%s) no terminó en %s: ¿espera entrada? Una herramienta interactiva no se puede usar desde el panel", strings.Join(argv, " "), topePrueba)
	}
	if err != nil || code != 0 {
		return fmt.Errorf("su prueba de humo (%s) salió con %d: %s", strings.Join(argv, " "), code, ultimasLineas(out.String(), 5))
	}
	return nil
}

// Ayuda corre <comando> --help (y si no, -h y help) con un tiempo límite y sin entrada, para que la IA lea qué pide.
func Ayuda(ctx context.Context, root string, argv []string) (string, error) {
	if problemas := validarComando(root, argv); len(problemas) > 0 {
		return "", errors.New(problemas[0])
	}
	primera := ""
	for _, intento := range []string{"--help", "-h", "help"} {
		texto, code := intentarAyuda(ctx, root, append(append([]string{}, argv...), intento))
		if code == 0 && strings.TrimSpace(texto) != "" {
			return texto, nil
		}
		if primera == "" {
			primera = texto
		}
	}
	return primera, fmt.Errorf("ni --help, ni -h, ni help salieron con 0: su ayuda es lo de arriba, o léela en su código")
}

func intentarAyuda(ctx context.Context, root string, argv []string) (string, int) {
	ctx, cancel := context.WithTimeout(ctx, topeAyuda)
	defer cancel()
	var out bytes.Buffer
	cmd := comando(ctx, root, argv)
	cmd.Stdout, cmd.Stderr = &out, &out
	code, err := salida(cmd.Run())
	texto := out.String()
	if len(texto) > topeTexto {
		texto = texto[:topeTexto]
	}
	if err != nil || ctx.Err() != nil {
		return texto, -1
	}
	return texto, code
}

func ultimasLineas(texto string, n int) string {
	lineas := strings.Split(strings.TrimSpace(texto), "\n")
	if len(lineas) > n {
		lineas = lineas[len(lineas)-n:]
	}
	return strings.Join(lineas, " · ")
}
