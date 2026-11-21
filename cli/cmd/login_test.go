package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/remoto"
)

// Rule: Prohibido usar 'else'
// Issue #102: emparejar con QDD Móvil es el login. qdd login funciona sin argumentos cuando qdd trae un relay por
// defecto; --relay y --clave quedan para quien tenga el suyo. Las pruebas no instalan nada.

func prepararLogin(t *testing.T, porDefecto remoto.ConfigRelay) *bytes.Buffer {
	t.Helper()
	t.Setenv("QDD_HOME", t.TempDir())
	usuario := t.TempDir()
	antesEjecutor, antesUsuario, antesConsultar := remotoEjecutor, remotoUsuario, remotoConsultarRelay
	remotoEjecutor = func(string, ...string) error { return nil }
	remotoUsuario = func() (string, error) { return usuario, nil }
	remotoConsultarRelay = func(context.Context, int) (remoto.ConfigRelay, error) {
		if porDefecto.URL == "" {
			return porDefecto, errors.New("no se pudo leer el directorio del relay")
		}
		return porDefecto, nil
	}
	remotoCadaEspera, remotoRelayURL, remotoClave, remotoSinServ = 5*time.Millisecond, "", "", false
	t.Cleanup(func() {
		remotoEjecutor, remotoUsuario, remotoConsultarRelay = antesEjecutor, antesUsuario, antesConsultar
		remotoCadaEspera, remotoRelayURL, remotoClave, remotoSinServ = 0, "", "", false
	})
	return &bytes.Buffer{}
}

func TestLogin_EsUnComandoYNoUnaIntencion(t *testing.T) {
	c, _, err := rootCmd.Find([]string{"login"})
	if err != nil || c.Name() != "login" {
		t.Fatalf("qdd login es un comando, no texto libre: %v %v", c.Name(), err)
	}
	if c.PersistentPreRunE == nil || c.PersistentPostRun == nil {
		t.Fatal("como qdd remoto, no exige proyecto ni pasa por el gatekeeper")
	}
	for _, f := range []string{"relay", "clave", "sin-servicio"} {
		if c.Flags().Lookup(f) == nil {
			t.Fatalf("le falta --%s, para quien tenga su propio relay", f)
		}
	}
}

func TestLogin_SinRelayPorDefectoDiceQueFalta(t *testing.T) {
	out := prepararLogin(t, remoto.ConfigRelay{})
	var errs bytes.Buffer
	if c := remotoActivar(context.Background(), out, &errs); c != 1 || !strings.Contains(errs.String(), "qdd login --relay") {
		t.Fatalf("dice cómo seguir: %d %s", c, errs.String())
	}
	if _, err := remoto.LeerRelay(); err == nil {
		t.Fatal("no guarda nada")
	}
}

func TestLogin_ConRelayPorDefectoEmparejaSinArgumentos(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("el servicio de prueba se genera para Linux y macOS")
	}
	f, url := nuevoRelayCmd(t)
	out := prepararLogin(t, remoto.ConfigRelay{URL: url, Clave: "sb_publishable_prueba"})
	var errs bytes.Buffer
	if c := remotoActivar(context.Background(), out, &errs); c != 0 {
		t.Fatalf("login: %d %s", c, errs.String())
	}
	if !strings.Contains(out.String(), "Escanea") {
		t.Fatalf("muestra el QR: %s", out.String())
	}
	cred, err := remoto.LeerCredenciales()
	if err != nil || cred.Topico != f.topico {
		t.Fatalf("queda emparejado: %+v %v", cred, err)
	}
	guardado, err := remoto.LeerRelay()
	if err != nil || guardado.URL != url {
		t.Fatalf("queda guardado: el servicio y qdd remoto dispositivos lo leen de ahí: %+v %v", guardado, err)
	}
}

func TestLogin_LasBanderasGananAlPorDefecto(t *testing.T) {
	prepararLogin(t, remoto.ConfigRelay{URL: "https://por-defecto.supabase.co", Clave: "sb_publishable_defecto"})
	remotoRelayURL, remotoClave = "https://propio.supabase.co", "sb_publishable_propio"
	c, err := relayDeFlags(io.Discard)
	if err != nil || c.URL != "https://propio.supabase.co" {
		t.Fatalf("--relay y --clave mandan: %+v %v", c, err)
	}
}

