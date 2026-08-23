package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/qdd-framework/qdd/pkg/cognitive"
	"github.com/qdd-framework/qdd/pkg/dashboard"
)

func TestDashboardStateIncludesLocalLLM(t *testing.T) {
	tempDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origWd)

	os.MkdirAll(filepath.Join(tempDir, ".qdd"), 0755)

	state := dashboard.BuildState()
	if state.LocalLLM.Provider == "" {
		t.Errorf("Expected state.LocalLLM.Provider to be populated, got empty string")
	}
	if state.LocalLLM.Status == "" {
		t.Errorf("Expected state.LocalLLM.Status to be populated, got empty string")
	}
}

func TestDashboardIntentHandler_Success(t *testing.T) {
	handler := createIntentHandler()

	reqBody := `{"input": "Revisa el código para asegurar cero-else"}`
	req := httptest.NewRequest(http.MethodPost, "/api/intent", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected HTTP 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var result cognitive.ExecutionResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Status == "" {
		t.Errorf("Expected non-empty status in ExecutionResult")
	}
	if len(result.GovernanceChecks) == 0 {
		t.Errorf("Expected governance checks in ExecutionResult")
	}
}

func TestDashboardIntentHandler_MethodNotAllowed(t *testing.T) {
	handler := createIntentHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/intent", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Expected HTTP 405 Method Not Allowed, got %d", w.Code)
	}
}

func TestDashboardIntentHandler_InvalidJSON(t *testing.T) {
	handler := createIntentHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/intent", bytes.NewBufferString("invalid json string"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected HTTP 400 Bad Request, got %d", w.Code)
	}
}

func TestDashboardAgentExecuteHandler_Success(t *testing.T) {
	handler := createAgentExecuteHandler()

	reqBody := `{"input": "Genera una función con salida temprana"}`
	req := httptest.NewRequest(http.MethodPost, "/api/agent/execute", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected HTTP 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var result cognitive.ExecutionResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Status == "" {
		t.Errorf("Expected non-empty status in ExecutionResult")
	}
}

func createIntentHandler() http.HandlerFunc {
	type IntentRequest struct {
		Input string `json:"input"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req IntentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		cwd, _ := os.Getwd()
		execResult, err := cognitive.ExecuteLocalIntent(r.Context(), cwd, req.Input)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(execResult)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(execResult)
	}
}

func createAgentExecuteHandler() http.HandlerFunc {
	type IntentRequest struct {
		Input string `json:"input"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req IntentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		cwd, _ := os.Getwd()
		execResult, _ := cognitive.ExecuteLocalIntent(r.Context(), cwd, req.Input)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(execResult)
	}
}
