// Package supervisor implements `qdd run --keep-alive`: it keeps an external
// process (a service, a binary, anything) always running, restarting it on
// clean exits. On a real error (non-zero exit code) it stops immediately and
// hands back a CrashReport instead of silently looping — QDD never tries to
// repair the target process's code itself (Modo Consultivo: report, don't
// auto-fix). The caller is expected to turn that report into a Finding
// (see pkg/bugreport) and only resume supervision once it's resolved.
package supervisor

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"time"
)

// CrashReport captures everything needed to file a bug for a supervised
// process that exited with a real error.
type CrashReport struct {
	Command    string
	Args       []string
	ExitCode   int
	Output     string
	OccurredAt time.Time
}

// maxCapturedOutput bounds how much combined stdout/stderr we keep in memory
// for the crash report, so a chatty long-running process can't exhaust RAM.
const maxCapturedOutput = 16 * 1024

// Supervise runs command/args repeatedly. Clean exits (code 0) are treated as
// expected and trigger an immediate restart, keeping the process "always
// alive". A non-zero exit is treated as a real error: supervision stops and a
// CrashReport is returned. If ctx is cancelled (e.g. Ctrl+C) before that
// happens, Supervise returns (nil, ctx.Err()) — a deliberate stop is not a bug.
func Supervise(ctx context.Context, command string, args []string, onAttempt func(attempt int)) (*CrashReport, error) {
	for attempt := 1; ; attempt++ {
		if onAttempt != nil {
			onAttempt(attempt)
		}

		exitCode, output, err := runOnce(ctx, command, args)
		if err != nil {
			return nil, err
		}
		if exitCode == 0 {
			continue
		}

		return &CrashReport{
			Command:    command,
			Args:       args,
			ExitCode:   exitCode,
			Output:     output,
			OccurredAt: time.Now(),
		}, nil
	}
}

// runOnce executes command/args to completion, streaming its output live to
// the terminal while also capturing a bounded tail of it. It returns the
// process's exit code, or a non-nil error only when ctx was cancelled before
// the process could finish (a deliberate supervisor stop).
func runOnce(ctx context.Context, command string, args []string) (int, string, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	var captured boundedBuffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &captured)
	cmd.Stderr = io.MultiWriter(os.Stderr, &captured)

	runErr := cmd.Run()
	if ctx.Err() != nil {
		return 0, captured.String(), ctx.Err()
	}
	return exitCodeOf(runErr), captured.String(), nil
}

func exitCodeOf(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 1
}

// boundedBuffer keeps only the last maxCapturedOutput bytes written to it.
type boundedBuffer struct {
	buf bytes.Buffer
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	n, err := b.buf.Write(p)
	if b.buf.Len() > maxCapturedOutput {
		trimmed := b.buf.Bytes()[b.buf.Len()-maxCapturedOutput:]
		b.buf = *bytes.NewBuffer(append([]byte(nil), trimmed...))
	}
	return n, err
}

func (b *boundedBuffer) String() string {
	return b.buf.String()
}
