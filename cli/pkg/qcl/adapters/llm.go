package adapters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/qdd-framework/qdd/pkg/qcl/prompts"
)

// CognitiveEngine define la interfaz estándar para comunicarse con modelos
// fundacionales (LLMs) como OpenAI, Anthropic, Gemini, etc.
type CognitiveEngine interface {
	// Ask envía un prompt (y opcionalmente contexto) al LLM y retorna su respuesta.
	Ask(prompt string, systemContext string) (string, error)
	
	// ModelName devuelve el nombre del modelo activo (ej. gpt-4, claude-3)
	ModelName() string
}

type GeminiEngine struct {
	apiKey string
}

func NewGeminiEngine() *GeminiEngine {
	return &GeminiEngine{
		apiKey: os.Getenv("GEMINI_API_KEY"),
	}
}

func (g *GeminiEngine) Ask(prompt string, systemContext string) (string, error) {
	if g.apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY no configurada")
	}

	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-pro-latest:generateContent?key=" + g.apiKey

	// Inyectar el Master Prompt de QDD obligatoriamente
	fullSystemContext := prompts.MasterPrompt

	// Inyectar el Wisdom Registry (Principios de Ingeniería del equipo) si existe
	if cwd, err := os.Getwd(); err == nil {
		wisdomPath := cwd + "/.qdd/wisdom/principles.md"
		if content, err := os.ReadFile(wisdomPath); err == nil {
			fullSystemContext += "\n\n" + string(content)
		}
	}

	fullSystemContext += "\n\n" + systemContext

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": "System Context: " + fullSystemContext + "\n\nUser Prompt: " + prompt},
				},
			},
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respText, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, respText)
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		return result.Candidates[0].Content.Parts[0].Text, nil
	}
	return "", fmt.Errorf("no response from Gemini")
}

func (g *GeminiEngine) ModelName() string {
	return "gemini-1.5-pro"
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
