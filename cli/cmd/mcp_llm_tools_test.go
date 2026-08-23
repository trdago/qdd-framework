package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/server"
	"github.com/qdd-framework/qdd/pkg/cognitive"
)

func TestLocalLLMMCPTools(t *testing.T) {
	s := server.NewMCPServer("test-server", "1.0.0")
	registerLocalLLMTools(s)

	testLocalLLMStatus(t)
	testLocalLLMExec(t)
}

func testLocalLLMStatus(t *testing.T) {
	info := cognitive.DetectLocalLLM()
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Failed to marshal LocalLLMInfo: %v", err)
	}

	var parsed cognitive.LocalLLMInfo
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if parsed.Provider == "" {
		t.Errorf("Expected non-empty provider in parsed status")
	}
	if parsed.Status == "" {
		t.Errorf("Expected non-empty status in parsed status")
	}
}

func testLocalLLMExec(t *testing.T) {
	ctx := context.Background()
	res, err := cognitive.ExecuteLocalIntent(ctx, ".", "Crea una función sin 'else'")
	if err != nil {
		t.Fatalf("ExecuteLocalIntent failed: %v", err)
	}

	if res.Status == "" {
		t.Errorf("Expected non-empty status in execution result")
	}
	if len(res.GovernanceChecks) == 0 {
		t.Errorf("Expected governance checks in execution result")
	}

	foundZeroElse := false
	for _, chk := range res.GovernanceChecks {
		if strings.Contains(chk, "Zero-Else") || strings.Contains(chk, "Consultivo") {
			foundZeroElse = true
			break
		}
	}

	if !foundZeroElse {
		t.Errorf("Expected Zero-Else or Consultivo check in governance checks, got %v", res.GovernanceChecks)
	}
}
