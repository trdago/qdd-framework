package supervisor

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSupervise_RestartsOnCleanExitThenReportsRealError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	counterFile := filepath.Join(t.TempDir(), "counter")
	// First two runs exit 0 (clean, should restart); third exits 1 (real error).
	script := fmt.Sprintf(
		`n=$(cat %s 2>/dev/null || echo 0); n=$((n+1)); echo $n > %s; if [ "$n" -ge 3 ]; then echo boom-error >&2; exit 1; fi; exit 0`,
		counterFile, counterFile)

	attempts := 0
	report, err := Supervise(ctx, "sh", []string{"-c", script}, func(attempt int) {
		attempts = attempt
	})
	if err != nil {
		t.Fatalf("Supervise returned unexpected error: %v", err)
	}

	assertCrashReport(t, report, 1, "boom-error")
	if attempts < 3 {
		t.Errorf("attempts = %d, want at least 3 (two clean restarts then the failure)", attempts)
	}
}

func assertCrashReport(t *testing.T, report *CrashReport, wantExitCode int, wantOutputSubstr string) {
	t.Helper()
	if report == nil {
		t.Fatal("expected a crash report once the process exits with a real error")
	}
	if report.ExitCode != wantExitCode {
		t.Errorf("ExitCode = %d, want %d", report.ExitCode, wantExitCode)
	}
	if !strings.Contains(report.Output, wantOutputSubstr) {
		t.Errorf("expected captured output to contain %q, got: %q", wantOutputSubstr, report.Output)
	}
}

func TestSupervise_StopsCleanlyWhenContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled before we even start

	report, err := Supervise(ctx, "sh", []string{"-c", "exit 0"}, nil)

	if err == nil {
		t.Error("expected Supervise to return the context error when cancelled")
	}
	if report != nil {
		t.Errorf("expected no crash report on a deliberate cancellation, got: %+v", report)
	}
}
