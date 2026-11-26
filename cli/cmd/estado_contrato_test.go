package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// Contrato de la ficha estado-por-tareas, de caja negra: «qdd estado …» de verdad, en otro proceso con su os.Exit.

// correrQDDEn corre qdd con args en dir como otro proceso (envComoQDD) y devuelve su salida (stdout y stderr) y su código.
func correrQDDEn(t *testing.T, dir, home string, args ...string) (string, int) {
	t.Helper()
	c := exec.Command(os.Args[0], args...)
	c.Dir = dir
	c.Env = append(sinGitDelPadre(), envComoQDD+"=1", "QDD_HOME="+home, "CLAUDECODE=")
	salida, err := c.CombinedOutput()
	if err == nil {
		return string(salida), 0
	}
	salio, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("no corrió: %v", err)
	}
	return string(salida), salio.ExitCode()
}

// gitFechado corre git en dir con autor y fecha fijos: los avisos comparan la fecha de los commits con la última celda.
func gitFechado(t *testing.T, dir string, en time.Time, args ...string) {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	fecha := en.Format(time.RFC3339)
	c.Env = append(sinGitDelPadre(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		"GIT_AUTHOR_DATE="+fecha, "GIT_COMMITTER_DATE="+fecha)
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func fichaPagosCon(decision string) string {
	return `id: pagos
nombre: Pagos
estado: aprobado
rutas: [src/pagos]
objetivo_operativo:
    problema: cobrar
    criterio_de_exito: se cobra
objetivo_tecnologico:
    arquitectura: un servicio
    decisiones: ["` + decision + `"]
pruebas:
    comandos: [go test ./...]
`
}

// RFC 0017 §2 y §3: con commits nuevos en las rutas de su ficha y la ficha cambiada después de su última celda, qdd
// avisa y pide revisar, pero check sale con 0 y el tablero queda igual: ni el nombre ni el criterio cambian solos.
// contrato: estado-por-tareas#6ffefcf9
func TestEstadoCheck_AvisarCommitsYPedirRevisarNoBloqueaNiReescribe(t *testing.T) {
	tocada := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	home := t.TempDir()
	t.Setenv("QDD_HOME", home)
	tablero := `formato: 2
etapas: [Código]
objetivos:
  - {id: caja, nombre: Caja, criterio: se cobra con Webpay, estado: abierto}
frentes:
  - {id: cobro, nombre: Cobro con Webpay, estado: abierto, objetivo: caja, modulo: pagos, tocada_el: "` + tocada.Format(time.RFC3339) + `", celdas: {Código: {estado: "❌", falla: falta el cobro}}}
`
	escribirEnRepo(t, dir, ".qdd/project/modules/pagos.yaml", fichaPagosCon("Webpay"))
	escribirEnRepo(t, dir, "src/pagos/pago.go", "package pagos\n")
	escribirEnRepo(t, dir, estado.Ruta, tablero)
	gitFechado(t, dir, tocada.Add(-48*time.Hour), "init", "-q")
	gitFechado(t, dir, tocada.Add(-48*time.Hour), "add", "-A")
	gitFechado(t, dir, tocada.Add(-48*time.Hour), "commit", "-qm", "base")
	escribirEnRepo(t, dir, "src/pagos/pago.go", "package pagos // cambio\n")
	escribirEnRepo(t, dir, ".qdd/project/modules/pagos.yaml", fichaPagosCon("Mercado Pago"))
	gitFechado(t, dir, tocada.Add(time.Hour), "add", "-A")
	gitFechado(t, dir, tocada.Add(time.Hour), "commit", "-qm", "cambio en pagos")

	salida, codigo := correrQDDEn(t, dir, home, "estado", "check")
	if codigo != 0 {
		t.Fatalf("el aviso de commits no bloquea: check sale con 0, salió con %d\n%s", codigo, salida)
	}
	if !strings.Contains(salida, "tarea cobro: 1 commit en las rutas de su ficha pagos") {
		t.Fatalf("check avisa de los commits nuevos en el módulo de la tarea:\n%s", salida)
	}
	revisar := estado.LineasPorRevisar(dir)
	if len(revisar) != 1 || !strings.Contains(revisar[0], "Revisa el nombre de la tarea y el criterio") {
		t.Fatalf("el contexto pide revisar el nombre y el criterio: %q", revisar)
	}
	despues, err := os.ReadFile(filepath.Join(dir, estado.Ruta))
	if err != nil {
		t.Fatal(err)
	}
	if string(despues) != tablero {
		t.Fatalf("avisar y pedir revisar no reescriben el tablero:\n%s", despues)
	}
}

// Un script con set -e que late su tarea sigue aunque el pulso no se pueda anotar (la tarea se cerró, no existe o no
// hay tablero): el pulso es una señal, no un paso del script (FND-275).
// contrato: estado-por-tareas#24b379fd
func TestEstadoPulso_NuncaCortaElComandoQueLoProduce_FND275(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	escribirEnRepo(t, dir, estado.Ruta, `formato: 1
etapas: [Código]
frentes:
  - {id: a, nombre: A, estado: cerrado, motivo_cierre: hecha, celdas: {Código: {estado: "—"}}}
  - {id: b, nombre: B, estado: abierto, celdas: {Código: {estado: "❌", falla: rota}}}
`)
	sinTablero := t.TempDir()
	for _, caso := range []struct {
		dir  string
		args []string
	}{
		{dir, []string{"estado", "pulso", "a", "--fuente", "comando"}},
		{dir, []string{"estado", "pulso", "no-existe", "--fuente", "comando"}},
		{dir, []string{"estado", "pulso", "b", "--fuente", "otra"}},
		{sinTablero, []string{"estado", "pulso", "b", "--fuente", "comando"}},
	} {
		inicio := time.Now()
		salida, codigo := correrQDDEn(t, caso.dir, home, caso.args...)
		if codigo != 0 {
			t.Errorf("qdd %s: el pulso no corta el script (salió con %d)\n%s", strings.Join(caso.args, " "), codigo, salida)
		}
		if !strings.Contains(salida, "pulso") {
			t.Errorf("qdd %s: dice por qué no anotó el pulso:\n%s", strings.Join(caso.args, " "), salida)
		}
		if d := time.Since(inicio); d > 5*time.Second {
			t.Errorf("qdd %s: el pulso no demora el script: %s", strings.Join(caso.args, " "), d)
		}
	}
	if salida, codigo := correrQDDEn(t, dir, home, "estado", "pulso", "b", "--fuente", "comando"); codigo != 0 || salida != "" {
		t.Fatalf("un pulso que se anota no dice nada: %d %q", codigo, salida)
	}
}