// FND-133: un PC sin relay guardado toma el que trae qdd, sin banderas, y lo deja guardado para el servicio.
func TestLogin_UnPCNuevoUsaElRelayQueTraeQdd_FND133(t *testing.T) {
	prepararLogin(t, remoto.ConfigRelay{URL: "https://del-directorio.supabase.co", Clave: "sb_publishable_prueba", Serie: 1})
	c, err := relayDeFlags(io.Discard)
	if err != nil {
		t.Fatalf("un PC nuevo no necesita banderas: %v", err)
	}
	guardado, err := remoto.LeerRelay()
	if err != nil || guardado != c {
		t.Fatalf("queda guardado: %v", err)
	}
}

func TestLogin_LasBanderasDejanUnRelayPropio(t *testing.T) {
	prepararLogin(t, remoto.ConfigRelay{URL: "https://del-directorio.supabase.co", Clave: "sb_publishable_prueba", Serie: 1})
	remotoRelayURL, remotoClave = "https://mio.supabase.co", "sb_publishable_mio"
	c, err := relayDeFlags(io.Discard)
	if err != nil || !c.Propio || c.URL != "https://mio.supabase.co" {
		t.Fatalf("un relay con banderas es propio: %+v %v", c, err)
	}
	emparejarPrueba(t)
	remotoRelayURL, remotoClave = "", ""
	c, err = relayDeFlags(io.Discard)
	if err != nil || c.URL != "https://mio.supabase.co" {
		t.Fatalf("con este PC emparejado, el directorio no lo pisa: %+v %v", c, err)
	}
}

func TestLogin_ConsultaDeNuevoYSiFallaSigueConElGuardado(t *testing.T) {
	prepararLogin(t, remoto.ConfigRelay{URL: "https://nuevo.supabase.co", Clave: "sb_publishable_n", Serie: 2})
	if err := remoto.GuardarRelay(remoto.ConfigRelay{URL: "https://viejo.supabase.co", Clave: "sb_publishable_v", Serie: 1}); err != nil {
		t.Fatal(err)
	}
	c, err := relayDeFlags(io.Discard)
	if err != nil || c.URL != "https://nuevo.supabase.co" {
		t.Fatalf("qdd login vuelve a consultar: %+v %v", c, err)
	}
	remotoConsultarRelay = func(context.Context, int) (remoto.ConfigRelay, error) {
		return remoto.ConfigRelay{}, errors.New("sin red")
	}
	c, err = relayDeFlags(io.Discard)
	if err != nil || c.URL != "https://nuevo.supabase.co" {
		t.Fatalf("sin directorio sigue con el guardado: %+v %v", c, err)
	}
}

// FND-171: un relay guardado como propio, sin este PC emparejado y sin banderas, daba un código que la app no acepta.
func TestLogin_UnRelayPropioSinEmparejarTomaElDelDirectorioYLoDice_FND171(t *testing.T) {
	prepararLogin(t, remoto.ConfigRelay{URL: "https://nuevo.supabase.co", Clave: "sb_publishable_n", Serie: 2})
	if err := remoto.GuardarRelay(remoto.ConfigRelay{URL: "https://antiguo.supabase.co", Clave: "sb_publishable_a", Propio: true}); err != nil {
		t.Fatal(err)
	}
	var errs bytes.Buffer
	c, err := relayDeFlags(&errs)
	if err != nil || c.URL != "https://nuevo.supabase.co" || c.Propio {
		t.Fatalf("debe tomar el del directorio: %+v %v", c, err)
	}
	for _, parte := range []string{"propio", "--relay"} {
		if !strings.Contains(errs.String(), parte) {
			t.Errorf("debe decir %q: %q", parte, errs.String())
		}
	}
	if guardado, err := remoto.LeerRelay(); err != nil || guardado != c {
		t.Fatalf("queda guardado: %+v %v", guardado, err)
	}
}

func TestLogin_UnRelayPropioSinEmparejarQueCoincideNoAvisa_FND171(t *testing.T) {
	prepararLogin(t, remoto.ConfigRelay{URL: "https://nuevo.supabase.co", Clave: "sb_publishable_n", Serie: 2})
	if err := remoto.GuardarRelay(remoto.ConfigRelay{URL: "https://nuevo.supabase.co", Clave: "sb_publishable_n", Propio: true}); err != nil {
		t.Fatal(err)
	}
	var errs bytes.Buffer
	if _, err := relayDeFlags(&errs); err != nil || errs.Len() != 0 {
		t.Fatalf("nada que avisar: %v %q", err, errs.String())
	}
}

