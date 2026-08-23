package audit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAIGovernanceChecks(t *testing.T) {
	tempDir := t.TempDir()

	testNonAIProject(t, tempDir)
	testAIDetectionAndPolicyMissing(t, tempDir)
	testSOC2SecretLeakDetection(t, tempDir)
	testISO23894PromptInjectionDetection(t, tempDir)
}

func testNonAIProject(t *testing.T, tempDir string) {
	violations := RunAIGovernanceCheck(tempDir)
	if len(violations) != 0 {
		t.Fatalf("Expected 0 violations for non-AI project, got %d", len(violations))
	}
}

func testAIDetectionAndPolicyMissing(t *testing.T, tempDir string) {
	srcDir := filepath.Join(tempDir, "src")
	os.MkdirAll(srcDir, 0755)

	aiFile := filepath.Join(srcDir, "agent.go")
	content := "package src\n// uses langchain and openai for completions\nfunc RunAgent() {}\n"
	if err := os.WriteFile(aiFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write AI file: %v", err)
	}

	violations := RunAIGovernanceCheck(tempDir)
	if len(violations) == 0 {
		t.Fatalf("Expected violations for AI project without policy or golden sets")
	}

	foundISO42001 := false
	for _, v := range violations {
		if v.RuleID == "ISO42001-GOV-01-MISSING-POLICY" {
			foundISO42001 = true
			break
		}
	}

	if !foundISO42001 {
		t.Fatalf("Expected ISO42001-GOV-01-MISSING-POLICY violation")
	}
}

func testSOC2SecretLeakDetection(t *testing.T, tempDir string) {
	srcDir := filepath.Join(tempDir, "src")
	leakyFile := filepath.Join(srcDir, "llm_client.go")
	content := "package src\nconst openai_api_key = \"sk-12345678901234567890\"\n"
	if err := os.WriteFile(leakyFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write leaky file: %v", err)
	}

	violations := RunAIGovernanceCheck(tempDir)
	foundSOC2 := false
	for _, v := range violations {
		if v.RuleID == "SOC2-AI-03-LEAKED-SECRETS-IN-PROMPTS" {
			foundSOC2 = true
			break
		}
	}

	if !foundSOC2 {
		t.Fatalf("Expected SOC2-AI-03-LEAKED-SECRETS-IN-PROMPTS violation")
	}
}

func testISO23894PromptInjectionDetection(t *testing.T, tempDir string) {
	srcDir := filepath.Join(tempDir, "src")
	injFile := filepath.Join(srcDir, "prompt_builder.go")
	content := "package src\nfunc Build(userInput string) {\n\tvar prompt string\n\tprompt += \"Hello \" + userInput\n}\n"
	if err := os.WriteFile(injFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write prompt injection file: %v", err)
	}

	violations := RunAIGovernanceCheck(tempDir)
	foundRisk := false
	for _, v := range violations {
		if v.RuleID == "ISO23894-RISK-01-UNSANITIZED-PROMPT-INJECTION" {
			foundRisk = true
			break
		}
	}

	if !foundRisk {
		t.Fatalf("Expected ISO23894-RISK-01-UNSANITIZED-PROMPT-INJECTION violation")
	}
}
