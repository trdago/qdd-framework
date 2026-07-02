package qcl

import (
	"fmt"

	"github.com/qdd-framework/qdd/pkg/qcl/models"
)

// CognitiveNode es la interfaz para cada filtro de la cadena.
type CognitiveNode interface {
	Process(session *models.CognitiveSession) error
}

// LLMAdapter es el puerto para comunicarse con cualquier IA.
type LLMAdapter interface {
	Analyze(prompt string) (string, error)
}

// Pipeline orquesta la cadena de responsabilidad.
type Pipeline struct {
	nodes []CognitiveNode
}

func NewPipeline(nodes ...CognitiveNode) *Pipeline {
	return &Pipeline{
		nodes: nodes,
	}
}

func (p *Pipeline) Execute(input string) (*models.CognitiveSession, error) {
	session := &models.CognitiveSession{
		RawInput: input,
	}

	for _, node := range p.nodes {
		err := node.Process(session)
		if err != nil {
			return session, fmt.Errorf("error in cognitive node: %w", err)
		}
		
		// Si el nodo generó una petición de aprobación, detenemos el pipeline
		// y devolvemos la sesión para que la UI pregunte al usuario.
		if session.ApprovalRequest != nil {
			return session, nil
		}
	}

	return session, nil
}
