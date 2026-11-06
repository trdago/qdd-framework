package flujo

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/qdd-framework/qdd/pkg/candado"
)

// Rule: Prohibido usar 'else'

// Las ediciones de un flujo (RFC 0012, ampliación, fases 2 y 3): qdd valida cada cambio antes de escribirlo y lo resume
// en una línea para la IA. Lo que hace la persona queda «por enviar» hasta que la IA lo recibe, y mientras tanto se puede
// deshacer. Ese registro es local: el flujo se versiona, sus pendientes no.

const (
	DePersona = "persona"
	DeIA      = "ia"

	maxComentario   = 2000
	maxLineaEdicion = 160
	maxIDGenerado   = 40
	esperaCandado   = 3 * time.Second
)

// Comentario es una conversación pegada a un nodo. DeNodo dice dónde se escribió, si ese nodo ya no está.
type Comentario struct {
	De     string `yaml:"de" json:"de"`
	Texto  string `yaml:"texto" json:"texto"`
	Cuando string `yaml:"cuando" json:"cuando"`
	DeNodo string `yaml:"de_nodo,omitempty" json:"de_nodo,omitempty"`
}

// hecho es lo que un cambio le cuenta a la IA. Entera marca una línea que no se corta: las palabras de la persona.
type hecho struct {
	linea  string
	pista  string
	entera bool
}

// Cambio modifica un flujo recién leído; si devuelve un error, no se escribe nada.
type Cambio func(f *Flujo) (hecho, error)

// Editar aplica el cambio bajo candado, para que la IA y la persona no se pisen, y solo guarda un flujo válido.
func Editar(root, id, de string, ahora time.Time, cambio Cambio) (Flujo, string, error) {
	soltar, err := tomarFlujo(root, id)
	if err != nil {
		return Flujo{}, "", err
	}
	defer soltar()
	antes, err := os.ReadFile(ruta(root, id))
	if err != nil {
		return Flujo{}, "", fmt.Errorf("no hay un flujo %q en este proyecto: 'qdd flujo lista' dice cuáles hay", id)
	}
	f, err := Leer(antes)
	if err != nil {
		return Flujo{}, "", err
	}
	h, err := cambio(&f)
	if err != nil {
		return Flujo{}, "", err
	}
	if err := Guardar(root, f); err != nil {
		return Flujo{}, "", err
	}
	if !h.entera {
		h.linea = acotar(h.linea, maxLineaEdicion)
	}
	if de != DePersona {
		return f, h.linea, nil
	}
	despues, err := os.ReadFile(ruta(root, id))
	if err != nil {
		return f, h.linea, err
	}
	anotar(root, id, entrada{Cuando: ahora.UTC().Format(time.RFC3339), Linea: h.linea, Pista: h.pista, Antes: string(antes), Huella: huella(despues)})
	return f, h.linea, nil
}

func tomarFlujo(root, id string) (func(), error) {
	if !idValido.MatchString(id) {
		return nil, fmt.Errorf("id de flujo %q: usa minúsculas, números y guiones", id)
	}
	carpeta := carpetaLocal(root)
	if err := os.MkdirAll(carpeta, 0o755); err != nil {
		return nil, err
	}
	return candado.Tomar(filepath.Join(carpeta, id+".lock"), esperaCandado, "el flujo "+id)
}

func indiceDe(f *Flujo, id string) (int, error) {
	i := slices.IndexFunc(f.Nodos, func(n Nodo) bool { return n.ID == id })
	if i < 0 {
		return 0, fmt.Errorf("el flujo no tiene un nodo %q", id)
	}
	return i, nil
}

// Agregar pone un nodo después de en: entre él y lo que seguía o, con rama, como una salida más.
func Agregar(en, nombre, tipo, descripcion string, rama bool) Cambio {
	return func(f *Flujo) (hecho, error) {
		i, err := indiceDe(f, en)
		if err != nil {
			return hecho{}, err
		}
		nombre = limpiar(nombre)
		if tipo == "" {
			tipo = TipoPaso
		}
		nuevo := Nodo{ID: idLibre(f, nombre), Tipo: tipo, Nombre: nombre, Descripcion: limpiar(descripcion)}
		donde := "después de " + en
		if rama {
			donde = "como rama de " + en
			f.Nodos[i].Siguen = append(f.Nodos[i].Siguen, nuevo.ID)
		}
		if !rama {
			nuevo.Siguen = f.Nodos[i].Siguen
			f.Nodos[i].Siguen = []string{nuevo.ID}
		}
		f.Nodos = slices.Insert(f.Nodos, i+1, nuevo)
		return hecho{linea: fmt.Sprintf("+ %s «%s» %s", nuevo.ID, nuevo.Nombre, donde)}, nil
	}
}

