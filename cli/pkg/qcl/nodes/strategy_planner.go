package nodes

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qdd-framework/qdd/pkg/qcl/adapters"
	"github.com/qdd-framework/qdd/pkg/qcl/models"
)

type StrategyPlanner struct{
	Engine adapters.CognitiveEngine
}

func NewStrategyPlanner(engine adapters.CognitiveEngine) *StrategyPlanner {
	return &StrategyPlanner{
		Engine: engine,
	}
}

func (s *StrategyPlanner) Process(session *models.CognitiveSession) error {
	fmt.Println("🤖 [QCL] StrategyPlanner: Formulando estrategia dinámica de ejecución...")

	if session.Intent == nil {
		return nil
	}
	
	if s.Engine == nil {
		fmt.Println("   [!] No engine configured, using fallback.")
		session.Strategy = &models.StrategyPlan{
			Steps:           []string{"Fallback Step 1"},
			TargetArtifacts: []string{"fallback.go"},
		}
		return nil
	}

	systemContext := `Your specific task right now is to create a technical strategy based on the user's input and their detected intent.
If the intent is FEATURE or FIX, your VERY FIRST step MUST be to write a failing unit test (TDD).
List the high-level steps needed to achieve the goal, and list the target files (artifacts) that need to be created or modified.
Respond strictly with a JSON object in this format:
{
  "steps": ["step 1", "step 2"],
  "targetArtifacts": ["file1.go", "file2.js"]
}`

	prompt := fmt.Sprintf("User Input: %s\nDetected Intent: %s", session.RawInput, session.Intent.Type)
	resp, err := s.Engine.Ask(prompt, systemContext)
	if err != nil {
		fmt.Printf("   [!] LLM error en StrategyPlanner: %v\n", err)
		return nil
	}

	// Clean Markdown backticks if any
	resp = strings.TrimPrefix(resp, "```json")
	resp = strings.TrimPrefix(resp, "```")
	resp = strings.TrimSuffix(resp, "```")
	resp = strings.TrimSpace(resp)

	var result struct {
		Steps           []string `json:"steps"`
		TargetArtifacts []string `json:"targetArtifacts"`
	}

	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		fmt.Printf("   [!] JSON Parse error en StrategyPlanner: %v\n", err)
		return nil
	}
	
	// TDD Policy Enforcement
	if session.Intent.Type == "FEATURE" || session.Intent.Type == "FIX" {
		hasTDD := false
		for _, step := range result.Steps {
			if strings.Contains(strings.ToLower(step), "test") || strings.Contains(strings.ToLower(step), "tdd") {
				hasTDD = true
				break
			}
		}
		if !hasTDD {
			result.Steps = append([]string{"TDD: Escribir test unitario fallido para la funcionalidad/bug"}, result.Steps...)
		}
	}

	session.Strategy = &models.StrategyPlan{
		Steps:           result.Steps,
		TargetArtifacts: result.TargetArtifacts,
	}
	
	fmt.Printf("   -> %d pasos y %d archivos objetivo identificados.\n", len(result.Steps), len(result.TargetArtifacts))
	return nil
}
