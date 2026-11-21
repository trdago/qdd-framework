//go:build !windows

package consola

import (
	"os/exec"
	"strings"
	"testing"
)

func shellsDeLaBarra() map[string]func(*testing.T, string, string) string {
	return map[string]func(*testing.T, string, string) string{"sh": func(t *testing.T, comando, entrada string) string {
		t.Helper()
		cmd := exec.Command("sh", "-c", comando)
		cmd.Stdin = strings.NewReader(entrada)
		out, _ := cmd.CombinedOutput()
		return string(out)
	}}
}