// Modificar cambia lo que no venga en nil, y dice qué cambió.
func Modificar(nodo string, nombre, descripcion, tipo *string) Cambio {
	return func(f *Flujo) (hecho, error) {
		i, err := indiceDe(f, nodo)
		if err != nil {
			return hecho{}, err
		}
		n := &f.Nodos[i]
		var cambios []string
		cambios = append(cambios, cambiar(&n.Nombre, nombre, "nombre")...)
		cambios = append(cambios, cambiar(&n.Descripcion, descripcion, "descripción")...)
		cambios = append(cambios, cambiar(&n.Tipo, tipo, "tipo")...)
		if len(cambios) == 0 {
			return hecho{}, fmt.Errorf("el nodo %q ya está así: no hay nada que cambiar", nodo)
		}
		return hecho{linea: "~ " + nodo + " " + strings.Join(cambios, "; ")}, nil
	}
}

func cambiar(campo, valor *string, nombre string) []string {
	if valor == nil {
		return nil
	}
	nuevo := limpiar(*valor)
	if nuevo == *campo {
		return nil
	}
	linea := fmt.Sprintf("%s: «%s» → «%s»", nombre, *campo, nuevo)
	*campo = nuevo
	return []string{linea}
}

// Quitar une lo de antes con lo de después del nodo. Sus comentarios no se pierden: pasan al primero.
func Quitar(nodo string) Cambio {
	return func(f *Flujo) (hecho, error) {
		n, err := desenganchar(f, nodo)
		if err != nil {
			return hecho{}, err
		}
		f.Nodos[0].Comentarios = append(f.Nodos[0].Comentarios, comentariosDeQuitado(n)...)
		return hecho{linea: fmt.Sprintf("− %s «%s»: se unió lo de antes con lo de después", n.ID, n.Nombre)}, nil
	}
}

// desenganchar saca el nodo de la lista y une a quienes llegaban a él con sus salidas.
func desenganchar(f *Flujo, nodo string) (Nodo, error) {
	i, err := indiceDe(f, nodo)
	if err != nil {
		return Nodo{}, err
	}
	n := f.Nodos[i]
	if i == 0 && len(n.Siguen) == 0 {
		return Nodo{}, fmt.Errorf("el nodo %q es el único camino del flujo: no se puede quitar", nodo)
	}
	if i == 0 && len(n.Siguen) > 1 {
		return Nodo{}, fmt.Errorf("el nodo %q es el inicio y tiene dos salidas o más: no se sabría cuál empieza", nodo)
	}
	f.Nodos = slices.Delete(f.Nodos, i, i+1)
	for j := range f.Nodos {
		f.Nodos[j].Siguen = sinPasarPor(f.Nodos[j], n)
	}
	if i != 0 {
		return n, nil
	}
	inicio, err := indiceDe(f, n.Siguen[0])
	if err != nil {
		return Nodo{}, err
	}
	nuevoInicio := f.Nodos[inicio]
	f.Nodos = slices.Insert(slices.Delete(f.Nodos, inicio, inicio+1), 0, nuevoInicio)
	return n, nil
}

// sinPasarPor reemplaza n por sus salidas en las de otro, sin repetir ninguna ni dejar a un nodo volviendo a sí mismo.
func sinPasarPor(otro, n Nodo) []string {
	if !contiene(otro.Siguen, n.ID) {
		return otro.Siguen
	}
	var quedan []string
	for _, s := range otro.Siguen {
		if s != n.ID {
			quedan = sumarSalida(quedan, s, otro.ID)
			continue
		}
		for _, siguiente := range n.Siguen {
			quedan = sumarSalida(quedan, siguiente, otro.ID)
		}
	}
	return quedan
}

func sumarSalida(lista []string, salida, propio string) []string {
	if salida == propio || contiene(lista, salida) {
		return lista
	}
	return append(lista, salida)
}

