package cognitive

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDetectLocalLLM_OfflineFallback(t *testing.T) {
	info := DetectLocalLLM()
	if info.Provider == "" {
		t.Errorf("Expected non-empty provider name")
	}
	if len(info.Capabilities) == 0 {
		t.Errorf("Expected non-empty capabilities")
	}
}

func TestExecuteLocalIntent_OfflineGuidance(t *testing.T) {
	ctx := context.Background()
	result, err := ExecuteLocalIntent(ctx, ".", "Crea una función de suma")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Status == "" {
		t.Errorf("Expected non-empty status in ExecutionResult")
	}
	if len(result.GovernanceChecks) == 0 {
		t.Errorf("Expected governance checks in ExecutionResult")
	}
}

func TestCallOllamaGenerate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"response": "Func suma(a, b int) int { return a + b }",
		})
	}))
	defer server.Close()

	ctx := context.Background()
	resp, err := callOllamaGenerate(ctx, server.URL, "llama3", "Haz una suma")
	if err != nil {
		t.Fatalf("callOllamaGenerate failed: %v", err)
	}

	if resp == "" {
		t.Errorf("Expected non-empty response from simulated Ollama")
	}
}

func TestCallOpenAICompatChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		respData := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": "Código generado correctamente sin 'else'.",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(respData)
	}))
	defer server.Close()

	ctx := context.Background()
	resp, err := callOpenAICompatChat(ctx, server.URL, "gpt-4-local", "Test prompt")
	if err != nil {
		t.Fatalf("callOpenAICompatChat failed: %v", err)
	}

	if resp == "" {
		t.Errorf("Expected non-empty response from simulated OpenAI-compatible endpoint")
	}
}
