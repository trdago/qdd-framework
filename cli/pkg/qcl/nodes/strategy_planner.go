package nodes

import (
	"fmt"

	"github.com/qdd-framework/qdd/pkg/qcl/models"
)

type StrategyPlanner struct{}

func NewStrategyPlanner() *StrategyPlanner {
	return &StrategyPlanner{}
}

func (s *StrategyPlanner) Process(session *models.CognitiveSession) error {
	fmt.Println("🤖 [QCL] StrategyPlanner: Formulando estrategia de mitigación y artefactos...")

	if session.Intent == nil {
		return nil
	}

	session.Strategy = &models.StrategyPlan{
		Steps:           []string{"Certificar primero", "Escribir test unitario", "Refactorizar código"},
		TargetArtifacts: []string{"certification", "finding"},
	}

	if len(session.Risks) > 0 {
		for _, risk := range session.Risks {
			if risk.Severity == "HIGH" {
				fmt.Println("   -> [🛡️] Mitigación Crítica: Agregando Requirement de ADR (Architecture Decision Record).")
				session.Strategy.TargetArtifacts = append(session.Strategy.TargetArtifacts, "adr")
				session.Strategy.Steps = append([]string{"Redactar ADR"}, session.Strategy.Steps...)
			}
		}
	}
	if len(session.Risks) == 0 {
		fmt.Println("   -> [✔] Estrategia estándar aplicada.")
	}

	return nil
}
