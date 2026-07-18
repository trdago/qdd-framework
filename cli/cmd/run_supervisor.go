package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/qdd-framework/qdd/pkg/bugreport"
	"github.com/qdd-framework/qdd/pkg/cognitive"
	"github.com/qdd-framework/qdd/pkg/integration"
	"github.com/qdd-framework/qdd/pkg/supervisor"
)

// maxAutoRepairAttempts bounds the crash -> auto-repair -> restart cycle so a
// bad fix (or an unfixable bug) can't loop forever burning agent calls. Once
// reached, supervision stops for good and the Finding is left OPEN for a
// human to look at.
const maxAutoRepairAttempts = 3

// executeSupervisor implements `qdd run --keep-alive <command> [args...]`.
// It keeps an external process always running, restarting it on clean exits.
// On a real error, it files a Finding (pkg/bugreport) and then asks a local
// AI agent (pkg/cognitive) to diagnose AND apply the fix directly — see
// ADR-003 for why this replaced the original report-and-pause design, and
// why that requires an unattended, unrestricted agent invocation.
func executeSupervisor(args []string) {
	if len(args) == 0 {
		fmt.Println("[!] Uso: qdd run --keep-alive <comando> [args...]")
		os.Exit(1)
	}

	cwd := resolveSupervisorCwd()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	command, cmdArgs := args[0], args[1:]
	fmt.Printf("[QDD SUPERVISOR] Manteniendo vivo: %s\n", strings.Join(args, " "))

	for attempt := 1; attempt <= maxAutoRepairAttempts; attempt++ {
		report, err := supervisor.Supervise(ctx, command, cmdArgs, logSupervisorAttempt)
		if err != nil {
			fmt.Println("\n[QDD SUPERVISOR] Detenido por el usuario.")
			return
		}

		if !handleSupervisedCrash(ctx, cwd, args, report, attempt) {
			os.Exit(1)
		}
	}

	fmt.Printf("\n[🛑 QDD SUPERVISOR] Se alcanzó el máximo de %d intentos de auto-reparación. Deteniendo para revisión humana.\n", maxAutoRepairAttempts)
	os.Exit(1)
}

func resolveSupervisorCwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("[!] Error obteniendo directorio actual: %v\n", err)
		os.Exit(1)
	}
	return integration.FindProjectRoot(cwd)
}

func logSupervisorAttempt(attempt int) {
	if attempt > 1 {
		fmt.Printf("[QDD SUPERVISOR] Reintento #%d...\n", attempt)
	}
}

// handleSupervisedCrash files the Finding and attempts an auto-repair. It
// returns true if the caller should keep looping (repair succeeded, retry
// the original command) and false if supervision should stop entirely.
func handleSupervisedCrash(ctx context.Context, cwd string, originalArgs []string, report *supervisor.CrashReport, attempt int) bool {
	fmt.Printf("\n[🛑 QDD SUPERVISOR] El proceso terminó con error (exit code %d). Generando reporte...\n", report.ExitCode)

	filed, err := bugreport.File(cwd, bugreport.Report{
		Command:  report.Command,
		Args:     report.Args,
		ExitCode: report.ExitCode,
		Output:   report.Output,
	})
	if err != nil {
		fmt.Printf("[!] Error generando el reporte del fallo: %v\n", err)
		return false
	}
	fmt.Printf("[!] Finding %s generado (%s).\n", filed.Finding.ID, filed.FindingPath)

	fmt.Printf("[QDD SUPERVISOR] Invocando agente de IA para reparar automáticamente (intento %d/%d)...\n", attempt, maxAutoRepairAttempts)
	return attemptAutoRepair(ctx, cwd, originalArgs, report, filed)
}

func attemptAutoRepair(ctx context.Context, cwd string, originalArgs []string, report *supervisor.CrashReport, filed *bugreport.Filed) bool {
	prompt := buildRepairPrompt(report, filed.FindingPath)

	resp, backend, err := cognitive.AttemptRepair(ctx, cwd, prompt)
	if err != nil {
		fmt.Printf("[!] No se pudo invocar un agente con capacidad de reparación: %v\n", err)
		return false
	}
	fmt.Printf("[QDD SUPERVISOR] Agente (%s) respondió.\n", backend)

	verdict, ok := parseRepairVerdict(resp)
	if !ok {
		fmt.Println("[!] El agente no devolvió un veredicto estructurado válido. Deteniendo para revisión humana.")
		return false
	}

	fmt.Printf("[QDD SUPERVISOR] Veredicto: fixed=%v — %s\n", verdict.Fixed, verdict.Summary)
	if !verdict.Fixed {
		return false
	}

	if err := bugreport.MarkResolved(filed.FindingPath, verdict.Summary, verdict.TestAdded); err != nil {
		fmt.Printf("[!] Fix aplicado pero no se pudo actualizar el finding: %v\n", err)
	}
	fmt.Printf("[QDD SUPERVISOR] Fix aplicado. Reanudando: %s\n", strings.Join(originalArgs, " "))
	return true
}

func buildRepairPrompt(report *supervisor.CrashReport, findingPath string) string {
	return fmt.Sprintf(`Eres parte de un ciclo de auto-reparación de QDD (qdd run --keep-alive). Un proceso supervisado falló:

Comando: %s %s
Exit code: %d
Salida capturada (tail):
%s

Ya se registró como Finding en %s. Tu tarea, operando sobre la raíz del proyecto en el directorio actual:
1. Encuentra la causa raíz en el código fuente.
2. Aplica el fix directamente (tienes permiso para editar archivos).
3. Si es razonable, agrega o actualiza un test que reproduzca este fallo (principio Auto-TDD ante Bugs).
4. Corre las pruebas relevantes si existen para confirmar que el fix no rompe nada más.

Al finalizar, responde ÚNICAMENTE con este bloque (nada más antes o después):
[VERDICT_START]
{"fixed": true/false, "summary": "qué cambiaste y por qué", "test_added": true/false}
[VERDICT_END]`, report.Command, strings.Join(report.Args, " "), report.ExitCode, report.Output, findingPath)
}

type repairVerdict struct {
	Fixed     bool   `json:"fixed"`
	Summary   string `json:"summary"`
	TestAdded bool   `json:"test_added"`
}

func parseRepairVerdict(resp string) (repairVerdict, bool) {
	raw, ok := cognitive.ExtractTagged(resp, "[VERDICT_START]", "[VERDICT_END]")
	if !ok {
		return repairVerdict{}, false
	}
	var v repairVerdict
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return repairVerdict{}, false
	}
	return v, true
}
