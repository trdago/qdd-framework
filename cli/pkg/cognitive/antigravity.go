package cognitive

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	antigravityPollInterval = 500 * time.Millisecond
	antigravityTimeout      = 45 * time.Second
)

// AntigravityBackend delegates to Antigravity's local agent CLI (agentapi).
// Unlike Claude's `-p` mode, agentapi is asynchronous: it starts a
// conversation and the answer has to be read back reactively from a
// transcript.jsonl log file once the model writes to it.
type AntigravityBackend struct{}

func (b *AntigravityBackend) Name() string { return "antigravity" }

// Available requires both the agentapi binary AND ANTIGRAVITY_LS_ADDRESS
// (the running IDE's language server address) — agentapi fails immediately
// without it, so treating it as available otherwise would be dishonest.
func (b *AntigravityBackend) Available() bool {
	return agentapiPath() != "" && os.Getenv("ANTIGRAVITY_LS_ADDRESS") != ""
}

func (b *AntigravityBackend) Ask(ctx context.Context, prompt string) (string, error) {
	path := agentapiPath()
	if path == "" {
		return "", fmt.Errorf("agentapi no está disponible en este entorno")
	}

	convID, err := startAntigravityConversation(ctx, path, prompt)
	if err != nil {
		return "", err
	}
	return waitForAntigravityResponse(ctx, convID)
}

func agentapiPath() string {
	if path, err := exec.LookPath("agentapi"); err == nil {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	fallback := filepath.Join(home, ".gemini", "antigravity", "bin", "agentapi")
	if _, err := os.Stat(fallback); err == nil {
		return fallback
	}
	return ""
}

func startAntigravityConversation(ctx context.Context, path, prompt string) (string, error) {
	cmd := exec.CommandContext(ctx, path, "new-conversation", "--model=pro", prompt)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error ejecutando agentapi: %w", err)
	}

	var initResp struct {
		Response struct {
			NewConversation struct {
				ConversationID string `json:"conversationId"`
			} `json:"newConversation"`
		} `json:"response"`
	}
	if err := json.Unmarshal(output, &initResp); err != nil {
		return "", fmt.Errorf("error parseando respuesta inicial de agentapi: %w", err)
	}
	return initResp.Response.NewConversation.ConversationID, nil
}

// waitForAntigravityResponse polls transcript.jsonl until the MODEL's turn
// appears, the context is cancelled, or antigravityTimeout elapses.
func waitForAntigravityResponse(ctx context.Context, convID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	transcriptPath := filepath.Join(home, ".gemini", "antigravity", "brain", convID, ".system_generated", "logs", "transcript.jsonl")
	deadline := time.Now().Add(antigravityTimeout)

	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if content, ok := readModelResponse(transcriptPath); ok {
			return content, nil
		}
		time.Sleep(antigravityPollInterval)
	}
	return "", fmt.Errorf("timeout esperando respuesta del agente (ID: %s)", convID)
}

func readModelResponse(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if content, ok := modelContentFromLine(line); ok {
			return content, true
		}
	}
	return "", false
}

func modelContentFromLine(line string) (string, bool) {
	if !strings.Contains(line, `"source":"MODEL"`) {
		return "", false
	}
	var step struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(line), &step); err != nil || step.Content == "" {
		return "", false
	}
	return step.Content, true
}
