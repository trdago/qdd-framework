package dashboard

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/audit"
)

// enProyecto crea un proyecto vacío y entra en él: BuildState lee el directorio actual.
func enProyecto(t *testing.T, archivos map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".qdd", "project"), 0755)
	for rel, contenido := range archivos {
		p := filepath.Join(dir, rel)
		os.MkdirAll(filepath.Dir(p), 0755)
		if err := os.WriteFile(p, []byte(contenido), 0644); err != nil {
			t.Fatal(err)
		}
	}
	old, _ := os.Getwd()
	os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(old) })
	return dir
}

// FND-287: sin mcp.log ni state.json el dashboard mostraba dos líneas de log
// del MCP que nadie escribió y la versión «v0.1.1».
// contrato: dashboard#f88cd0d7
func TestBuildState_SinFuenteDiceSinMedirYNoInventa(t *testing.T) {
	enProyecto(t, nil)

	s := BuildState()
	if s.Score != 0 || s.Grade != audit.GradoSinMedir {
		t.Errorf("sin certificación: puntaje %d %q; want 0 y %q", s.Score, s.Grade, audit.GradoSinMedir)
	}
	if len(s.Historical) != 0 {
		t.Errorf("historial sin fuente: %v", s.Historical)
	}
	if s.Version != "" {
		t.Errorf("versión sin state.json = %q; want vacía (la página dice sin medir)", s.Version)
	}
	for _, l := range s.MCPLogs {
		if strings.HasPrefix(l, "[MCP]") {
			t.Errorf("línea de log del MCP sin mcp.log: %q", l)
		}
	}
}

// Con fuente, el dashboard muestra lo que dice la fuente.
// contrato: dashboard#f88cd0d7
func TestBuildState_ConFuenteMuestraLaFuente(t *testing.T) {
	enProyecto(t, map[string]string{
		".qdd/mcp.log":    "[MCP] qdd_audit llamado\n",
		".qdd/state.json": `{"version":"3.4.5"}`,
	})

	s := BuildState()
	if s.Version != "3.4.5" {
		t.Errorf("versión = %q; want la de state.json", s.Version)
	}
	var vista bool
	for _, l := range s.MCPLogs {
		vista = vista || l == "[MCP] qdd_audit llamado"
		if strings.Contains(l, "Servidor QDD inicializado") {
			t.Errorf("línea inventada junto a la real: %q", l)
		}
	}
	if !vista {
		t.Errorf("logs = %v; want la línea de mcp.log", s.MCPLogs)
	}
}

// El puntaje es el de la última certificación, aunque el código de hoy tenga
// violaciones: refrescar no audita ni corre las pruebas.
// contrato: dashboard#5c623d60
func TestBuildState_PuntajeDeLaUltimaCertificacionSinAuditar(t *testing.T) {
	historial := `[{"timestamp":"2026-11-01T10:00:00Z","score":70,"escala":2},{"timestamp":"2026-11-02T10:00:00Z","score":84,"escala":2}]`
	dir := enProyecto(t, map[string]string{
		".qdd/project/metrics/certificate_history.json": historial,
		".qdd/config.yaml": "test_commands:\n  - \"touch auditoria-corrio\"\n",
		"go.mod":           "module x\n",
		"main.go":          "package main\n\nfunc f(a bool) int {\n\tif a {\n\t\treturn 1\n\t} else {\n\t\treturn 2\n\t}\n}\n",
	})

	s := BuildState()
	if s.Score != 84 || s.Grade != audit.Grado(84) {
		t.Errorf("puntaje = %d %q; want 84 %q (audit.Score de la última certificación)", s.Score, s.Grade, audit.Grado(84))
	}
	if _, err := os.Stat(filepath.Join(dir, "auditoria-corrio")); err == nil {
		t.Error("refrescar el dashboard corrió la auditoría (sus pruebas)")
	}
	got, _ := os.ReadFile(filepath.Join(dir, ".qdd", "project", "metrics", "certificate_history.json"))
	if string(got) != historial {
		t.Errorf("refrescar cambió el historial de certificaciones:\n%s", got)
	}
}

// huellaDelArbol lista cada archivo del proyecto con su tamaño y su hora de modificación.
func huellaDelArbol(t *testing.T, dir string) map[string]string {
	t.Helper()
	h := map[string]string{}
	filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		h[p] = fmt.Sprint(info.ModTime(), " ", info.Size())
		return nil
	})
	return h
}

// Con el proyecto quieto, el vigilante no reconstruye, no escribe y no difunde.
// contrato: dashboard#3d6c6819
func TestVigilar_QuietoNoReconstruyeNiEscribeNiDifunde(t *testing.T) {
	dir := proyectoMedido(t)
	var reconstrucciones atomic.Int32
	antes := difundir
	difundir = func() { reconstrucciones.Add(1); antes() }
	t.Cleanup(func() { difundir = antes })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	listo := make(chan struct{})
	go Vigilar(ctx, dir, listo)
	<-listo
	ch := Broker.AddClient()
	defer Broker.RemoveClient(ch)
	mensajesEn(ch, 200*time.Millisecond) // estado inicial, si llegó tarde
	inicial := reconstrucciones.Load()
	huella := huellaDelArbol(t, dir)

	n := mensajesEn(ch, 2500*time.Millisecond)

	if n != 0 {
		t.Errorf("con el proyecto quieto se difundieron %d mensajes; want 0", n)
	}
	if r := reconstrucciones.Load() - inicial; r != 0 {
		t.Errorf("con el proyecto quieto hubo %d reconstrucciones; want 0", r)
	}
	despues := huellaDelArbol(t, dir)
	for p, h := range despues {
		if huella[p] != h {
			t.Errorf("con el proyecto quieto se escribió %s", p)
		}
	}
	for p := range huella {
		if _, ok := despues[p]; !ok {
			t.Errorf("con el proyecto quieto se borró %s", p)
		}
	}
}
