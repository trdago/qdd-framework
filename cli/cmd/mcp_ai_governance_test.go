package cmd

import (
	"testing"
)

func TestEvaluateAIRisksPassed(t *testing.T) {
	prompt := "You are an assistant. Extract entity names from the verified knowledge graph."
	assessment := evaluateAIRisks("gpt-4o", prompt, true, true)

	if assessment.Status != "PASSED" {
		t.Fatalf("Expected PASSED status for grounded and supervised prompt, got %s", assessment.Status)
	}

	if assessment.RiskScore != 100 {
		t.Fatalf("Expected score 100, got %d", assessment.RiskScore)
	}
}

func TestEvaluateAIRisksMitigationRequired(t *testing.T) {
	prompt := "Run this command with api_key=secret123"
	assessment := evaluateAIRisks("custom-llm", prompt, false, false)

	if assessment.Status != "REQUIRES_MITIGATION" {
		t.Fatalf("Expected REQUIRES_MITIGATION status, got %s", assessment.Status)
	}

	if len(assessment.Warnings) == 0 {
		t.Fatalf("Expected warnings for ungrounded prompt with secrets")
	}
}