func comentariosDeQuitado(n Nodo) []Comentario {
	movidos := make([]Comentario, 0, len(n.Comentarios))
	for _, c := range n.Comentarios {
		if c.DeNodo == "" {
			c.DeNodo = n.Nombre
		}
		movidos = append(movidos, c)
	}
	return movidos
}

// Mover cuelga el nodo después de otro; donde estaba, se une lo de antes con lo de después.
func Mover(nodo, despuesDe string) Cambio {
	return func(f *Flujo) (hecho, error) {
		if nodo == despuesDe {
			return hecho{}, fmt.Errorf("el nodo %q no puede ir después de sí mismo", nodo)
		}
		if _, err := indiceDe(f, despuesDe); err != nil {
			return hecho{}, err
		}
		n, err := desenganchar(f, nodo)
		if err != nil {
			return hecho{}, err
		}
		i, err := indiceDe(f, despuesDe)
		if err != nil {
			return hecho{}, err
		}
		n.Siguen = f.Nodos[i].Siguen
		f.Nodos[i].Siguen = []string{n.ID}
		f.Nodos = slices.Insert(f.Nodos, i+1, n)
		return hecho{linea: fmt.Sprintf("↷ %s «%s» ahora va después de %s", n.ID, n.Nombre, despuesDe)}, nil
	}
}

// Conectar agrega una flecha entre dos nodos que ya existen: un ciclo o un encuentro.
func Conectar(desde, hacia string) Cambio {
	return func(f *Flujo) (hecho, error) {
		i, err := indiceDe(f, desde)
		if err != nil {
			return hecho{}, err
		}
		if _, err := indiceDe(f, hacia); err != nil {
			return hecho{}, err
		}
		if desde == hacia {
			return hecho{}, fmt.Errorf("el nodo %q no puede seguir en sí mismo", desde)
		}
		if contiene(f.Nodos[i].Siguen, hacia) {
			return hecho{}, fmt.Errorf("el nodo %q ya sigue en %q", desde, hacia)
		}
		f.Nodos[i].Siguen = append(f.Nodos[i].Siguen, hacia)
		return hecho{linea: fmt.Sprintf("→ %s también sigue en %s", desde, hacia)}, nil
	}
}

// Desconectar quita una flecha; si deja un nodo sin camino, Guardar la rechaza con el porqué.
func Desconectar(desde, hacia string) Cambio {
	return func(f *Flujo) (hecho, error) {
		i, err := indiceDe(f, desde)
		if err != nil {
			return hecho{}, err
		}
		if !contiene(f.Nodos[i].Siguen, hacia) {
			return hecho{}, fmt.Errorf("el nodo %q no sigue en %q", desde, hacia)
		}
		f.Nodos[i].Siguen = slices.DeleteFunc(f.Nodos[i].Siguen, func(s string) bool { return s == hacia })
		return hecho{linea: fmt.Sprintf("✂ %s ya no sigue en %s", desde, hacia)}, nil
	}
}

// Comentar suma un comentario al nodo. Le llega entero a la IA: son las palabras de la persona.
func Comentar(nodo, texto, de string, ahora time.Time) Cambio {
	return func(f *Flujo) (hecho, error) {
		i, err := indiceDe(f, nodo)
		if err != nil {
			return hecho{}, err
		}
		texto = limpiar(texto)
		if texto == "" {
			return hecho{}, fmt.Errorf("el comentario está vacío")
		}
		if largo := len([]rune(texto)); largo > maxComentario {
			return hecho{}, fmt.Errorf("el comentario tiene %d caracteres: el tope es %d", largo, maxComentario)
		}
		n := &f.Nodos[i]
		n.Comentarios = append(n.Comentarios, Comentario{De: de, Texto: texto, Cuando: ahora.UTC().Format(time.RFC3339)})
		return hecho{
			linea:  fmt.Sprintf("💬 %s «%s»: %s", n.ID, n.Nombre, texto),
			pista:  fmt.Sprintf("Responde el comentario con: qdd flujo comentar %s %s \"…\"", f.ID, n.ID),
			entera: true,
		}, nil
	}
}

// Detalle pone en el nodo la explicación que la persona pidió con «?».
func Detalle(nodo, texto string) Cambio {
	return Modificar(nodo, nil, &texto, nil)
}

