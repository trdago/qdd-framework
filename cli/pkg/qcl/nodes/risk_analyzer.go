package nodes

import (
	"fmt"

	"github.com/qdd-framework/qdd/pkg/qcl/models"
)

type RiskAnalyzer struct{}

func NewRiskAnalyzer() *RiskAnalyzer {
	return &RiskAnalyzer{}
}

func (r *RiskAnalyzer) Process(session *models.CognitiveSession) error {
	fmt.Println("🤖 [QCL] RiskAnalyzer: Evaluando impacto y riesgos de la intención...")

	if session.Intent == nil {
		return nil
	}

	// Simple heuristic risk analysis
	switch session.Intent.Type {
	case "FIX":
		session.Risks = append(session.Risks, models.Risk{
			Type:        "RSK-001",
			Description: "Riesgo de romper retrocompatibilidad al modificar código existente.",
			Severity:    "HIGH",
		})
		fmt.Println("   -> [!] Riesgo detectado: Modificación de código existente (Posible quiebre de retrocompatibilidad).")
	case "FEATURE":
		session.Risks = append(session.Risks, models.Risk{
			Type:        "RSK-002",
			Description: "Riesgo de aumentar la deuda técnica si no se certifica primero.",
			Severity:    "MEDIUM",
		})
		fmt.Println("   -> [!] Riesgo detectado: Nueva funcionalidad sin certificación previa.")
	case "ASK":
		fmt.Println("   -> [✔] Riesgo nulo (Solo lectura / consulta).")
	}

	return nil
}
