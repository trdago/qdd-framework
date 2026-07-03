package nodes

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qdd-framework/qdd/pkg/qcl/adapters"
	"github.com/qdd-framework/qdd/pkg/qcl/models"
)

type IntentAnalyzer struct {
	Engine adapters.CognitiveEngine
}

func NewIntentAnalyzer(engine adapters.CognitiveEngine) *IntentAnalyzer {
	return &IntentAnalyzer{
		Engine: engine,
	}
}

func (i *IntentAnalyzer) Process(session *models.CognitiveSession) error {
	fmt.Println("🤖 [QCL] IntentAnalyzer: Analizando input con LLM...")

	session.Intent = &models.IntentModel{
		Objectives: []string{session.RawInput},
		Type:       "ASK", // Default
	}

	if i.Engine == nil {
		fmt.Println("   [!] No engine configured, using fallback.")
		return nil
	}

	systemContext := `Your specific task right now is to classify the user's intent into one of the following types:
- FEATURE (Adding a new feature or functionality)
- FIX (Fixing a bug or error)
- ASK (Asking a question, default)
- CLARIFY (If the request is too ambiguous)

Respond strictly with a JSON object in this format:
{
  "type": "FEATURE|FIX|ASK|CLARIFY",
  "reasoning": "string"
}`

	resp, err := i.Engine.Ask(session.RawInput, systemContext)
	if err != nil {
		fmt.Printf("   [!] LLM error: %v. Using fallback.\n", err)
		return nil
	}
	
	// Remove markdown ticks if present
	resp = strings.TrimPrefix(resp, "```json")
	resp = strings.TrimPrefix(resp, "```")
	resp = strings.TrimSuffix(resp, "```")
	resp = strings.TrimSpace(resp)

	var result struct {
		Type      string `json:"type"`
		Reasoning string `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		fmt.Printf("   [!] JSON Parse error: %v. Raw: %s\n", err, resp)
		return nil
	}

	if result.Type == "CLARIFY" {
		session.ClarificationRequest = &models.ClarificationRequest{
			Message: "Ambigüedad detectada: " + result.Reasoning,
			Options: []string{"Agregar Autenticación", "Agregar Base de Datos", "Auditar Código"},
		}
		return nil
	}

	session.Intent.Type = result.Type
	fmt.Printf("   -> Intención detectada por %s: %s\n", i.Engine.ModelName(), session.Intent.Type)
	return nil
}
