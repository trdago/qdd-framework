package consola

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// RFC 0006 §3.9: comandos pedidos por la IA. Cuando un comando lo tiene que ejecutar la persona (la IA no tiene
// permiso, o el comando publica algo), la IA lo deja en la cola del proyecto y el panel lo ofrece como un botón.
// Solo corre después de «s  Sí» en el panel, que muestra el comando exacto y su carpeta (consola-claude-code).
// La cola vive en la carpeta central de QDD, no en el proyecto: es de esta máquina y no se versiona.
//
// Un pedido pasa por pendiente → ejecutando → hecho o fallo. Solo uno pendiente o con fallo se puede ejecutar: uno
// aceptado no vuelve a correr (FND-070). Cada pedido guarda su tecla desde que se pide (FND-071).

const (
	TipoPedido = "pedido"

	// MaxPedidos es el tope de pedidos abiertos (no hechos): una tecla por pedido en el panel.
	MaxPedidos = 9

	EstadoPendiente  = "pendiente"
	EstadoEjecutando = "ejecutando"
	EstadoHecho      = "hecho"
	EstadoFallo      = "fallo"

	// VidaHecho es cuánto se sigue viendo, en gris, un pedido hecho.
	VidaHecho = 30 * time.Minute
	// TopeEjecucion es cuánto puede estar ejecutándose un pedido antes de darlo por cortado.
	TopeEjecucion = 15 * time.Minute

	prefijoPedido = "pedido:"
	lineasSalida  = 4
)

// TeclasPedidos son las teclas de los pedidos en el panel: no usan las de otros botones (1…9 fichas, c revisar,
// u auditar, s y n confirmar, x descartar).
var TeclasPedidos = []string{"a", "b", "d", "e", "f", "g", "h", "i", "j"}

// Pedido es un comando que la IA deja para que la persona lo ejecute desde el panel.
type Pedido struct {
	ID      string     `json:"id"`
	Comando string     `json:"comando"`
	Motivo  string     `json:"motivo"`
	Carpeta string     `json:"carpeta"`
	Creado  time.Time  `json:"creado"`
	Tecla   string     `json:"tecla,omitempty"`
	Estado  string     `json:"estado,omitempty"` // vacío en una cola de qdd 2.6.0 = pendiente
	Inicio  *time.Time `json:"inicio,omitempty"`
	Fin     *time.Time `json:"fin,omitempty"`
	Codigo  int        `json:"codigo,omitempty"`
	Salida  string     `json:"salida,omitempty"` // últimas líneas de la salida
}

// Abierto dice si el pedido todavía cuenta en la cola: no está hecho.
func (p Pedido) Abierto() bool {
	return p.Estado != EstadoHecho
}

// Ejecutable dice si la persona lo puede ejecutar: pendiente o con fallo (para reintentarlo).
func (p Pedido) Ejecutable() bool {
	return p.Estado == EstadoPendiente || p.Estado == EstadoFallo
}

// RutaLocal es un archivo local del proyecto, fuera del repositorio: <QDD_HOME>/claude/<carpeta>/<slug>-<hash>.json.
var RutaLocal = estado.RutaLocal

// colaPath es el archivo de la cola del proyecto: <QDD_HOME>/claude/pedidos/<slug>-<hash>.json.
func colaPath(root string) (string, error) {
	return RutaLocal(root, "pedidos")
}

// Pedidos devuelve la cola del proyecto en el orden en que se pidió, ya normalizada a la hora now: los hechos de
// hace más de VidaHecho no aparecen y uno ejecutándose desde hace más de TopeEjecucion se da por cortado. Una cola
// ilegible se trata como vacía.
func Pedidos(root string) []Pedido {
	return PedidosA(root, time.Now())
}

// PedidosA es Pedidos a una hora dada.
func PedidosA(root string, now time.Time) []Pedido {
	return normalizar(leerCola(root), now)
}

