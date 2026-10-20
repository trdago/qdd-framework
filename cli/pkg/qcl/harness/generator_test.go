package harness

import (
	"strings"
	"testing"
)

func TestGenerateSystemPrompt(t *testing.T) {
	testGenerateSystemPromptAllowed(t)
	testGenerateSystemPromptDisabled(t)
}

func testGenerateSystemPromptAllowed(t *testing.T) {
	promptAllowed := GenerateSystemPrompt(true)
	if strings.Contains(promptAllowed, "<security_override>") {
		t.Errorf("Expected prompt without security_override when execution is allowed")
	}
	if !strings.Contains(promptAllowed, "<qdd_agentic_harness>") {
		t.Errorf("Missing root XML tag")
	}
	if !strings.Contains(promptAllowed, "<rule_zero_else>") {
		t.Errorf("Missing rule_zero_else")
	}
	if !strings.Contains(promptAllowed, "<rule_data_uniqueness>") {
		t.Errorf("Missing rule_data_uniqueness")
	}
	if !strings.Contains(promptAllowed, "<long_running_tasks_and_goal_mode>") {
		t.Errorf("Missing long_running_tasks_and_goal_mode")
	}
	if !strings.Contains(promptAllowed, "implementation_plan.md") {
		t.Errorf("Missing implementation_plan.md reference")
	}
	if !strings.Contains(promptAllowed, "walkthrough.md") {
		t.Errorf("Missing walkthrough.md reference")
	}
}

func testGenerateSystemPromptDisabled(t *testing.T) {
	promptDisabled := GenerateSystemPrompt(false)
	if !strings.Contains(promptDisabled, "<security_override>") {
		t.Errorf("Expected prompt WITH security_override when execution is disabled")
	}
	if !strings.Contains(promptDisabled, "FORBIDDEN from modifying code") {
		t.Errorf("Missing forbidden text in security override")
	}
}
