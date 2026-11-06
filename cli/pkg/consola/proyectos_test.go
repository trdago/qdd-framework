package consola

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/qdd-framework/qdd/pkg/blueprints"
	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/findings"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// RFC 0018 §«Contrato»: qdd entrega una fila por proyecto y el panel solo la dibuja. Sin fuente es null («sin medir»),
// nunca un cero ni un valor inventado.

var enProyectos = time.Date(2026, 11, 6, 15, 0, 0, 0, time.UTC)

func registrar(t *testing.T, rutas ...string) {
	t.Helper()
	t.Setenv("QDD_HOME", t.TempDir())
	home, err := blueprints.Home()
	if err != nil {
		t.Fatal(err)
	}
	reg := registro.Registro{Formato: registro.Formato}
	for i, ruta := range rutas {
		reg.Proyectos = append(reg.Proyectos, registro.Proyecto{Ruta: ruta, Clave: fmt.Sprintf("clave-%d", i), Visto: enProyectos.Add(-time.Duration(i) * time.Hour)})
	}
	b, err := json.Marshal(reg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "proyectos.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func escribirHistorial(t *testing.T, root string, certs ...audit.Certificate) {
	t.Helper()
	ruta := filepath.Join(root, ".qdd", "project", "metrics", "certificate_history.json")
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(certs)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func escribirFinding(t *testing.T, root, nombre, status, fecha string) {
	t.Helper()
	if err := os.MkdirAll(findings.Dir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	contenido := fmt.Sprintf("id: %s\ntitle: un hallazgo\nstatus: %s\ndate: %q\n", nombre, status, fecha)
	if err := os.WriteFile(filepath.Join(findings.Dir(root), nombre+".yaml"), []byte(contenido), 0o644); err != nil {
		t.Fatal(err)
	}
}

func unaFila(t *testing.T) FilaProyecto {
	t.Helper()
	datos := Proyectos(enProyectos)
	if len(datos.Proyectos) != 1 {
		t.Fatalf("debe entregar una fila: %+v", datos)
	}
	return datos.Proyectos[0]
}

func TestProyectos_SinRegistroEsUnaListaVacia(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	datos := Proyectos(enProyectos)
	if datos.Aviso != "" || len(datos.Proyectos) != 0 {
		t.Fatalf("sin registro no hay nada que avisar: %+v", datos)
	}
	b, err := json.Marshal(datos)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"proyectos":[]`) {
		t.Errorf("la lista vacía debe ser [], no null: %s", b)
	}
}

func TestProyectos_RegistroIlegibleAvisaSinDarLaRuta(t *testing.T) {
	home := t.TempDir()
	t.Setenv("QDD_HOME", home)
	if err := os.WriteFile(filepath.Join(home, "proyectos.json"), []byte("{no es json"), 0o600); err != nil {
		t.Fatal(err)
	}
	datos := Proyectos(enProyectos)
	if !strings.Contains(datos.Aviso, "qdd doctor --fix") {
		t.Errorf("debe decir cómo se repara: %q", datos.Aviso)
	}
	if strings.Contains(datos.Aviso, home) {
		t.Errorf("el aviso no debe traer la ruta: %q", datos.Aviso)
	}
}

func TestProyectos_CarpetaQueYaNoExiste(t *testing.T) {
	registrar(t, filepath.Join(t.TempDir(), "se-fue"))
	fila := unaFila(t)
	if fila.Existe || fila.Nombre != "se-fue" {
		t.Fatalf("debe decir que no existe, con su nombre: %+v", fila)
	}
	if fila.Puntaje != nil || fila.Hace != "" || fila.FindingsAbiertos != nil || fila.Esperan != nil || fila.Fallan != nil {
		t.Errorf("sin carpeta todo queda sin medir: %+v", fila)
	}
}

func TestProyectos_SinFuentesTodoQuedaSinMedir(t *testing.T) {
	root := t.TempDir()
	registrar(t, root)
	fila := unaFila(t)
	if !fila.Existe {
		t.Fatalf("la carpeta existe: %+v", fila)
	}
	if fila.Puntaje != nil || fila.Hace != "" || fila.FindingsAbiertos != nil || fila.Esperan != nil || fila.Fallan != nil {
		t.Errorf("sin fuentes nunca un cero: %+v", fila)
	}
	if fila.Puntajes == nil || fila.FindingsPorFecha == nil {
		t.Errorf("los historiales vacíos deben ser [], no null: %+v", fila)
	}
}

func TestProyectos_FuentesIlegiblesQuedanSinMedir(t *testing.T) {
	root := t.TempDir()
	registrar(t, root)
	ruta := filepath.Join(root, ".qdd", "project", "metrics", "certificate_history.json")
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte("{roto"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, estado.Ruta), []byte(": no: es: yaml"), 0o644); err != nil {
		t.Fatal(err)
	}
	fila := unaFila(t)
	if fila.Puntaje != nil || fila.Esperan != nil || fila.Fallan != nil {
		t.Errorf("una fuente ilegible no es un cero: %+v", fila)
	}
}

func TestProyectos_LaFilaSaleDeLasFuentesReales(t *testing.T) {
	root := filepath.Join(t.TempDir(), "mi-proyecto")
	if err := os.MkdirAll(filepath.Join(root, ".qdd", "project"), 0o755); err != nil {
		t.Fatal(err)
	}
	registrar(t, root)
	escribirHistorial(t, root,
		audit.Certificate{Timestamp: "2026-11-05T10:00:00Z", Score: 70, Escala: audit.EscalaActual},
		audit.Certificate{Timestamp: "2026-11-06T09:00:00Z", Score: 80, Escala: audit.EscalaActual},
		audit.Certificate{Timestamp: "2026-11-06T12:00:00Z", Score: 90, Escala: audit.EscalaActual},
	)
	escribirFinding(t, root, "FND-1", "open", "2026-11-05")
	escribirFinding(t, root, "FND-2", "resolved", "2026-11-05")
	escribirFinding(t, root, "FND-3", "wontfix", "2026-11-06")
	escribirFinding(t, root, "FND-4", "", "2026-11-06")
	if err := estado.Init(root, []string{"Código", "PR"}, enProyectos); err != nil {
		t.Fatal(err)
	}
	if err := estado.Set(root, estado.Cambio{Frente: "a", Nombre: "A", Etapa: "Código", Estado: "⏳", Bloqueo: "falta", DeQuien: "Claude"}, enProyectos); err != nil {
		t.Fatal(err)
	}
	if err := estado.Set(root, estado.Cambio{Frente: "a", Etapa: "PR", Estado: "❌", Falla: "CI rojo"}, enProyectos); err != nil {
		t.Fatal(err)
	}

	fila := unaFila(t)
	if fila.Nombre != "mi-proyecto" || fila.Clave != "clave-0" {
		t.Errorf("nombre y clave del registro: %+v", fila)
	}
	if fila.Puntaje == nil || *fila.Puntaje != 90 || fila.Hace != "hace 3 h" {
		t.Errorf("puntaje y fecha de la última certificación: %+v", fila)
	}
	if fila.FindingsAbiertos == nil || *fila.FindingsAbiertos != 1 {
		t.Errorf("findings abiertos con el lector único: %+v", fila)
	}
	if fila.Esperan == nil || *fila.Esperan != 1 || fila.Fallan == nil || *fila.Fallan != 1 {
		t.Errorf("⏳ y ❌ del estado: %+v", fila)
	}
	if len(fila.Puntajes) != 2 || fila.Puntajes[1] != (PuntajeDelDia{Fecha: "2026-11-06", Puntaje: 90}) {
		t.Errorf("el último puntaje de cada día: %+v", fila.Puntajes)
	}
	if len(fila.FindingsPorFecha) != 2 {
		t.Fatalf("findings por fecha: %+v", fila.FindingsPorFecha)
	}
	if fila.FindingsPorFecha[0] != (FindingsDelDia{Fecha: "2026-11-05", Abiertos: 1, Cerrados: 1}) {
		t.Errorf("abiertos y cerrados por separado: %+v", fila.FindingsPorFecha[0])
	}
	if fila.FindingsPorFecha[1] != (FindingsDelDia{Fecha: "2026-11-06", Cerrados: 1, SinEstado: 1}) {
		t.Errorf("descartado es cerrado y sin estado va aparte: %+v", fila.FindingsPorFecha[1])
	}
}

func TestProyectos_PuntajeDeOtraEscalaNoSeCompara(t *testing.T) {
	root := t.TempDir()
	registrar(t, root)
	escribirHistorial(t, root,
		audit.Certificate{Timestamp: "2026-11-04T10:00:00Z", Score: 90, Escala: audit.EscalaActual},
		audit.Certificate{Timestamp: "2026-11-05T10:00:00Z", Score: 40, Escala: audit.EscalaActual - 1},
		audit.Certificate{Timestamp: "2026-11-05T11:00:00Z", Score: 45, Escala: audit.EscalaActual - 1},
	)
	fila := unaFila(t)
	if fila.Puntaje != nil || fila.Hace != "" {
		t.Errorf("otra escala cuenta como sin medir en la tabla: %+v", fila)
	}
	if len(fila.Puntajes) != 2 || fila.Puntajes[0].OtraEscala || !fila.Puntajes[1].OtraEscala || fila.Puntajes[1].Puntaje != 45 {
		t.Errorf("en el historial se marca aparte: %+v", fila.Puntajes)
	}
}

func TestProyectos_UnPuntajeCeroYCeroFindingsSonMedidas(t *testing.T) {
	root := t.TempDir()
	registrar(t, root)
	escribirHistorial(t, root, audit.Certificate{Timestamp: "2026-11-06T14:30:00Z", Score: 0, Escala: audit.EscalaActual})
	if err := os.MkdirAll(findings.Dir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	fila := unaFila(t)
	if fila.Puntaje == nil || *fila.Puntaje != 0 || fila.Hace != "hace 30 min" {
		t.Errorf("un 0 medido se muestra: %+v", fila)
	}
	if fila.FindingsAbiertos == nil || *fila.FindingsAbiertos != 0 {
		t.Errorf("una carpeta vacía son 0 findings: %+v", fila)
	}
}

func TestProyectos_FechaImposibleNoInventaUnHace(t *testing.T) {
	casos := map[string]string{"fecha rota": "ayer", "reloj adelantado": "2026-11-06T16:00:00Z"}
	for nombre, fecha := range casos {
		t.Run(nombre, func(t *testing.T) {
			root := t.TempDir()
			registrar(t, root)
			escribirHistorial(t, root, audit.Certificate{Timestamp: fecha, Score: 77, Escala: audit.EscalaActual})
			fila := unaFila(t)
			if fila.Hace != "" {
				t.Errorf("no hay un «hace» que decir: %q", fila.Hace)
			}
		})
	}
}

func TestProyectos_FindingSinFechaVaAparte(t *testing.T) {
	root := t.TempDir()
	registrar(t, root)
	escribirFinding(t, root, "FND-1", "open", "")
	escribirFinding(t, root, "FND-2", "open", "la semana pasada")
	escribirFinding(t, root, "FND-3", "open", "2026-13-45")
	fila := unaFila(t)
	if fila.FindingsSinFecha != 3 || len(fila.FindingsPorFecha) != 0 {
		t.Errorf("sin fecha válida no se inventa un día: %+v", fila)
	}
	if fila.FindingsAbiertos == nil || *fila.FindingsAbiertos != 3 {
		t.Errorf("igual cuentan como abiertos: %+v", fila)
	}
}

func TestProyectos_FechaConHoraCuentaEnSuDia(t *testing.T) {
	root := t.TempDir()
	registrar(t, root)
	escribirFinding(t, root, "FND-1", "open", "2026-11-05T10:00:00Z")
	fila := unaFila(t)
	if len(fila.FindingsPorFecha) != 1 || fila.FindingsPorFecha[0].Fecha != "2026-11-05" {
		t.Errorf("cuenta en su día: %+v", fila.FindingsPorFecha)
	}
}

func TestProyectos_LosHistorialesTienenTopeYGuardanLoUltimo(t *testing.T) {
	root := t.TempDir()
	registrar(t, root)
	var certs []audit.Certificate
	dia := enProyectos.AddDate(0, 0, -(TopeHistorial + 10))
	for i := 0; i <= TopeHistorial+10; i++ {
		fecha := dia.AddDate(0, 0, i)
		certs = append(certs, audit.Certificate{Timestamp: fecha.Format(time.RFC3339), Score: i % 100, Escala: audit.EscalaActual})
		escribirFinding(t, root, fmt.Sprintf("FND-%d", i), "open", fecha.Format("2006-01-02"))
	}
	escribirHistorial(t, root, certs...)
	fila := unaFila(t)
	if len(fila.Puntajes) != TopeHistorial || len(fila.FindingsPorFecha) != TopeHistorial {
		t.Fatalf("tope de %d: %d y %d", TopeHistorial, len(fila.Puntajes), len(fila.FindingsPorFecha))
	}
	if fila.Puntajes[TopeHistorial-1].Fecha != "2026-11-06" || fila.FindingsPorFecha[TopeHistorial-1].Fecha != "2026-11-06" {
		t.Errorf("debe guardar lo último: %+v y %+v", fila.Puntajes[TopeHistorial-1], fila.FindingsPorFecha[TopeHistorial-1])
	}
}

func TestProyectos_NoEntregaLaRutaCompleta(t *testing.T) {
	root := filepath.Join(t.TempDir(), "proyecto-privado")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	registrar(t, root)
	b, err := json.Marshal(Proyectos(enProyectos))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), filepath.Dir(root)) {
		t.Errorf("del proyecto solo se muestra el nombre de la carpeta: %s", b)
	}
	if !strings.Contains(string(b), "proyecto-privado") {
		t.Errorf("falta el nombre: %s", b)
	}
}

func TestProyectos_NoEscribeEnElProyecto(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".qdd", "project"), 0o755); err != nil {
		t.Fatal(err)
	}
	registrar(t, root)
	escribirHistorial(t, root, audit.Certificate{Timestamp: "2026-11-06T12:00:00Z", Score: 90, Escala: audit.EscalaActual})
	escribirFinding(t, root, "FND-1", "open", "2026-11-05")
	if err := estado.Init(root, []string{"Código"}, enProyectos); err != nil {
		t.Fatal(err)
	}
	antes := fotoCarpeta(t, root)
	Proyectos(enProyectos)
	if despues := fotoCarpeta(t, root); despues != antes {
		t.Errorf("cambió el proyecto:\nantes:\n%s\ndespués:\n%s", antes, despues)
	}
}

func fotoCarpeta(t *testing.T, root string) string {
	t.Helper()
	var foto strings.Builder
	err := filepath.WalkDir(root, func(ruta string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		fmt.Fprintf(&foto, "%s %d %s\n", ruta, info.Size(), info.ModTime().Format(time.RFC3339Nano))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return foto.String()
}

func TestProyectos_FindingConFechaEnOtroCampoCuentaEnSuDia_FND170(t *testing.T) {
	root := t.TempDir()
	registrar(t, root)
	if err := os.MkdirAll(findings.Dir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(findings.Dir(root), "a.yaml"), []byte("id: A\nstatus: open\ndiscovered_at: \"2026-11-03T09:00:00Z\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(findings.Dir(root), "b.yaml"), []byte("id: B\nstatus: resolved\ncreated_at: 2026-11-03T10:00:00-03:00\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fila := unaFila(t)
	if fila.FindingsSinFecha != 0 {
		t.Errorf("tienen fecha, en otro campo: %+v", fila)
	}
	if len(fila.FindingsPorFecha) != 1 || fila.FindingsPorFecha[0] != (FindingsDelDia{Fecha: "2026-11-03", Abiertos: 1, Cerrados: 1}) {
		t.Errorf("cuentan en su día: %+v", fila.FindingsPorFecha)
	}
}
