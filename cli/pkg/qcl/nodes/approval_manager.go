package nodes

import (
	"fmt"

	"github.com/qdd-framework/qdd/pkg/qcl/models"
)

type ApprovalManager struct{}

func NewApprovalManager() *ApprovalManager {
	return &ApprovalManager{}
}

func (a *ApprovalManager) Process(session *models.CognitiveSession) error {
	fmt.Println("🤖 [QCL] ApprovalManager: Verificando si el plan de ejecución requiere autorización humana...")

	if len(session.Risks) > 0 {
		var highRiskCount int
		for _, risk := range session.Risks {
			if risk.Severity == "HIGH" || risk.Severity == "CRITICAL" {
				highRiskCount++
			}
		}

		if highRiskCount > 0 {
			fmt.Println("   -> [ALERTA] Se detectaron riesgos altos o críticos. Solicitando autorización...")
			session.ApprovalRequest = &models.ApprovalRequest{
				Reason: "El plan de ejecución involucra riesgos críticos para los contratos públicos o la base de datos.",
				Risks:  session.Risks,
			}
			return nil
		}
	}

	fmt.Println("   -> Aprobación automática. No hay riesgos críticos.")
	return nil
}
