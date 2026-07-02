package nodes

import (
	"fmt"
	"strings"

	"github.com/qdd-framework/qdd/pkg/qcl/models"
)

type IntentAnalyzer struct {
	// Aquí inyectaríamos el LLMAdapter en el futuro
}

func NewIntentAnalyzer() *IntentAnalyzer {
	return &IntentAnalyzer{}
}

func (i *IntentAnalyzer) Process(session *models.CognitiveSession) error {
	fmt.Println("🤖 [QCL] IntentAnalyzer: Analizando input...")
	
	// Simulación básica del comportamiento del LLM
	lowerInput := strings.ToLower(session.RawInput)
	
	session.Intent = &models.IntentModel{
		Objectives: []string{session.RawInput},
		Type:       "ASK", // Default
	}

	// 🛑 Filtro de Ambigüedad
	ambiguas := []string{"algo", "eso", "cosas", "lo de ayer", "ayuda"}
	for _, palabra := range ambiguas {
		if strings.Contains(lowerInput, palabra) {
			session.ClarificationRequest = &models.ClarificationRequest{
				Message: fmt.Sprintf("Ambigüedad detectada. La intención '%s' es demasiado genérica. ¿Qué deseas hacer exactamente?", palabra),
				Options: []string{
					"Agregar Autenticación",
					"Agregar Base de Datos",
					"Agregar Endpoint REST",
					"Auditar Código",
				},
			}
			// Retornamos sin error para que el pipeline devuelva la sesión a la CLI
			return nil
		}
	}

	if strings.Contains(lowerInput, "autenticación") || strings.Contains(lowerInput, "agregar") {
		session.Intent.Type = "FEATURE"
		fmt.Printf("   -> Intención detectada: %s\n", session.Intent.Type)
		return nil
	}
	
	if strings.Contains(lowerInput, "bug") || strings.Contains(lowerInput, "error") {
		session.Intent.Type = "FIX"
		fmt.Printf("   -> Intención detectada: %s\n", session.Intent.Type)
		return nil
	}

	fmt.Printf("   -> Intención detectada: %s\n", session.Intent.Type)
	return nil
}
