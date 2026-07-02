package nodes

import (
	"fmt"
	"gopkg.in/yaml.v3"

	"github.com/qdd-framework/qdd/pkg/qcl/models"
)

type PlanBuilder struct{}

func NewPlanBuilder() *PlanBuilder {
	return &PlanBuilder{}
}

func (p *PlanBuilder) Process(session *models.CognitiveSession) error {
	fmt.Println("🤖 [QCL] PlanBuilder: Construyendo Execution Plan...")

	intentType := "UNKNOWN"
	if session.Intent != nil {
		intentType = session.Intent.Type
	}

	session.ExecutionPlan = &models.ExecutionPlan{
		Goal:          session.RawInput,
		Intent:        intentType,
		Strategy:      []string{"learn", "discover", "feature", "certify"},
		Artifacts:     []string{"certification", "evidence"},
		Quality:       []string{"world-class"},
		Compatibility: "strict",
	}

	yamlData, _ := yaml.Marshal(session.ExecutionPlan)
	fmt.Printf("\n--- Execution Plan Generado ---\n%s-------------------------------\n", string(yamlData))

	return nil
}
