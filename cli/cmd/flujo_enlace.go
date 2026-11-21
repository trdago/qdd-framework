package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/qdd-framework/qdd/pkg/baul"
	"github.com/qdd-framework/qdd/pkg/enlaces"
	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/flujo"
	"github.com/qdd-framework/qdd/pkg/invitaciones"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// Rule: Prohibido usar 'else'
// qdd flujo compartir --enlace (RFC 0025, fase 3b; propuesta de la persona, 2026-11-08): con un token de Cloudflare en
// el baúl, la página del flujo se publica en un Worker de la cuenta de la persona y quien la recibe abre un enlace. Sin
// el token en el entorno, qdd se vuelve a correr con 'qdd baul usar', que pide el permiso de la persona en el panel; sin
// Cloudflare en el baúl, escribe el archivo de siempre. El token de una invitación nunca sube a Cloudflare: va en el
// fragmento del enlace.

const (
	// FinEnlace es el fin del permiso del baúl: uno por sesión y proyecto, como cualquier otro uso.
	FinEnlace = "compartir flujos por enlace"

	variableCloudflare = "CLOUDFLARE_API_TOKEN"
	// variableHijoDelBaul marca la vuelta que corre con el token: si aun así no está, no se vuelve a pedir.
	variableHijoDelBaul = "QDD_COMPARTIR_POR_BAUL"
)

var (
	flujoEnlace       bool
	flujoQuitarEnlace string
	flujoCuenta       string
	flujoSesion       string
)

// publicador es lo que compartir necesita de Cloudflare.
type publicador interface {
	Preparar(ctx context.Context) (enlaces.Sitio, error)
	Subir(ctx context.Context, s enlaces.Sitio, id string, pagina []byte) (string, error)
	Quitar(ctx context.Context, s enlaces.Sitio, id string) error
}

var (
	cloudflareDeEnlace = func(token, cuenta string) publicador {
		return enlaces.Cuenta{Token: token, CuentaID: cuenta}
	}
	entradaCloudflare = cloudflareDelBaul
	usarBaul          = baul.Usar
	salirDeCompartir  = os.Exit
)

// resultadoCompartir es lo que lee el panel: el enlace (o el archivo, sin Cloudflare) y qué falta.
type resultadoCompartir struct {
	Enlace     string `json:"enlace,omitempty"`
	Archivo    string `json:"archivo,omitempty"`
	Invitacion string `json:"invitacion,omitempty"`
	Vence      string `json:"vence,omitempty"`
	Permiso    string `json:"permiso,omitempty"`
	Mensaje    string `json:"mensaje"`
}

func informar(cmd *cobra.Command, r resultadoCompartir) {
	if flujoJSON {
		b, _ := json.Marshal(r)
		fmt.Fprintln(cmd.OutOrStdout(), string(b))
		return
	}
	if r.Enlace != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "[+] %s\n", r.Enlace)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "    %s\n", r.Mensaje)
}

