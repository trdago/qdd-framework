package cognitive

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ClaudeBackend delegates to the Claude Code CLI's non-interactive print mode
// (`claude -p "<prompt>"`), which replies directly on stdout in one turn —
// verified live against a real `claude` binary while building this package.
type ClaudeBackend struct{}

func (b *ClaudeBackend) Name() string { return "claude" }

func (b *ClaudeBackend) Available() bool {
	return claudeBinaryPath() != ""
}

func (b *ClaudeBackend) Ask(ctx context.Context, prompt string) (string, error) {
	path := claudeBinaryPath()
	if path == "" {
		return "", fmt.Errorf("claude no está disponible en este entorno")
	}

	cmd := exec.CommandContext(ctx, path, "-p", prompt)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error ejecutando claude -p: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// AttemptRepair runs claude with --dangerously-skip-permissions so it can
// write files and run tools unattended — verified live to be the only way
// to get an unattended file edit out of `claude -p` (without it, Claude Code
// correctly refuses to write and asks for interactive approval instead).
// This is deliberately not used by the default Ask(): only callers that
// explicitly want autonomous repair should reach this.
func (b *ClaudeBackend) AttemptRepair(ctx context.Context, cwd, prompt string) (string, error) {
	path := claudeBinaryPath()
	if path == "" {
		return "", fmt.Errorf("claude no está disponible en este entorno")
	}

	cmd := exec.CommandContext(ctx, path, "-p", "--dangerously-skip-permissions", prompt)
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error ejecutando claude --dangerously-skip-permissions: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// claudeBinaryPath prefers whatever `claude` resolves to in PATH, falling
// back to CLAUDE_CODE_EXECPATH (set by IDEs like Antigravity that bundle
// their own Claude Code binary rather than relying on a global install).
func claudeBinaryPath() string {
	if path, err := exec.LookPath("claude"); err == nil {
		return path
	}
	if path := os.Getenv("CLAUDE_CODE_EXECPATH"); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}
