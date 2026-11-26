//go:build windows

package consola

import (
	"os/exec"
	"strings"
	"syscall"
	"testing"
)

// En Windows, Claude Code corre la barra con Git Bash o, sin él, con cmd como lo hace Node con shell: true.
func shellsDeLaBarra() map[string]func(*testing.T, string, string) string {
	return map[string]func(*testing.T, string, string) string{
		"cmd": func(t *testing.T, comando, entrada string) string {
			t.Helper()
			cmd := exec.Command("cmd.exe")
			cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `cmd.exe /d /s /c "` + comando + `"`}
			cmd.Stdin = strings.NewReader(entrada)
			out, _ := cmd.CombinedOutput()
			return string(out)
		},
		"bash": func(t *testing.T, comando, entrada string) string {
			t.Helper()
			cmd := exec.Command("bash", "-c", comando)
			cmd.Stdin = strings.NewReader(entrada)
			out, _ := cmd.CombinedOutput()
			return string(out)
		},
	}
}
