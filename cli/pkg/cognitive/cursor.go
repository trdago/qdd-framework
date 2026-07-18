package cognitive

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CursorBackend is best-effort and UNVERIFIED: while building this package,
// no Cursor CLI agent binary was found on the machine used to build it, and
// no non-interactive invocation contract could be confirmed live (unlike
// Antigravity's agentapi and Claude's `claude -p`, both verified). It assumes
// an interface similar to `claude -p` under either candidate binary name.
// Available() safely reports false if neither exists, so this backend is
// simply skipped rather than failing loudly.
type CursorBackend struct{}

func (b *CursorBackend) Name() string { return "cursor" }

func (b *CursorBackend) Available() bool {
	return cursorBinaryPath() != ""
}

func (b *CursorBackend) Ask(ctx context.Context, prompt string) (string, error) {
	path := cursorBinaryPath()
	if path == "" {
		return "", fmt.Errorf("no se encontró un CLI de Cursor disponible")
	}

	cmd := exec.CommandContext(ctx, path, "-p", prompt)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error ejecutando %s: %w", path, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func cursorBinaryPath() string {
	for _, candidate := range []string{"cursor-agent", "cursor"} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}
	return ""
}
