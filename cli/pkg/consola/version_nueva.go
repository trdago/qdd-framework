package consola

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'
// Una versión nueva de qdd, a una tecla (pedido del usuario, 2026-11-05: como la app, que avisa y actualiza con un clic).
// qdd pregunta a npm cada IntervaloVersion, aparte y sin demorar al panel (VersionAlDia), y deja un pedido opcional
// «⬆ Actualizar qdd a X» del canal instalado: una versión estable sigue a latest; un rc, a preview y a latest. Corre solo
// si la persona lo confirma con «s  Sí». Como un PR, no abre el panel, no es una espera de la persona, cede su lugar a un
// pedido de la IA, no lo lanza el modo goal y uno descartado no vuelve hasta que salga otra versión.
//
// En Windows, una instalación que todavía deja qdd.exe dentro del paquete no tiene botón: npm no puede reemplazarlo
// mientras algo lo usa, y un pedido corre desde qdd.exe (issue #107). El panel avisa con lo que hay que hacer. Desde que
// el binario vive en su carpeta de versión (binario_version.go), Windows tiene el mismo botón.

const (
	// IntervaloVersion es cada cuánto se pregunta a npm.
	IntervaloVersion = time.Hour
	// topeRegistro es cuánto puede tardar la respuesta de npm.
	topeRegistro = 8 * time.Second
	urlDistTags  = "https://registry.npmjs.org/-/package/qdd-framework/dist-tags"
	// maxRespuestaRegistro acota lo que se lee de npm: dist-tags son unas pocas líneas.
	maxRespuestaRegistro = 64 * 1024
)

