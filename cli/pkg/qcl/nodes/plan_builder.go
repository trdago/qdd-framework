package nodes

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/qdd-framework/qdd/pkg/qcl/adapters"
	"github.com/qdd-framework/qdd/pkg/qcl/models"
)

type PlanBuilder struct{
	Engine adapters.CognitiveEngine
}

func NewPlanBuilder(engine adapters.CognitiveEngine) *PlanBuilder {
	return &PlanBuilder{
		Engine: engine,
	}
}

func (p *PlanBuilder) Process(session *models.CognitiveSession) error {
	fmt.Println("🤖 [QCL] PlanBuilder: Construyendo Execution Plan dinámico...")

	intentType := "UNKNOWN"
	if session.Intent != nil {
		intentType = session.Intent.Type
	}
	
	if p.Engine == nil {
		fmt.Println("   [!] No engine configured, using fallback.")
		session.ExecutionPlan = &models.ExecutionPlan{
			Goal:          session.RawInput,
			Intent:        intentType,
			Strategy:      []string{"fallback strategy"},
			Artifacts:     []string{"fallback.go"},
			Quality:       []string{"world-class"},
			Compatibility: "strict",
		}
		yamlData, _ := yaml.Marshal(session.ExecutionPlan)
		fmt.Printf("\n--- Execution Plan Generado ---\n%s-------------------------------\n", string(yamlData))
		return nil
	}

	systemContext := `Your specific task right now is to output a structured JSON that represents the Execution Plan based on the user's input, intent, and strategy.
Respond strictly with a JSON object in this format:
{
  "goal": "string (the overarching goal)",
  "intent": "string (the intent type)",
  "strategy": ["step 1", "step 2"],
  "artifacts": ["file1.go"],
  "quality": ["world-class"],
  "compatibility": "strict"
}`

	prompt := fmt.Sprintf("User Input: %s\nDetected Intent: %s", session.RawInput, intentType)
	if session.Strategy != nil {
		prompt += fmt.Sprintf("\nStrategy Steps: %v\nTarget Artifacts: %v", session.Strategy.Steps, session.Strategy.TargetArtifacts)
	}

	resp, err := p.Engine.Ask(prompt, systemContext)
	if err != nil {
		fmt.Printf("   [!] LLM error en PlanBuilder: %v\n", err)
		return nil
	}

	// Clean Markdown backticks if any
	resp = strings.TrimPrefix(resp, "```json")
	resp = strings.TrimPrefix(resp, "```")
	resp = strings.TrimSuffix(resp, "```")
	resp = strings.TrimSpace(resp)

	var plan models.ExecutionPlan
	if err := json.Unmarshal([]byte(resp), &plan); err != nil {
		fmt.Printf("   [!] JSON Parse error en PlanBuilder: %v\n", err)
		return nil
	}

	session.ExecutionPlan = &plan

	yamlData, _ := yaml.Marshal(session.ExecutionPlan)
	fmt.Printf("\n--- Execution Plan Generado (QDD AI) ---\n%s----------------------------------------\n", string(yamlData))

	return nil
}
