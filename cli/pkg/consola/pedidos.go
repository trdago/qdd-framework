package consola

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/blueprints"
)

// Rule: Prohibido usar 'else'
// RFC 0006 §3.9: comandos pedidos por la IA. Cuando un comando lo tiene que ejecutar la persona (la IA no tiene
// permiso, o el comando publica algo), la IA lo deja en la cola del proyecto y el panel lo ofrece como un botón.
// Solo corre después de «s  Sí» en el panel, que muestra el comando exacto y su carpeta (consola-claude-code).
// La cola vive en la carpeta central de QDD, no en el proyecto: es de esta máquina y no se versiona.

const (
	TipoPedido = "pedido"

	// MaxPedidos es el tope de la cola: una tecla por pedido en el panel.
	MaxPedidos = 9

	prefijoPedido = "pedido:"
)

// TeclasPedidos son las teclas de los pedidos en el panel, en orden: no usan las de otros botones (1…9 fichas,
// c revisar, u auditar, s y n confirmar, x descartar).
var TeclasPedidos = []string{"a", "b", "d", "e", "f", "g", "h", "i", "j"}

// Pedido es un comando que la IA deja para que la persona lo ejecute desde el panel.
type Pedido struct {
	ID      string    `json:"id"`
	Comando string    `json:"comando"`
	Motivo  string    `json:"motivo"`
	Carpeta string    `json:"carpeta"`
	Creado  time.Time `json:"creado"`
}

var noSlug = regexp.MustCompile(`[^a-z0-9]+`)

// colaPath es el archivo de la cola del proyecto: <QDD_HOME>/claude/pedidos/<carpeta>-<hash>.json.
func colaPath(root string) (string, error) {
	home, err := blueprints.Home()
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(abs))
	slug := strings.Trim(noSlug.ReplaceAllString(strings.ToLower(filepath.Base(abs)), "-"), "-")
	return filepath.Join(home, "claude", "pedidos", slug+"-"+hex.EncodeToString(sum[:])[:12]+".json"), nil
}

// Pedidos devuelve la cola del proyecto, en el orden en que se pidió. Una cola ilegible se trata como vacía.
func Pedidos(root string) []Pedido {
	ruta, err := colaPath(root)
	if err != nil {
		return nil
	}
	b, err := os.ReadFile(ruta)
	if err != nil {
		return nil
	}
	var cola []Pedido
	if json.Unmarshal(b, &cola) != nil {
		return nil
	}
	return cola
}

// Pedir agrega un comando a la cola. Sin carpeta, corre en la raíz del proyecto. El mismo comando en la misma
// carpeta no se duplica: se devuelve el pedido que ya estaba.
func Pedir(root, comando, motivo, carpeta string, now time.Time) (Pedido, error) {
	comando, motivo = strings.TrimSpace(comando), strings.TrimSpace(motivo)
	if comando == "" {
		return Pedido{}, errors.New("falta el comando que la persona debe ejecutar")
	}
	if motivo == "" {
		return Pedido{}, errors.New("falta el motivo: la persona lo lee en el panel antes de confirmar")
	}
	dir, err := carpetaDe(root, carpeta)
	if err != nil {
		return Pedido{}, err
	}
	cola := Pedidos(root)
	for _, p := range cola {
		if p.Comando == comando && p.Carpeta == dir {
			return p, nil
		}
	}
	if len(cola) >= MaxPedidos {
		return Pedido{}, fmt.Errorf("ya hay %d comandos esperando en el panel: que la persona los ejecute o descarte antes de pedir otro", MaxPedidos)
	}
	p := Pedido{ID: siguienteID(cola), Comando: comando, Motivo: motivo, Carpeta: dir, Creado: now}
	return p, guardarCola(root, append(cola, p))
}

func carpetaDe(root, carpeta string) (string, error) {
	dir := strings.TrimSpace(carpeta)
	if dir == "" {
		dir = root
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("la carpeta '%s' no existe", dir)
	}
	return abs, nil
}

// siguienteID numera los pedidos 1, 2, 3…: el mayor de la cola más uno.
func siguienteID(cola []Pedido) string {
	mayor := 0
	for _, p := range cola {
		if n, err := strconv.Atoi(p.ID); err == nil && n > mayor {
			mayor = n
		}
	}
	return strconv.Itoa(mayor + 1)
}

// TeclaDe dice con qué tecla el panel ofrece un pedido ("" si no está en la cola).
func TeclaDe(root, id string) string {
	for _, a := range accionesDePedidos(root) {
		if a.pedido == id {
			return a.Tecla
		}
	}
	return ""
}

// Descartar quita un pedido de la cola sin ejecutarlo.
func Descartar(root, id string) error {
	cola := Pedidos(root)
	resto := make([]Pedido, 0, len(cola))
	for _, p := range cola {
		if p.ID != id {
			resto = append(resto, p)
		}
	}
	if len(resto) == len(cola) {
		return fmt.Errorf("no hay un pedido '%s' en la cola", id)
	}
	return guardarCola(root, resto)
}

// guardarCola escribe la cola de forma atómica (archivo temporal y rename), privada para el usuario.
func guardarCola(root string, cola []Pedido) error {
	ruta, err := colaPath(root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cola, "", "  ")
	if err != nil {
		return err
	}
	tmp := ruta + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, ruta)
}

func accionesDePedidos(root string) []Accion {
	var acciones []Accion
	for i, p := range Pedidos(root) {
		if i >= MaxPedidos {
			break
		}
		acciones = append(acciones, Accion{
			ID: prefijoPedido + p.ID, Tipo: TipoPedido, Etiqueta: p.Motivo, Detalle: p.Comando, Carpeta: p.Carpeta,
			Tecla: TeclasPedidos[i], Confirmar: true, pedido: p.ID,
		})
	}
	return acciones
}

// ejecutarPedido corre el comando con la shell del sistema en su carpeta. Si termina bien, sale de la cola; si
// falla, queda para reintentarlo o descartarlo.
func ejecutarPedido(root string, a Accion, salida io.Writer) int {
	shell, flag := "sh", "-c"
	if runtime.GOOS == "windows" {
		shell, flag = "cmd", "/C"
	}
	c := exec.Command(shell, flag, a.Detalle)
	c.Dir = a.Carpeta
	c.Stdout = salida
	c.Stderr = salida
	codigo := codigoDe(c.Run(), a, salida)
	if codigo != 0 {
		return codigo
	}
	if err := Descartar(root, a.pedido); err != nil {
		fmt.Fprintf(salida, "[!] el comando terminó bien, pero no se pudo quitar de la cola: %v\n", err)
	}
	return 0
}

func codigoDe(err error, a Accion, salida io.Writer) int {
	var salio *exec.ExitError
	if errors.As(err, &salio) {
		return salio.ExitCode()
	}
	if err != nil {
		fmt.Fprintf(salida, "[!] no se pudo ejecutar %s: %v\n", a.Detalle, err)
		return 1
	}
	return 0
}