// reVersion es la única forma que se acepta: lo que dice el registro termina dentro de un comando.
var reVersion = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-rc\.(\d+))?$`)

// numerosDe da mayor, menor, parche y rc; una versión estable va después de todos sus rc.
func numerosDe(v string) ([4]int, bool) {
	m := reVersion.FindStringSubmatch(strings.TrimPrefix(v, "v"))
	if m == nil {
		return [4]int{}, false
	}
	out := [4]int{}
	for i := 0; i < 3; i++ {
		out[i], _ = strconv.Atoi(m[i+1])
	}
	out[3] = int(^uint(0) >> 1)
	if m[4] != "" {
		out[3], _ = strconv.Atoi(m[4])
	}
	return out, true
}

func esMayor(a, b string) bool {
	na, okA := numerosDe(a)
	nb, okB := numerosDe(b)
	if !okA || !okB {
		return false
	}
	for i := range na {
		if na[i] != nb[i] {
			return na[i] > nb[i]
		}
	}
	return false
}

// VersionAOfrecer es la versión del canal instalado más nueva que la instalada; vacía si no hay.
func VersionAOfrecer(instalada string, tags map[string]string) string {
	if _, ok := numerosDe(instalada); !ok {
		return ""
	}
	mejor := ""
	canales := []string{"latest"}
	if strings.Contains(instalada, "-") {
		canales = append(canales, "preview")
	}
	for _, canal := range canales {
		v := tags[canal]
		if _, ok := numerosDe(v); !ok || strings.HasPrefix(v, "v") || !esMayor(v, instalada) {
			continue
		}
		if mejor == "" || esMayor(v, mejor) {
			mejor = v
		}
	}
	return mejor
}

// InstaladoConNPM: solo una instalación de npm se actualiza con npm; un binario compilado a mano no se toca.
func InstaladoConNPM(bin string) bool {
	return BinarioPorVersion(bin) || strings.Contains(strings.ReplaceAll(bin, `\`, "/"), "/node_modules/qdd-framework/")
}

// ComandoActualizarQDD es el comando del pedido. dist-tags nombra una versión antes de que npm la entregue (FND-194):
// primero se le pregunta a npm, y si no la ve, el pedido dice «todavía no» (75) y se reintenta solo. --prefer-online
// evita que la caché de npm responda por el registro. El paréntesis y «exit 75» valen igual en sh y en cmd.
func ComandoActualizarQDD(v string) string {
	return "(npm view qdd-framework@" + v + " version --prefer-online || exit 75) && npm i -g qdd-framework@" + v + " --prefer-online --allow-scripts=qdd-framework && echo Listo. Escribe /reload-plugins para cargar el panel nuevo"
}

// OfrecerVersion deja el pedido de la versión v y quita el pendiente de otra. Con v vacía, solo lo quita. Sin lugar en la
// cola, o si la persona ya descartó esa versión, no se ofrece.
func OfrecerVersion(root, v string, now time.Time) error {
	descartada := leerVersionDescartada(root)
	return conCola(root, now, func(cola []Pedido) ([]Pedido, error) {
		var queda []Pedido
		for _, p := range cola {
			if p.Version != "" && p.Version != v && p.Estado == EstadoPendiente {
				continue
			}
			queda = append(queda, p)
		}
		if v == "" || v == descartada || yaOfrecida(queda, v) {
			return queda, nil
		}
		usadas := teclasAbiertas(queda)
		if len(usadas) >= MaxPedidos {
			return queda, nil
		}
		return append(queda, Pedido{
			ID: siguienteID(queda), Comando: ComandoActualizarQDD(v), Motivo: "⬆ Actualizar qdd a " + v, Carpeta: root, Creado: now,
			Tecla: teclaLibre(usadas), Estado: EstadoPendiente, Version: v,
		}), nil
	})
}

func yaOfrecida(cola []Pedido, v string) bool {
	for _, p := range cola {
		if p.Version == v {
			return true
		}
	}
	return false
}

func rutaVersionDescartada(root string) (string, error) {
	return RutaLocal(root, "version-descartada")
}

func leerVersionDescartada(root string) string {
	ruta, err := rutaVersionDescartada(root)
	if err != nil {
		return ""
	}
	b, err := os.ReadFile(ruta)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func anotarVersionDescartada(root, v string) {
	ruta, err := rutaVersionDescartada(root)
	if err != nil || os.MkdirAll(filepath.Dir(ruta), 0700) != nil {
		return
	}
	_ = os.WriteFile(ruta, []byte(v), 0600)
}

// ── El aviso de Windows ───────────────────────────────────────────────────────────────────────────────────────────

func rutaVersionAvisada(root string) (string, error) {
	return RutaLocal(root, "version-avisada")
}

func guardarVersionAvisada(root, v string) error {
	ruta, err := rutaVersionAvisada(root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0700); err != nil {
		return err
	}
	return os.WriteFile(ruta, []byte(v), 0600)
}

// AvisoDeVersion es lo que el panel muestra cuando no puede ofrecer el botón; vacío si no hay nada que decir.
func AvisoDeVersion(root, instalada string) string {
	ruta, err := rutaVersionAvisada(root)
	if err != nil {
		return ""
	}
	b, err := os.ReadFile(ruta)
	if err != nil {
		return ""
	}
	v := strings.TrimSpace(string(b))
	if !esMayor(v, instalada) {
		return ""
	}
	return "⬆ qdd " + v + " disponible: cierra las sesiones de Claude Code y corre npm i -g qdd-framework@" + v + " --allow-scripts=qdd-framework"
}

// ── Preguntar: aparte, cada IntervaloVersion ──────────────────────────────────────────────────────────────────────

// VersionAlDia lanza aparte «bin consola version-nueva --raiz root» si hace más de IntervaloVersion que no se pregunta o
// si la versión instalada cambió; vuelve en el acto. Un binario que no vino de npm no pregunta nada.
func VersionAlDia(root, bin, instalada string, now time.Time) {
	if !InstaladoConNPM(bin) {
		return
	}
	marca, err := RutaLocal(root, "version-revisada")
	if err != nil {
		return
	}
	if info, err := os.Stat(marca); err == nil && now.Sub(info.ModTime()) < IntervaloVersion && marcaDe(marca) == instalada {
		return
	}
	if os.MkdirAll(filepath.Dir(marca), 0700) != nil || os.WriteFile(marca, []byte(instalada), 0600) != nil {
		return
	}
	c := exec.Command(bin, "consola", "version-nueva", "--raiz", root)
	c.Dir = root
	separar(c)
	if c, err := arrancar(c); err == nil {
		soltar(c)
	}
}

func marcaDe(ruta string) string {
	b, err := os.ReadFile(ruta)
	if err != nil {
		return ""
	}
	return string(b)
}

// distTagsReal pregunta a npm las etiquetas del paquete.
func distTagsReal() (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), topeRegistro)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlDistTags, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("npm respondió %d", res.StatusCode)
	}
	var tags map[string]string
	if err := json.NewDecoder(io.LimitReader(res.Body, maxRespuestaRegistro)).Decode(&tags); err != nil {
		return nil, fmt.Errorf("npm respondió algo ilegible: %v", err)
	}
	return tags, nil
}

// TraerVersion pregunta a npm y ofrece lo que corresponda en el panel.
func TraerVersion(root, bin, instalada string, now time.Time) error {
	return traerVersionCon(distTagsReal, runtime.GOOS, bin, root, instalada, now)
}

func traerVersionCon(obtener func() (map[string]string, error), sistema, bin, root, instalada string, now time.Time) error {
	tags, err := obtener()
	if err != nil {
		return err
	}
	v := VersionAOfrecer(instalada, tags)
	if sistema == "windows" && !BinarioPorVersion(bin) {
		return guardarVersionAvisada(root, v)
	}
	return OfrecerVersion(root, v, now)
}