// PedirDetalle deja la pregunta por enviar. No cambia el flujo: gasta tokens solo porque la persona lo pidió.
func PedirDetalle(root, id, nodo string, ahora time.Time) error {
	soltar, err := tomarFlujo(root, id)
	if err != nil {
		return err
	}
	defer soltar()
	f, err := Cargar(root, id)
	if err != nil {
		return err
	}
	i, err := indiceDe(&f, nodo)
	if err != nil {
		return err
	}
	anotar(root, id, entrada{
		Cuando: ahora.UTC().Format(time.RFC3339),
		Linea:  fmt.Sprintf("🔎 %s «%s»: la persona pide más detalle", nodo, f.Nodos[i].Nombre),
		Pista:  fmt.Sprintf("Explícalo con: qdd flujo detalle %s %s \"…\"; si es un subflujo, agrega nodos con qdd flujo agregar", id, nodo),
	})
	return nil
}

// Reemplazar guarda un flujo escrito entero (qdd flujo crear) sin perder comentarios: quien lo reescribe casi nunca los
// trae. Devuelve cuántos pasaron al primer nodo porque su nodo ya no está.
func Reemplazar(root string, f Flujo) (int, error) {
	if problemas := Validar(f); len(problemas) > 0 {
		return 0, fmt.Errorf("%s", strings.Join(problemas, "\n"))
	}
	soltar, err := tomarFlujo(root, f.ID)
	if err != nil {
		return 0, err
	}
	defer soltar()
	previo, err := Cargar(root, f.ID)
	if err != nil {
		return 0, Guardar(root, f)
	}
	movidos := conservarComentarios(previo, &f)
	return movidos, Guardar(root, f)
}

func conservarComentarios(previo Flujo, f *Flujo) int {
	var huerfanos []Comentario
	for _, antes := range previo.Nodos {
		i := slices.IndexFunc(f.Nodos, func(n Nodo) bool { return n.ID == antes.ID })
		if i < 0 {
			huerfanos = append(huerfanos, comentariosDeQuitado(antes)...)
			continue
		}
		if len(f.Nodos[i].Comentarios) == 0 {
			f.Nodos[i].Comentarios = antes.Comentarios
		}
	}
	f.Nodos[0].Comentarios = append(f.Nodos[0].Comentarios, huerfanos...)
	return len(huerfanos)
}

// Nuevo crea un flujo con un solo nodo, para seguir desde el panel. Su id sale del nombre y no pisa otro.
func Nuevo(root, nombre string) (Flujo, error) {
	nombre = limpiar(nombre)
	if nombre == "" {
		return Flujo{}, fmt.Errorf("el flujo no tiene nombre")
	}
	base := aID(nombre, "flujo")
	id := base
	for n := 2; existe(ruta(root, id)); n++ {
		id = fmt.Sprintf("%s-%d", base, n)
	}
	f := Flujo{Formato: Formato, ID: id, Nombre: nombre, Nodos: []Nodo{{ID: "inicio", Tipo: TipoResultado, Nombre: "Inicio"}}}
	return f, Guardar(root, f)
}

func existe(archivo string) bool {
	_, err := os.Stat(archivo)
	return err == nil
}

// limpiar deja el texto en una línea y sin caracteres de control, para que no rompa el dibujo ni el mensaje a la IA.
func limpiar(texto string) string {
	sinControles := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, texto)
	return strings.Join(strings.Fields(sinControles), " ")
}

func acotar(linea string, tope int) string {
	runas := []rune(linea)
	if len(runas) <= tope {
		return linea
	}
	return string(runas[:tope-1]) + "…"
}

var sinAcentos = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n", "à", "a", "è", "e", "ì", "i", "ò", "o", "ù", "u", "ç", "c")

// aID lleva un nombre a la forma de un id; si no queda nada, usa porDefecto.
func aID(nombre, porDefecto string) string {
	var b strings.Builder
	for _, r := range sinAcentos.Replace(strings.ToLower(nombre)) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			continue
		}
		if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
			b.WriteByte('-')
		}
	}
	id := strings.TrimRight(acotarBytes(b.String(), maxIDGenerado), "-")
	if id == "" {
		return porDefecto
	}
	return id
}

func acotarBytes(s string, tope int) string {
	if len(s) <= tope {
		return s
	}
	return s[:tope]
}

func idLibre(f *Flujo, nombre string) string {
	base := aID(nombre, "nodo")
	id := base
	for n := 2; slices.ContainsFunc(f.Nodos, func(x Nodo) bool { return x.ID == id }); n++ {
		id = fmt.Sprintf("%s-%d", base, n)
	}
	return id
}
