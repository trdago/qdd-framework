package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/baul"
	"github.com/qdd-framework/qdd/pkg/enlaces"
	"github.com/qdd-framework/qdd/pkg/flujo"
	"github.com/qdd-framework/qdd/pkg/invitaciones"
)

// Rule: Prohibido usar 'else'
// RFC 0025, fase 3b (pedido de la persona, 2026-11-08): «cuando el flujo está armado debería bastar con un click en
// una esquina que diga compartir para que genere el enlace». Con Cloudflare en el baúl, compartir da un enlace; sin
// token en el entorno pasa por qdd baul usar (y su permiso), y sin Cloudflare en el baúl escribe el archivo de siempre.

type cloudflareDePrueba struct {
	subidas  map[string]string
	quitadas []string
}

func (c *cloudflareDePrueba) Preparar(context.Context) (enlaces.Sitio, error) {
	return enlaces.Sitio{Cuenta: "a1", Espacio: "ns", Base: "https://qdd-flujos.ana.workers.dev"}, nil
}

func (c *cloudflareDePrueba) Subir(_ context.Context, s enlaces.Sitio, id string, pagina []byte) (string, error) {
	c.subidas[id] = string(pagina)
	return s.Base + "/f/" + id, nil
}

func (c *cloudflareDePrueba) Quitar(_ context.Context, _ enlaces.Sitio, id string) error {
	c.quitadas = append(c.quitadas, id)
	return nil
}

func conCloudflareDePrueba(t *testing.T) *cloudflareDePrueba {
	t.Helper()
	cf := &cloudflareDePrueba{subidas: map[string]string{}}
	antes := cloudflareDeEnlace
	cloudflareDeEnlace = func(token, cuenta string) publicador { return cf }
	t.Cleanup(func() {
		cloudflareDeEnlace = antes
		flujoEnlace, flujoRol, flujoQuitarEnlace, flujoSesion = false, "", "", ""
	})
	return cf
}

func compartirJSON(t *testing.T, args ...string) resultadoCompartir {
	t.Helper()
	var salida bytes.Buffer
	flujoCompartirCmd.SetOut(&salida)
	flujoEnlace, flujoJSON = true, true
	flujoCompartirCmd.Run(flujoCompartirCmd, args)
	var r resultadoCompartir
	if err := json.Unmarshal(salida.Bytes(), &r); err != nil {
		t.Fatalf("%v: %q", err, salida.String())
	}
	return r
}

func TestCompartirEnlace_DeSoloLecturaSeSubeYSeReusa(t *testing.T) {
	root := proyectoConFlujo(t)
	t.Setenv("QDD_HOME", t.TempDir())
	t.Setenv(variableCloudflare, "tok")
	cf := conCloudflareDePrueba(t)
	r := compartirJSON(t, "pago")
	if !strings.HasPrefix(r.Enlace, "https://qdd-flujos.ana.workers.dev/f/") || strings.Contains(r.Enlace, "#") || len(cf.subidas) != 1 {
		t.Fatalf("un enlace sin fragmento: %+v", r)
	}
	otra := compartirJSON(t, "pago")
	if otra.Enlace != r.Enlace || len(cf.subidas) != 1 {
		t.Fatalf("volver a compartir actualiza el mismo enlace: %q %q", otra.Enlace, r.Enlace)
	}
	l, _ := invitaciones.Leer(root)
	if len(l.Publicados) != 1 || l.Publicados[0].Flujo != "pago" {
		t.Fatalf("queda anotado en el PC: %+v", l.Publicados)
	}
	id := l.Publicados[0].ID
	var salida bytes.Buffer
	flujoCompartirCmd.SetOut(&salida)
	flujoQuitarEnlace = id
	flujoCompartirCmd.Run(flujoCompartirCmd, nil)
	l, _ = invitaciones.Leer(root)
	if len(cf.quitadas) != 1 || cf.quitadas[0] != id || !l.Publicados[0].Quitado {
		t.Fatalf("quitar lo borra de Cloudflare y lo anota: %+v %+v", cf.quitadas, l.Publicados)
	}
}