func leerCola(root string) []Pedido {
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

func normalizar(cola []Pedido, now time.Time) []Pedido {
	out := make([]Pedido, 0, len(cola))
	usadas := map[string]bool{}
	for _, p := range cola {
		if p.Estado == "" {
			p.Estado = EstadoPendiente
		}
		if p.Estado == EstadoHecho && p.Fin != nil && now.Sub(*p.Fin) > VidaHecho {
			continue
		}
		if p.Estado == EstadoEjecutando && p.Inicio != nil && now.Sub(*p.Inicio) > TopeEjecucion {
			p.Estado, p.Codigo = EstadoFallo, -1
			p.Salida = strings.TrimSpace(p.Salida + "\nse cortó: no terminó en " + TopeEjecucion.String())
		}
		if p.Abierto() && p.Tecla != "" {
			usadas[p.Tecla] = true
		}
		out = append(out, p)
	}
	// Una cola de qdd 2.6.0 no guardaba teclas: se reparten las libres, en orden.
	for i := range out {
		if out[i].Abierto() && out[i].Tecla == "" {
			out[i].Tecla = teclaLibre(usadas)
			usadas[out[i].Tecla] = true
		}
	}
	return out
}

func teclaLibre(usadas map[string]bool) string {
	for _, t := range TeclasPedidos {
		if !usadas[t] {
			return t
		}
	}
	return ""
}

// conCola lee, cambia y guarda la cola con el archivo bloqueado: dos procesos (la IA pidiendo y el panel
// ejecutando) nunca se pisan, y un pedido no se acepta dos veces.
func conCola(root string, now time.Time, cambiar func([]Pedido) ([]Pedido, error)) error {
	ruta, err := colaPath(root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0700); err != nil {
		return err
	}
	soltar, err := bloquear(ruta + ".lock")
	if err != nil {
		return err
	}
	defer soltar()
	cola, err := cambiar(normalizar(leerCola(root), now))
	if err != nil {
		return err
	}
	return guardarCola(ruta, cola)
}

// bloquear toma un archivo de bloqueo exclusivo; uno de más de 10 s es de un proceso que murió y se descarta.
func bloquear(lock string) (func(), error) {
	for intento := 0; intento < 200; intento++ {
		f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			f.Close()
			return func() { os.Remove(lock) }, nil
		}
		if info, errStat := os.Stat(lock); errStat == nil && time.Since(info.ModTime()) > 10*time.Second {
			os.Remove(lock)
			continue
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil, fmt.Errorf("la cola de pedidos está ocupada (%s)", lock)
}

// guardarCola escribe la cola de forma atómica (archivo temporal y rename), privada para el usuario.
func guardarCola(ruta string, cola []Pedido) error {
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

// Pedir agrega un comando a la cola. Sin carpeta, corre en la raíz del proyecto. El mismo comando en la misma
// carpeta, si sigue abierto, no se duplica: se devuelve el pedido que ya estaba.
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
	var nuevo Pedido
	err = conCola(root, now, func(cola []Pedido) ([]Pedido, error) {
		usadas := map[string]bool{}
		for _, p := range cola {
			if p.Abierto() && p.Comando == comando && p.Carpeta == dir {
				nuevo = p
				return cola, nil
			}
			if p.Abierto() {
				usadas[p.Tecla] = true
			}
		}
		if len(usadas) >= MaxPedidos {
			return nil, fmt.Errorf("ya hay %d comandos esperando en el panel: que la persona los ejecute o descarte antes de pedir otro", MaxPedidos)
		}
		nuevo = Pedido{ID: siguienteID(cola), Comando: comando, Motivo: motivo, Carpeta: dir, Creado: now, Tecla: teclaLibre(usadas), Estado: EstadoPendiente}
		return append(cola, nuevo), nil
	})
	return nuevo, err
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

// Descartar quita un pedido de la cola sin ejecutarlo. Uno que se está ejecutando no se descarta.
func Descartar(root, id string) error {
	return conCola(root, time.Now(), func(cola []Pedido) ([]Pedido, error) {
		resto := make([]Pedido, 0, len(cola))
		for _, p := range cola {
			if p.ID == id && p.Estado == EstadoEjecutando {
				return nil, fmt.Errorf("el pedido %s se está ejecutando: espera a que termine", id)
			}
			if p.ID != id {
				resto = append(resto, p)
			}
		}
		if len(resto) == len(cola) {
			return nil, fmt.Errorf("no hay un pedido '%s' en la cola", id)
		}
		return resto, nil
	})
}

func accionesDePedidos(root string) []Accion {
	var acciones []Accion
	for _, p := range Pedidos(root) {
		a := Accion{
			ID: prefijoPedido + p.ID, Tipo: TipoPedido, Etiqueta: p.Motivo, Detalle: p.Comando, Carpeta: p.Carpeta,
			Estado: p.Estado, Resultado: p.Salida, Inicio: p.Inicio, Fin: p.Fin, Confirmar: true, pedido: p.ID,
		}
		if p.Ejecutable() {
			a.Tecla = p.Tecla
		}
		acciones = append(acciones, a)
	}
	return acciones
}

// TeclaDe dice con qué tecla el panel ofrece un pedido ("" si no está en la cola o ya no se puede ejecutar).
func TeclaDe(root, id string) string {
	for _, a := range accionesDePedidos(root) {
		if a.pedido == id {
			return a.Tecla
		}
	}
	return ""
}

// errNoEjecutable explica por qué un pedido no se puede ejecutar ahora.
func errNoEjecutable(id, estado string) error {
	if estado == EstadoEjecutando {
		return fmt.Errorf("el pedido %s ya se está ejecutando", id)
	}
	return fmt.Errorf("el pedido %s ya se ejecutó: no se acepta dos veces", id)
}

// ejecutarPedido lo marca como ejecutando con la cola bloqueada (así no corre dos veces), lo corre con la shell
// del sistema en su carpeta y guarda el resultado: hecho si sale con 0, fallo si no.
func ejecutarPedido(root string, a Accion, salida io.Writer) int {
	now := time.Now()
	err := conCola(root, now, func(cola []Pedido) ([]Pedido, error) {
		for i, p := range cola {
			if p.ID != a.pedido {
				continue
			}
			if !p.Ejecutable() {
				return nil, errNoEjecutable(p.ID, p.Estado)
			}
			cola[i].Estado, cola[i].Inicio, cola[i].Fin, cola[i].Salida = EstadoEjecutando, &now, nil, ""
			return cola, nil
		}
		return nil, fmt.Errorf("no hay un pedido '%s' en la cola", a.pedido)
	})
	if err != nil {
		fmt.Fprintf(salida, "[!] %v\n", err)
		return 1
	}
	shell, flag := "sh", "-c"
	if runtime.GOOS == "windows" {
		shell, flag = "cmd", "/C"
	}
	var copia bytes.Buffer
	c := exec.Command(shell, flag, a.Detalle)
	c.Dir = a.Carpeta
	c.Stdout = io.MultiWriter(salida, &copia)
	c.Stderr = io.MultiWriter(salida, &copia)
	codigo := codigoDe(c.Run(), a, io.MultiWriter(salida, &copia))
	terminar(root, a.pedido, codigo, ultimasLineas(copia.String(), lineasSalida))
	return codigo
}

func terminar(root, id string, codigo int, texto string) {
	fin := time.Now()
	conCola(root, fin, func(cola []Pedido) ([]Pedido, error) {
		for i := range cola {
			if cola[i].ID != id {
				continue
			}
			cola[i].Estado, cola[i].Fin, cola[i].Codigo, cola[i].Salida = EstadoHecho, &fin, codigo, texto
			if codigo != 0 {
				cola[i].Estado = EstadoFallo
			}
		}
		return cola, nil
	})
}

func ultimasLineas(s string, n int) string {
	lineas := strings.Split(strings.TrimSpace(s), "\n")
	if len(lineas) > n {
		lineas = lineas[len(lineas)-n:]
	}
	return strings.Join(lineas, "\n")
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