func TestLogin_SiNoPuedeConfirmarElRelayLoAvisaAntesDelCodigo_FND171(t *testing.T) {
	for _, propio := range []bool{false, true} {
		prepararLogin(t, remoto.ConfigRelay{})
		guardado := remoto.ConfigRelay{URL: "https://antiguo.supabase.co", Clave: "sb_publishable_a", Propio: propio}
		if err := remoto.GuardarRelay(guardado); err != nil {
			t.Fatal(err)
		}
		var errs bytes.Buffer
		c, err := relayDeFlags(&errs)
		if err != nil || c != guardado {
			t.Fatalf("propio=%v: sigue con el guardado, sin perderlo: %+v %v", propio, c, err)
		}
		for _, parte := range []string{"no se pudo confirmar", "no se pudo leer el directorio", "qdd doctor --fix"} {
			if !strings.Contains(errs.String(), parte) {
				t.Errorf("propio=%v: debe decir %q: %q", propio, parte, errs.String())
			}
		}
	}
}

func TestLogin_ConElPCEmparejadoUnRelayPropioNiSeConsulta_FND171(t *testing.T) {
	prepararLogin(t, remoto.ConfigRelay{URL: "https://nuevo.supabase.co", Clave: "sb_publishable_n", Serie: 2})
	emparejarPrueba(t)
	guardado := remoto.ConfigRelay{URL: "https://mio.supabase.co", Clave: "sb_publishable_m", Propio: true}
	if err := remoto.GuardarRelay(guardado); err != nil {
		t.Fatal(err)
	}
	remotoConsultarRelay = func(context.Context, int) (remoto.ConfigRelay, error) {
		t.Fatal("un PC emparejado en su relay propio no consulta")
		return remoto.ConfigRelay{}, nil
	}
	var errs bytes.Buffer
	if c, err := relayDeFlags(&errs); err != nil || c != guardado || errs.Len() != 0 {
		t.Fatalf("no se toca: %+v %v %q", c, err, errs.String())
	}
}

func TestLogin_UnRelayPorDefectoSecretoNoSeGuarda(t *testing.T) {
	prepararLogin(t, remoto.ConfigRelay{URL: "https://x.supabase.co", Clave: "sb_secret_nunca"})
	if _, err := relayDeFlags(io.Discard); err == nil {
		t.Fatal("la clave por defecto pasa por la misma validación")
	}
	if _, err := remoto.LeerRelay(); err == nil {
		t.Fatal("no se guarda")
	}
}

// FND-152: en un PC sin relay, dispositivos y revocar pedían la URL y la clave, justo lo que qdd login evita (issue #102).
func TestRemoto_SinRelayMandaAQddLoginYNoPideLaClave_FND152(t *testing.T) {
	prepararLogin(t, remoto.ConfigRelay{})
	var salida, errores bytes.Buffer
	if c := remotoDispositivos(context.Background(), &salida, &errores); c != 1 {
		t.Fatalf("sin relay, dispositivos falla: %d", c)
	}
	if !strings.Contains(errores.String(), "qdd login") || strings.Contains(errores.String(), "--clave") {
		t.Fatalf("manda a qdd login, sin pedir la clave: %q", errores.String())
	}
	errores.Reset()
	if c := remotoRevocar(context.Background(), "d1", &salida, &errores); c != 1 {
		t.Fatalf("sin relay, revocar falla: %d", c)
	}
	if !strings.Contains(errores.String(), "qdd login") || strings.Contains(errores.String(), "--clave") {
		t.Fatalf("manda a qdd login, sin pedir la clave: %q", errores.String())
	}
}

// FND-152: la ayuda de qdd remoto presenta qdd login como el camino, y las banderas como opción avanzada (issue #102).
func TestRemoto_LaAyudaNoPideLaClaveParaActivar_FND152(t *testing.T) {
	if strings.Contains(remotoCmd.Long, "qdd remoto activar --relay <url> --clave <publicable>\n") {
		t.Fatal("activar no exige banderas: son para quien opera su propio relay")
	}
	if !strings.Contains(remotoCmd.Long, "qdd remoto activar [--relay <url> --clave <publicable>]") {
		t.Fatalf("la ayuda dice que las banderas son opcionales: %s", remotoCmd.Long)
	}
}