func TestCompartirEnlace_SinTokenPasaPorElBaulConSuPermiso(t *testing.T) {
	proyectoConFlujo(t)
	t.Setenv("QDD_HOME", t.TempDir())
	t.Setenv(variableCloudflare, "")
	t.Setenv(variableHijoDelBaul, "")
	conCloudflareDePrueba(t)
	antesEntrada, antesUsar := entradaCloudflare, usarBaul
	t.Cleanup(func() { entradaCloudflare, usarBaul = antesEntrada, antesUsar })
	entradaCloudflare = func() (baul.Entrada, bool) { return baul.Entrada{Nombre: "cf-ana", Tipo: "cloudflare"}, true }
	var pedido baul.OpcionesUsar
	usarBaul = func(o baul.OpcionesUsar) (int, error) {
		pedido = o
		return baul.CodigoTodaviaNo, baul.ErrSinPermiso
	}
	flujoSesion = "sesion-panel"
	salidaCon := 0
	antesSalir := salirDeCompartir
	salirDeCompartir = func(codigo int) { salidaCon = codigo }
	t.Cleanup(func() { salirDeCompartir = antesSalir })
	r := compartirJSON(t, "pago")
	if pedido.Nombre != "cf-ana" || pedido.Fin != FinEnlace || pedido.Sesion != "sesion-panel" || len(pedido.Comando) == 0 {
		t.Fatalf("usa el token del baúl para ese fin, desde la sesión del panel: %+v", pedido)
	}
	if r.Permiso != "pendiente" || !strings.Contains(r.Mensaje, "Para ti") || salidaCon != baul.CodigoTodaviaNo {
		t.Fatalf("sin permiso dice dónde darlo y sale con 75: %+v %d", r, salidaCon)
	}
}

func TestCompartirEnlace_SinCloudflareEscribeElArchivo(t *testing.T) {
	root := proyectoConFlujo(t)
	t.Setenv("QDD_HOME", t.TempDir())
	t.Setenv(variableCloudflare, "")
	t.Setenv(variableHijoDelBaul, "")
	conCloudflareDePrueba(t)
	antes := entradaCloudflare
	t.Cleanup(func() { entradaCloudflare = antes })
	entradaCloudflare = func() (baul.Entrada, bool) { return baul.Entrada{}, false }
	r := compartirJSON(t, "pago")
	if r.Enlace != "" || !strings.HasPrefix(r.Archivo, root) || !strings.Contains(r.Mensaje, "qdd baul guardar") {
		t.Fatalf("escribe el archivo y dice cómo tener enlaces: %+v", r)
	}
	if _, err := os.Stat(r.Archivo); err != nil {
		t.Fatal(err)
	}
}

func TestCompartirEnlace_LaPaginaDeUnaInvitacionNoLlevaElToken(t *testing.T) {
	foto := flujo.ArmarFoto(flujo.Flujo{Formato: 1, ID: "pago", Nombre: "Pago", Nodos: []flujo.Nodo{{ID: "a", Tipo: "paso", Nombre: "A"}}}, func(s string) string { return s }, time.Now())
	e := flujo.Enlace{Relay: "https://relay.ejemplo.co/functions/v1/", Clave: "k", Token: "tok-secreto", Rol: flujo.RolCliente, Invitacion: "inv-1"}
	cf := &cloudflareDePrueba{subidas: map[string]string{}}
	enlace, id, err := publicarInvitacion(context.Background(), cf, enlaces.Sitio{Base: "https://x.workers.dev"}, foto, e)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(cf.subidas[id], "tok-secreto") || !strings.HasSuffix(enlace, "/f/"+id+"#t=tok-secreto") {
		t.Fatalf("a Cloudflare va sin token; el token va solo en el fragmento: %q", enlace)
	}
}
