package adapters

// CognitiveEngine define la interfaz estándar para comunicarse con modelos
// fundacionales (LLMs) como OpenAI, Anthropic, Gemini, etc.
type CognitiveEngine interface {
	// Ask envía un prompt (y opcionalmente contexto) al LLM y retorna su respuesta.
	Ask(prompt string, systemContext string) (string, error)
	
	// ModelName devuelve el nombre del modelo activo (ej. gpt-4, claude-3)
	ModelName() string
}

// MockEngine es un simulador que usamos actualmente para pruebas de desarrollo
// antes de integrar una API real.
type MockEngine struct{}

func (m *MockEngine) Ask(prompt string, systemContext string) (string, error) {
	// Aquí se podría derivar parte de la lógica estática que tiene IntentAnalyzer hoy en día
	return "Mock Response", nil
}

func (m *MockEngine) ModelName() string {
	return "mock-engine-v1"
}
