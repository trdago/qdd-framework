package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/goldenset"
	"github.com/qdd-framework/qdd/pkg/integration"
)

type DoctorInput struct {
	DirsToCreate  []string `json:"dirs_to_create"`
	FilesToCreate []string `json:"files_to_create"`
	AutoFix       bool     `json:"auto_fix"`
}

type DoctorOutput struct {
	Success  bool `json:"success"`
	Failures int  `json:"failures"`
}

func TestDoctorWithGoldenSet(t *testing.T) {
	goldenset.RunSuite(t, "cli_doctor", func(in DoctorInput) (DoctorOutput, error) {
		tempDir := t.TempDir()

		for _, d := range in.DirsToCreate {
			_ = os.MkdirAll(filepath.Join(tempDir, d), 0755)
		}

		for _, f := range in.FilesToCreate {
			_ = os.MkdirAll(filepath.Dir(filepath.Join(tempDir, f)), 0755)
			content := []byte("")
			if strings.HasSuffix(f, ".json") {
				content = []byte("[]")
			}
			_ = os.WriteFile(filepath.Join(tempDir, f), content, 0644)
		}

		success, failures := RunDoctorCheck(tempDir, in.AutoFix)

		return DoctorOutput{
			Success:  success,
			Failures: failures,
		}, nil
	})
}

// Para validaciones específicas de output visual (Evidencia generada)
func validateDoctorReportContains(t *testing.T, qddDir, expectedStatus string) {
	evidenceDir := filepath.Join(qddDir, "project", "evidence", "doctor")
	files, err := os.ReadDir(evidenceDir)
	if err != nil || len(files) == 0 {
		t.Errorf("🚨 Regla violada: RunDoctorCheck no generó el reporte de evidencia en %s", evidenceDir)
		return
	}

	reportContent, _ := os.ReadFile(filepath.Join(evidenceDir, files[0].Name()))
	if !strings.Contains(string(reportContent), expectedStatus) {
		t.Errorf("El reporte no marcó '%s'. Contenido:\n%s", expectedStatus, reportContent)
	}
}

func TestSelectiveFixDashboardContracts(t *testing.T) {
	tempDir := t.TempDir()
	
	historyPath := filepath.Join(tempDir, ".qdd", "project", "metrics", "cognitive_history.json")
	_ = os.MkdirAll(filepath.Dir(historyPath), 0755)
	_ = os.WriteFile(historyPath, []byte("corrupted_no_json"), 0644)
	
	success, _ := RunDoctorCheck(tempDir, false)
	if success {
		t.Errorf("Expected failure for corrupted dashboard contract")
	}
	
	RunDoctorCheck(tempDir, true)
	
	content, err := os.ReadFile(historyPath)
	if err != nil {
		t.Fatalf("Failed to read repaired history: %v", err)
	}
	if string(content) != "[]" {
		t.Errorf("Expected repaired JSON to be '[]', got '%s'", string(content))
	}
}

func escribirRegistro(t *testing.T, dir, ruta, contenido string) {
	t.Helper()
	path := filepath.Join(dir, ".qdd", ruta)
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	if err := os.WriteFile(path, []byte(contenido), 0644); err != nil {
		t.Fatal(err)
	}
}

func itemsFallidos(g *CheckGroup) map[string]string {
	out := map[string]string{}
	for _, item := range g.Items {
		if !item.Success {
			out[item.Name] = item.Error
		}
	}
	return out
}

// FND-061: doctor no señalaba los findings sin estado. Ahora los informa, y con
// --fix escribe la forma canónica de los que sí tienen un estado reconocido.
// El estado que falta lo decide una persona: doctor no lo inventa.
func TestDoctor_RegistrosDelProyecto(t *testing.T) {
	dir := t.TempDir()
	escribirRegistro(t, dir, "project/findings/FND-001.yaml", "title: a\nstatus: RESOLVED\n")
	escribirRegistro(t, dir, "project/findings/FND-002.yaml", "title: b\n")
	escribirRegistro(t, dir, "project/certification/CERT-001.yaml", "id: CERT-001\n")

	fallidos := itemsFallidos(buildRegistrosChecks(dir))
	for _, nombre := range []string{itemFindingsSinEstado, itemFindingsNoCanonicos, itemCertificacionesSinEstado} {
		if _, ok := fallidos[nombre]; !ok {
			t.Errorf("doctor no señaló %q; fallidos = %v", nombre, fallidos)
		}
	}
	if !strings.Contains(fallidos[itemFindingsSinEstado], "FND-002.yaml") {
		t.Errorf("el diagnóstico no nombra el finding sin estado: %q", fallidos[itemFindingsSinEstado])
	}

	runSelectiveFixes(dir, &Checklist{Groups: []*CheckGroup{buildRegistrosChecks(dir)}})

	data, _ := os.ReadFile(filepath.Join(dir, ".qdd", "project", "findings", "FND-001.yaml"))
	if string(data) != "title: a\nstatus: resolved\n" {
		t.Errorf("FND-001 tras --fix = %q; want el estado en minúscula", data)
	}
	data, _ = os.ReadFile(filepath.Join(dir, ".qdd", "project", "findings", "FND-002.yaml"))
	if string(data) != "title: b\n" {
		t.Errorf("doctor --fix inventó el estado de FND-002: %q", data)
	}
	fallidos = itemsFallidos(buildRegistrosChecks(dir))
	if _, ok := fallidos[itemFindingsNoCanonicos]; ok {
		t.Errorf("tras --fix sigue fallando %q", itemFindingsNoCanonicos)
	}
	if _, ok := fallidos[itemFindingsSinEstado]; !ok {
		t.Errorf("tras --fix dejó de señalar el finding sin estado")
	}
}

