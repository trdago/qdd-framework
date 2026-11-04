package cmd

import (
	"bytes"
	"context"
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
	antesEjecutor, antesUsuario, antesDefecto := remotoEjecutor, remotoUsuario, remotoRelayPorDefecto
	remotoEjecutor = func(string, ...string) error { return nil }
	remotoUsuario = func() (string, error) { return usuario, nil }
	remotoRelayPorDefecto = porDefecto
	remotoCadaEspera, remotoRelayURL, remotoClave, remotoSinServ = 5*time.Millisecond, "", "", false
	t.Cleanup(func() {
		remotoEjecutor, remotoUsuario, remotoRelayPorDefecto = antesEjecutor, antesUsuario, antesDefecto
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

func TestLogin_LoGuardadoGanaAlPorDefecto(t *testing.T) {
	prepararLogin(t, remoto.ConfigRelay{URL: "https://por-defecto.supabase.co", Clave: "sb_publishable_defecto"})
	propio := remoto.ConfigRelay{URL: "https://propio.supabase.co", Clave: "sb_publishable_propio"}
	if err := remoto.GuardarRelay(propio); err != nil {
		t.Fatal(err)
	}
	c, err := relayDeFlags()
	if err != nil || c.URL != propio.URL {
		t.Fatalf("un PC con su relay no cambia de relay: %+v %v", c, err)
	}
}

func TestLogin_LasBanderasGananAlPorDefecto(t *testing.T) {
	prepararLogin(t, remoto.ConfigRelay{URL: "https://por-defecto.supabase.co", Clave: "sb_publishable_defecto"})
	remotoRelayURL, remotoClave = "https://propio.supabase.co", "sb_publishable_propio"
	c, err := relayDeFlags()
	if err != nil || c.URL != "https://propio.supabase.co" {
		t.Fatalf("--relay y --clave mandan: %+v %v", c, err)
	}
}

// FND-133: un PC sin relay guardado toma el que trae qdd, sin banderas, y lo deja guardado para el servicio.
func TestLogin_UnPCNuevoUsaElRelayQueTraeQdd_FND133(t *testing.T) {
	prepararLogin(t, remoto.RelayPorDefecto)
	c, err := relayDeFlags()
	if err != nil {
		t.Fatalf("un PC nuevo no necesita banderas: %v", err)
	}
	guardado, err := remoto.LeerRelay()
	if err != nil || guardado != c {
		t.Fatalf("queda guardado: %v", err)
	}
}

func TestLogin_UnRelayPorDefectoSecretoNoSeGuarda(t *testing.T) {
	prepararLogin(t, remoto.ConfigRelay{URL: "https://x.supabase.co", Clave: "sb_secret_nunca"})
	if _, err := relayDeFlags(); err == nil {
		t.Fatal("la clave por defecto pasa por la misma validación")
	}
	if _, err := remoto.LeerRelay(); err == nil {
		t.Fatal("no se guarda")
	}
}
