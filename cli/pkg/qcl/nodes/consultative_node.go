package nodes

import (
	"fmt"
	"strings"

	"github.com/qdd-framework/qdd/pkg/qcl/models"
)

type ConsultativeNode struct{}

func NewConsultativeNode() *ConsultativeNode {
	return &ConsultativeNode{}
}

func (c *ConsultativeNode) Process(session *models.CognitiveSession) error {
	fmt.Println("🤖 [QCL] ConsultativeNode: Evaluando necesidad de estándares (Modo Consultivo)...")
	
	if session.Intent == nil {
		return nil
	}
	
	input := strings.ToLower(session.RawInput)
	
	if session.Intent.Type == "FEATURE" || session.Intent.Type == "REFACTOR" {
		if strings.Contains(input, "ui") || strings.Contains(input, "frontend") || strings.Contains(input, "vue") || strings.Contains(input, "react") {
			fmt.Println("   -> [Modo Consultivo] Dominio UI detectado. Sugiriendo estándar de Accesibilidad y Performance Base.")
			if session.ExecutionPlan != nil {
				session.ExecutionPlan.Quality = append(session.ExecutionPlan.Quality, "Accessibility Standards", "UI Performance Base")
			}
		}
		
		if strings.Contains(input, "api") || strings.Contains(input, "backend") || strings.Contains(input, "endpoint") {
			fmt.Println("   -> [Modo Consultivo] Dominio API detectado. Sugiriendo estándar RESTful estricto u OpenAPI.")
			if session.ExecutionPlan != nil {
				session.ExecutionPlan.Quality = append(session.ExecutionPlan.Quality, "RESTful Strict", "OpenAPI Specification")
			}
		}
	}
	
	return nil
}
