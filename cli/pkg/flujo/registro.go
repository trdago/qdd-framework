package flujo

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Rule: Prohibido usar 'else'

const (
	// CarpetaLocal guarda lo que la IA todavía no recibe y lo que se puede deshacer. No se versiona.
	CarpetaLocal = ".qdd/working/flujos"

	// maxRegistro acota el archivo: cada entrada guarda el flujo anterior, que pesa hasta MaxBytes.
	maxRegistro = 50
)

// entrada es una edición de la persona. Antes y Huella permiten deshacerla sin pisar lo que otro cambió después.
type entrada struct {
	Cuando  string `json:"cuando"`
	Linea   string `json:"linea"`
	Pista   string `json:"pista,omitempty"`
	Antes   string `json:"antes,omitempty"`
	Huella  string `json:"huella,omitempty"`
	Enviada bool   `json:"enviada,omitempty"`
}

func carpetaLocal(root string) string {
	return filepath.Join(root, CarpetaLocal)
}

func rutaRegistro(root, id string) string {
	return filepath.Join(carpetaLocal(root), id+".jsonl")
}

// leerRegistro salta una línea rota: perder una nota local no puede impedir editar el flujo.
func leerRegistro(root, id string) []entrada {
	datos, err := os.ReadFile(rutaRegistro(root, id))
	if err != nil {
		return nil
	}
	var lista []entrada
	lector := bufio.NewScanner(bytes.NewReader(datos))
	lector.Buffer(nil, 2*MaxBytes+1024)
	for lector.Scan() {
		var e entrada
		if json.Unmarshal(lector.Bytes(), &e) != nil {
			continue
		}
		lista = append(lista, e)
	}
	return lista
}

func escribirRegistro(root, id string, lista []entrada) error {
	var b bytes.Buffer
	for _, e := range lista {
		linea, err := json.Marshal(e)
		if err != nil {
			return err
		}
		b.Write(linea)
		b.WriteByte('\n')
	}
	return escribirAtomico(rutaRegistro(root, id), b.Bytes())
}

// anotar suma la edición y olvida lo ya enviado, que no se puede deshacer.
func anotar(root, id string, e entrada) {
	lista := []entrada{}
	for _, previa := range leerRegistro(root, id) {
		if !previa.Enviada {
			lista = append(lista, previa)
		}
	}
	lista = append(lista, e)
	if len(lista) > maxRegistro {
		lista = lista[len(lista)-maxRegistro:]
	}
	_ = escribirRegistro(root, id, lista)
}

// PorEnviar son las líneas que la IA todavía no recibe.
func PorEnviar(root, id string) []string {
	lineas := []string{}
	if !idValido.MatchString(id) {
		return lineas
	}
	for _, e := range leerRegistro(root, id) {
		if !e.Enviada {
			lineas = append(lineas, e.Linea)
		}
	}
	return lineas
}

// PuedeDeshacer dice si la última edición todavía no le llegó a la IA.
func PuedeDeshacer(root, id string) bool {
	if !idValido.MatchString(id) {
		return false
	}
	lista := leerRegistro(root, id)
	return len(lista) > 0 && !lista[len(lista)-1].Enviada
}

// Enviar reclama lo pendiente y arma el mensaje para la IA: dos sesiones no reciben lo mismo. Devuelve cuántas líneas
// lleva; si no se pudo entregar, Devolver las deja pendientes otra vez.
func Enviar(root, id string) (string, int) {
	soltar, err := tomarFlujo(root, id)
	if err != nil {
		return "", 0
	}
	defer soltar()
	lista := leerRegistro(root, id)
	var lineas, pistas []string
	for i, e := range lista {
		if e.Enviada {
			continue
		}
		lineas = append(lineas, e.Linea)
		if e.Pista != "" && !contiene(pistas, e.Pista) {
			pistas = append(pistas, e.Pista)
		}
		lista[i].Enviada = true
	}
	if len(lineas) == 0 {
		return "", 0
	}
	if escribirRegistro(root, id, lista) != nil {
		return "", 0
	}
	nombre := id
	if f, err := Cargar(root, id); err == nil {
		nombre = f.Nombre
	}
	partes := []string{fmt.Sprintf("✎ La persona trabajó en el flujo «%s» desde el panel:", nombre)}
	partes = append(partes, lineas...)
	partes = append(partes, fmt.Sprintf("Lo ves con: qdd flujo ver %s", id))
	partes = append(partes, pistas...)
	return strings.Join(partes, "\n"), len(lineas)
}

// Devolver deja como pendientes las últimas n líneas enviadas.
func Devolver(root, id string, n int) error {
	soltar, err := tomarFlujo(root, id)
	if err != nil {
		return err
	}
	defer soltar()
	lista := leerRegistro(root, id)
	for i := len(lista) - 1; i >= 0 && n > 0; i-- {
		if !lista[i].Enviada {
			continue
		}
		lista[i].Enviada = false
		n--
	}
	return escribirRegistro(root, id, lista)
}

// EnviarTodos reclama lo pendiente de todos los flujos, para que viaje con el próximo mensaje de la persona. Sin nada
// pendiente solo cuesta listar una carpeta.
func EnviarTodos(root string) string {
	archivos, err := filepath.Glob(filepath.Join(carpetaLocal(root), "*.jsonl"))
	if err != nil {
		return ""
	}
	sort.Strings(archivos)
	var mensajes []string
	for _, a := range archivos {
		mensaje, n := Enviar(root, strings.TrimSuffix(filepath.Base(a), ".jsonl"))
		if n > 0 {
			mensajes = append(mensajes, mensaje)
		}
	}
	return strings.Join(mensajes, "\n")
}

// Deshacer revierte la última edición mientras la IA no la recibe y nadie cambió el flujo después.
func Deshacer(root, id string) error {
	soltar, err := tomarFlujo(root, id)
	if err != nil {
		return err
	}
	defer soltar()
	lista := leerRegistro(root, id)
	if len(lista) == 0 {
		return errors.New("no hay nada que deshacer en este flujo")
	}
	ultima := lista[len(lista)-1]
	if ultima.Enviada {
		return errors.New("esa edición ya la recibió la IA: hay que cambiarla con otra edición")
	}
	if err := restaurar(root, id, ultima); err != nil {
		return err
	}
	return escribirRegistro(root, id, lista[:len(lista)-1])
}

func restaurar(root, id string, e entrada) error {
	if e.Antes == "" {
		return nil
	}
	actual, err := os.ReadFile(ruta(root, id))
	if err != nil {
		return err
	}
	if huella(actual) != e.Huella {
		return errors.New("el flujo cambió después de esa edición: deshacerla pisaría ese cambio")
	}
	return escribirAtomico(ruta(root, id), []byte(e.Antes))
}