func compartirPorEnlace(cmd *cobra.Command, root string, f flujo.Flujo, foto flujo.Foto, ahora time.Time) {
	token := os.Getenv(variableCloudflare)
	if token == "" {
		compartirConElBaul(cmd, root, f, foto, ahora)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cf := cloudflareDeEnlace(token, flujoCuenta)
	sitio, err := cf.Preparar(ctx)
	if err != nil {
		salirCon("%v", err)
	}
	if flujoRol == "" {
		informar(cmd, publicarSoloLectura(ctx, cf, sitio, root, f, foto, ahora))
		return
	}
	inv, e := crearInvitacion(root, f, foto, ahora)
	enlace, id, err := publicarInvitacion(ctx, cf, sitio, foto, e)
	if err != nil {
		salirCon("la invitación %s quedó creada, pero su página no se subió: %v (ciérrala con qdd flujo compartir --revocar %s)", inv.ID, err, inv.ID)
	}
	anotarPublicado(root, invitaciones.Publicado{ID: id, Flujo: f.ID, Invitacion: inv.ID, URL: sitio.Base + "/f/" + id, Cuenta: sitio.Cuenta, Espacio: sitio.Espacio, Creado: ahora})
	informar(cmd, resultadoCompartir{Enlace: enlace, Invitacion: inv.ID, Vence: inv.Vence.Format(time.RFC3339), Mensaje: fmt.Sprintf(
		"Enlace para %s (%s, vence el %s, hasta %d propuestas): quien lo abra propone y cada propuesta llega a tu panel. "+
			"Lleva la llave de la invitación después de #: mándaselo solo a esa persona. Para cerrarlo: qdd flujo compartir --revocar %s",
		paraQuien(), inv.Rol, inv.Vence.Local().Format("02/01 15:04"), flujoTope, inv.ID)})
}

// publicarSoloLectura sube la página de solo lectura; el flujo conserva su enlace, así que volver a compartir lo
// actualiza en vez de crear otro.
func publicarSoloLectura(ctx context.Context, cf publicador, sitio enlaces.Sitio, root string, f flujo.Flujo, foto flujo.Foto, ahora time.Time) resultadoCompartir {
	id, err := enlaces.NuevoID()
	if err != nil {
		salirCon("%v", err)
	}
	l, _ := invitaciones.Leer(root)
	if previo, ok := l.SoloLectura(f.ID); ok && previo.Cuenta == sitio.Cuenta {
		id = previo.ID
	}
	enlace, err := cf.Subir(ctx, sitio, id, mustArchivo(flujo.ArchivoHTML(foto)))
	if err != nil {
		salirCon("%v", err)
	}
	anotarPublicado(root, invitaciones.Publicado{ID: id, Flujo: f.ID, URL: enlace, Cuenta: sitio.Cuenta, Espacio: sitio.Espacio, Creado: ahora})
	return resultadoCompartir{Enlace: enlace, Mensaje: fmt.Sprintf("Enlace de solo lectura (r%d): compartir de nuevo lo actualiza. Para quitarlo: qdd flujo compartir --quitar-enlace %s", f.Revision, id)}
}

// publicarInvitacion sube la página sin el token y devuelve el enlace con el token en el fragmento.
func publicarInvitacion(ctx context.Context, cf publicador, sitio enlaces.Sitio, foto flujo.Foto, e flujo.Enlace) (string, string, error) {
	pagina, err := flujo.PaginaParaEnlace(foto, e)
	if err != nil {
		return "", "", err
	}
	id, err := enlaces.NuevoID()
	if err != nil {
		return "", "", err
	}
	enlace, err := cf.Subir(ctx, sitio, id, pagina)
	if err != nil {
		return "", "", err
	}
	return enlace + flujo.FragmentoDeEnlace(e), id, nil
}

func anotarPublicado(root string, p invitaciones.Publicado) {
	if err := invitaciones.AnotarPublicado(root, p); err != nil {
		salirCon("el enlace %s quedó publicado, pero no se anotó en este PC: %v", p.URL, err)
	}
}

// compartirConElBaul vuelve a correr este mismo comando con el token de Cloudflare del baúl. Sin permiso, el baúl deja
// el pedido 🔑 en el panel y esto sale con 75; sin Cloudflare en el baúl, escribe el archivo.
func compartirConElBaul(cmd *cobra.Command, root string, f flujo.Flujo, foto flujo.Foto, ahora time.Time) {
	if os.Getenv(variableHijoDelBaul) != "" {
		salirCon("el baúl corrió compartir sin %s: revisa la entrada de Cloudflare con qdd baul verificar", variableCloudflare)
	}
	entrada, ok := entradaCloudflare()
	if !ok {
		informar(cmd, compartirComoArchivo(cmd, root, f, foto, ahora))
		return
	}
	o := baul.OpcionesUsar{Root: root, Nombre: entrada.Nombre, Fin: FinEnlace, Sesion: sesionDeCompartir(), Comando: estaMismaVuelta(),
		Entrada: os.Stdin, Salida: cmd.OutOrStdout(), Ahora: time.Now()}
	if o.Sesion == "" && term.IsTerminal(int(os.Stdin.Fd())) {
		o.Terminal = &baul.Terminal{PID: os.Getppid(), Preguntar: preguntarEnTerminal}
	}
	os.Setenv(variableHijoDelBaul, "1")
	codigo, err := usarBaul(o)
	if errors.Is(err, baul.ErrSinPermiso) {
		informar(cmd, resultadoCompartir{Permiso: "pendiente", Mensaje: fmt.Sprintf(
			"Falta tu permiso para usar %s: confírmalo en «Para ti» (🔑) y vuelve a compartir. Dura lo que esta consola.", entrada.Nombre)})
		salirDeCompartir(codigo)
		return
	}
	if err != nil {
		salirCon("%v", err)
	}
	if codigo != 0 {
		salirDeCompartir(codigo)
	}
}

// compartirComoArchivo es el camino sin Cloudflare: el archivo de siempre, con o sin invitación.
func compartirComoArchivo(cmd *cobra.Command, root string, f flujo.Flujo, foto flujo.Foto, ahora time.Time) resultadoCompartir {
	destino := archivoDeCompartir(root, f.ID)
	html := mustArchivo(flujo.ArchivoHTML(foto))
	r := resultadoCompartir{Archivo: destino}
	if flujoRol != "" {
		inv, conInvitacion := invitar(root, f, foto, ahora)
		html, r.Invitacion, r.Vence = conInvitacion, inv.ID, inv.Vence.Format(time.RFC3339)
	}
	escribirEn(destino, html)
	r.Mensaje = "Sin Cloudflare en el baúl, quedó el archivo: mándalo por correo o chat. Para compartir con un enlace, " +
		"guarda un token de Cloudflare (Workers Scripts y Workers KV): qdd baul guardar cloudflare --tipo cloudflare"
	return r
}

func escribirEn(destino string, html []byte) {
	if err := os.MkdirAll(filepath.Dir(destino), 0o755); err != nil {
		salirCon("%v", err)
	}
	if err := os.WriteFile(destino, html, 0o600); err != nil {
		salirCon("%v", err)
	}
}

// quitarEnlace borra la página de Cloudflare y la anota quitada. Sin el token, pasa por el baúl como compartir.
func quitarEnlace(cmd *cobra.Command, root, id string) {
	l, err := invitaciones.Leer(root)
	if err != nil {
		salirCon("%v", err)
	}
	p, ok := l.Publicado(id)
	if !ok {
		salirCon("este proyecto no tiene el enlace %s vigente: 'qdd flujo invitaciones' dice cuáles hay", id)
	}
	token := os.Getenv(variableCloudflare)
	if token == "" {
		quitarConElBaul(cmd, root, id)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := cloudflareDeEnlace(token, p.Cuenta).Quitar(ctx, enlaces.Sitio{Cuenta: p.Cuenta, Espacio: p.Espacio}, id); err != nil {
		salirCon("%v", err)
	}
	if err := invitaciones.MarcarQuitado(root, id); err != nil {
		salirCon("%v", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "[+] Enlace %s quitado: ya no abre.\n", p.URL)
}

func quitarConElBaul(cmd *cobra.Command, root, id string) {
	entrada, ok := entradaCloudflare()
	if !ok || os.Getenv(variableHijoDelBaul) != "" {
		salirCon("para quitar el enlace hace falta el token de Cloudflare del baúl: qdd baul usar <cloudflare> --para %q -- qdd flujo compartir --quitar-enlace %s", FinEnlace, id)
	}
	os.Setenv(variableHijoDelBaul, "1")
	codigo, err := usarBaul(baul.OpcionesUsar{Root: root, Nombre: entrada.Nombre, Fin: FinEnlace, Sesion: sesionDeCompartir(),
		Comando: estaMismaVuelta(), Entrada: os.Stdin, Salida: cmd.OutOrStdout(), Ahora: time.Now()})
	if errors.Is(err, baul.ErrSinPermiso) {
		fmt.Fprintf(cmd.OutOrStdout(), "[⏳] Falta tu permiso para usar %s: confírmalo en «Para ti» (🔑) y vuelve a correrlo.\n", entrada.Nombre)
		salirDeCompartir(codigo)
		return
	}
	if err != nil {
		salirCon("%v", err)
	}
	if codigo != 0 {
		salirDeCompartir(codigo)
	}
}

// quitarEnlaceDeInvitacion: revocar una invitación también borra su enlace, si tenía.
func quitarEnlaceDeInvitacion(cmd *cobra.Command, root, invitacion string) {
	l, err := invitaciones.Leer(root)
	if err != nil {
		return
	}
	p, ok := l.DeInvitacion(invitacion)
	if !ok {
		return
	}
	quitarEnlace(cmd, root, p.ID)
}

// cloudflareDelBaul es la primera entrada de Cloudflare del baúl que no fue rechazada.
func cloudflareDelBaul() (baul.Entrada, bool) {
	indice, err := baul.Indice()
	if err != nil {
		return baul.Entrada{}, false
	}
	for _, e := range indice {
		if e.Tipo == "cloudflare" && !e.Rechazado {
			return e, true
		}
	}
	return baul.Entrada{}, false
}

func sesionDeCompartir() string {
	if flujoSesion != "" {
		return flujoSesion
	}
	return estado.SesionActual()
}

// estaMismaVuelta es este mismo comando, para correrlo de nuevo con el token en el entorno.
func estaMismaVuelta() []string {
	ejecutable, err := os.Executable()
	if err != nil {
		ejecutable = os.Args[0]
	}
	return append([]string{ejecutable}, os.Args[1:]...)
}