func TestDoctor_RegistrosSanos(t *testing.T) {
	dir := t.TempDir()
	if fallidos := itemsFallidos(buildRegistrosChecks(dir)); len(fallidos) != 0 {
		t.Errorf("proyecto sin registros: fallidos = %v; want ninguno", fallidos)
	}
	escribirRegistro(t, dir, "project/findings/FND-001.yaml", "title: a\nstatus: open\n")
	escribirRegistro(t, dir, "project/certification/CERT-001.yaml", "id: CERT-001\nstatus: certified\n")
	escribirRegistro(t, dir, "core/certification/CERT-900.yaml", "id: CERT-900\n")
	if fallidos := itemsFallidos(buildRegistrosChecks(dir)); len(fallidos) != 0 {
		t.Errorf("registros sanos: fallidos = %v; want ninguno", fallidos)
	}
}

// FND-109: doctor señala las tareas que parecen creadas por un id equivocado y dice cómo resolverlo; no las toca.
func TestDoctor_TareasQueParecenFantasma_FND109(t *testing.T) {
	dir := t.TempDir()
	escribirRegistro(t, dir, "project/estado.yaml", "formato: 1\netapas: [Código]\nfrentes:\n  - {id: barra-estado, nombre: Segmento de estado en la barra, estado: abierto, celdas: {Código: {estado: ·}}}\n  - {id: segmento-barra, nombre: segmento-barra, estado: abierto, celdas: {Código: {estado: ✅, evidencia: x}}}\n")
	fallidos := itemsFallidos(buildRegistrosChecks(dir))
	if !strings.Contains(fallidos[itemTareasFantasma], "segmento-barra se llama como su id") || !strings.Contains(fallidos[itemTareasFantasma], "--descartado") {
		t.Fatalf("doctor señala la tarea fantasma y cómo descartarla: %v", fallidos)
	}
	runSelectiveFixes(dir, &Checklist{Groups: []*CheckGroup{buildRegistrosChecks(dir)}})
	if _, ok := itemsFallidos(buildRegistrosChecks(dir))[itemTareasFantasma]; !ok {
		t.Fatal("--fix no decide cuál es la buena: lo decide la persona")
	}
}

func TestDoctor_ReglasViejasSeDetectanYFixLasRenueva_FND126(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".qdd"), 0o755); err != nil {
		t.Fatal(err)
	}
	viejo := "Mis notas\n\n### QDD AUTOGENERATED COMMANDS BEGIN ###\n# reglas de antes\n### QDD AUTOGENERATED COMMANDS END ###\n"
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(viejo), 0o644); err != nil {
		t.Fatal(err)
	}
	item := checkBloquesDeReglas(root)
	if item.Success || !strings.Contains(item.Error, "AGENTS.md") || !strings.Contains(item.Error, "qdd doctor --fix") {
		t.Fatalf("falla, nombra el archivo y dice cómo repararlo: %+v", item)
	}
	dispatchRepairCommand(item.Name, root, filepath.Join(root, ".qdd"))
	if item = checkBloquesDeReglas(root); !item.Success {
		t.Fatalf("--fix lo renueva: %+v", item)
	}
	data, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Mis notas") || !strings.Contains(string(data), "QUESTIONS FOR THE HUMAN") {
		t.Fatalf("trae las reglas nuevas y conserva lo demás: %q", data)
	}
}

// MEJ-062: las copias del bloque en los AGENTS.md de carpetas padre llegan a la IA junto con la del proyecto.
func TestDoctor_BloquesEnCarpetasPadreSeDetectanYFixLosQuita_MEJ062(t *testing.T) {
	tope := t.TempDir()
	viejo := integration.TopePadres
	integration.TopePadres = func() string { return tope }
	t.Cleanup(func() { integration.TopePadres = viejo })
	root := filepath.Join(tope, "proyectos", "app")
	if err := os.MkdirAll(filepath.Join(root, ".qdd"), 0o755); err != nil {
		t.Fatal(err)
	}
	bloque := "### QDD AUTOGENERATED COMMANDS BEGIN ###\n# reglas\n### QDD AUTOGENERATED COMMANDS END ###\n"
	if err := os.WriteFile(filepath.Join(tope, "AGENTS.md"), []byte(bloque), 0o644); err != nil {
		t.Fatal(err)
	}
	item := checkReglasUnaVez(root)
	if item.Success || !strings.Contains(item.Error, filepath.Join(tope, "AGENTS.md")) || !strings.Contains(item.Error, "qdd doctor --fix") {
		t.Fatalf("falla, nombra el archivo y dice cómo repararlo: %+v", item)
	}
	dispatchRepairCommand(item.Name, root, filepath.Join(root, ".qdd"))
	if item = checkReglasUnaVez(root); !item.Success {
		t.Fatalf("--fix quita la copia: %+v", item)
	}
	respaldos, _ := filepath.Glob(filepath.Join(tope, "AGENTS.md.qdd-respaldo-*"))
	if len(respaldos) != 1 {
		t.Fatalf("deja un respaldo: %v", respaldos)
	}
}
