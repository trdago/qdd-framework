// Package cognitive gives QDD itself a way to ask a locally available AI
// agent a single-turn question and get a structured answer back — agnostic
// of which agent/IDE is actually installed (Antigravity, Claude Code, and
// best-effort Cursor). It is the inverse direction of the framework's normal
// MCP flow (where an external AI calls QDD's tools); here QDD calls the AI.
//
// This exists for judgment calls the deterministic Fast Path can't make on
// its own — not for open-ended autonomous coding sessions. Backends are tried
// in order and the first available one answers.
package cognitive

import (
	"context"
	"errors"
	"strings"
)

// Backend is one AI agent QDD can delegate a single-turn question to.
type Backend interface {
	Name() string
	Available() bool
	Ask(ctx context.Context, prompt string) (string, error)
}

// defaultBackends lists every backend QDD knows how to talk to, in the order
// they're tried. Antigravity and Claude are verified against real CLIs;
// Cursor is best-effort (see cursor.go).
func defaultBackends() []Backend {
	return []Backend{&AntigravityBackend{}, &ClaudeBackend{}, &CursorBackend{}}
}

// Ask tries each available backend in order and returns the first successful
// response, along with which backend answered. It returns an error only if
// no backend is available or every available one failed.
func Ask(ctx context.Context, prompt string) (response string, backend string, err error) {
	return askUsing(ctx, prompt, defaultBackends())
}

// RepairCapable is implemented only by backends that can attempt an
// unattended fix — i.e. invoke the agent with real file-write/tool access,
// not just a read-only single-turn judgment. This is a separate, explicitly
// named interface from Backend/Ask so callers can't accidentally grant
// file-write capability through the generic, safe Ask() path.
type RepairCapable interface {
	Backend
	AttemptRepair(ctx context.Context, cwd, prompt string) (string, error)
}

// AttemptRepair tries each available RepairCapable backend in order and
// returns the first one's response. Backends that don't implement
// RepairCapable (unverified ones like Cursor) are skipped entirely.
func AttemptRepair(ctx context.Context, cwd, prompt string) (response string, backend string, err error) {
	return attemptRepairUsing(ctx, cwd, prompt, defaultBackends())
}

func attemptRepairUsing(ctx context.Context, cwd, prompt string, backends []Backend) (string, string, error) {
	for _, b := range backends {
		repairable, ok := b.(RepairCapable)
		if !ok || !b.Available() {
			continue
		}
		resp, err := repairable.AttemptRepair(ctx, cwd, prompt)
		if err != nil {
			continue
		}
		return resp, b.Name(), nil
	}
	return "", "", errors.New("ningún agente disponible soporta reparación autónoma")
}

func askUsing(ctx context.Context, prompt string, backends []Backend) (string, string, error) {
	for _, b := range backends {
		if !b.Available() {
			continue
		}
		resp, err := b.Ask(ctx, prompt)
		if err != nil {
			continue
		}
		return resp, b.Name(), nil
	}
	return "", "", errors.New("no hay ningún agente de IA disponible (Antigravity/Claude/Cursor)")
}

// ExtractTagged pulls the trimmed substring between startTag and endTag out
// of text. Used to parse a [VERDICT_START]...[VERDICT_END]-wrapped JSON
// block out of whichever backend answered, independent of its raw format.
func ExtractTagged(text, startTag, endTag string) (string, bool) {
	startIdx := strings.Index(text, startTag)
	if startIdx == -1 {
		return "", false
	}
	rest := text[startIdx+len(startTag):]
	endIdx := strings.Index(rest, endTag)
	if endIdx == -1 {
		return "", false
	}
	return strings.TrimSpace(rest[:endIdx]), true
}
