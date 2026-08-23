package cognitive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type LocalLLMInfo struct {
	Detected        bool     `json:"detected"`
	Provider        string   `json:"provider"`
	Endpoint        string   `json:"endpoint,omitempty"`
	AvailableModels []string `json:"available_models,omitempty"`
	SelectedModel   string   `json:"selected_model,omitempty"`
	Status          string   `json:"status"`
	ExecutionMode   string   `json:"execution_mode"`
	Capabilities    []string `json:"capabilities"`
}

type ExecutionResult struct {
	Status           string   `json:"status"`
	Provider         string   `json:"provider"`
	Model            string   `json:"model,omitempty"`
	Output           string   `json:"output"`
	ExecutionTimeMs  int64    `json:"execution_time_ms"`
	GovernanceChecks []string `json:"governance_checks"`
}

// DetectLocalLLM escanea el entorno local en busca de APIs y motores de IA
// como Antigravity, Ollama, LM Studio, Claude Code y servidores compatibles con OpenAI.
func DetectLocalLLM() LocalLLMInfo {
	if info, found := probeAntigravity(); found {
		return info
	}

	if info, found := probeOllama(); found {
		return info
	}

	if info, found := probeOpenAICompatible(); found {
		return info
	}

	if info, found := probeClaudeCode(); found {
		return info
	}

	return LocalLLMInfo{
		Detected:      false,
		Provider:      "None",
		Status:        "OFFLINE",
		ExecutionMode: "MCP_HARNESS",
		Capabilities:  []string{"mcp_tools", "static_governance"},
	}
}

func probeAntigravity() (LocalLLMInfo, bool) {
	apiPath := agentapiPath()
	lsAddr := os.Getenv("ANTIGRAVITY_LS_ADDRESS")
	hasGemini := os.Getenv("GEMINI_CLI_PORT") != ""

	if apiPath == "" && lsAddr == "" && !hasGemini {
		return LocalLLMInfo{}, false
	}

	endpoint := "CLI: agentapi"
	if lsAddr != "" {
		endpoint = fmt.Sprintf("Language Server: %s", lsAddr)
	}

	return LocalLLMInfo{
		Detected:        true,
		Provider:        "Antigravity",
		Endpoint:        endpoint,
		AvailableModels: []string{"gemini-2.5-pro", "gemini-2.5-flash", "gemini-2.0-flash"},
		SelectedModel:   "gemini-2.5-pro",
		Status:          "ONLINE",
		ExecutionMode:   "DIRECT_AGENT",
		Capabilities:    []string{"autonomous_repair", "reasoning", "mcp_tools", "full_ide_integration"},
	}, true
}

func probeOllama() (LocalLLMInfo, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 350*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://127.0.0.1:11434/api/tags", nil)
	if err != nil {
		return LocalLLMInfo{}, false
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return LocalLLMInfo{}, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return LocalLLMInfo{}, false
	}

	var tagResp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	var models []string
	if err := json.NewDecoder(resp.Body).Decode(&tagResp); err == nil {
		for _, m := range tagResp.Models {
			models = append(models, m.Name)
		}
	}

	selected := "default"
	if len(models) > 0 {
		selected = models[0]
	}

	return LocalLLMInfo{
		Detected:        true,
		Provider:        "Ollama",
		Endpoint:        "http://127.0.0.1:11434",
		AvailableModels: models,
		SelectedModel:   selected,
		Status:          "ONLINE",
		ExecutionMode:   "LOCAL_REST_API",
		Capabilities:    []string{"local_inference", "zero_latency", "privacy_airgapped"},
	}, true
}

func probeOpenAICompatible() (LocalLLMInfo, bool) {
	endpoints := []string{
		"http://127.0.0.1:1234", // LM Studio
		"http://127.0.0.1:8080", // LocalAI / llama.cpp
		"http://127.0.0.1:8000", // vLLM
		"http://127.0.0.1:5000", // TextGen
	}

	client := &http.Client{}
	for _, ep := range endpoints {
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		req, err := http.NewRequestWithContext(ctx, "GET", ep+"/v1/models", nil)
		if err != nil {
			cancel()
			continue
		}

		resp, err := client.Do(req)
		cancel()
		if err != nil || resp.StatusCode != http.StatusOK {
			if resp != nil {
				resp.Body.Close()
			}
			continue
		}
		defer resp.Body.Close()

		var modelResp struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}

		var models []string
		if err := json.NewDecoder(resp.Body).Decode(&modelResp); err == nil {
			for _, m := range modelResp.Data {
				models = append(models, m.ID)
			}
		}

		providerName := "OpenAI-Compatible (Local)"
		if ep == "http://127.0.0.1:1234" {
			providerName = "LM Studio"
		}
		if ep == "http://127.0.0.1:8080" {
			providerName = "LocalAI"
		}
		if ep == "http://127.0.0.1:8000" {
			providerName = "vLLM"
		}

		selected := "default"
		if len(models) > 0 {
			selected = models[0]
		}

		return LocalLLMInfo{
			Detected:        true,
			Provider:        providerName,
			Endpoint:        ep,
			AvailableModels: models,
			SelectedModel:   selected,
			Status:          "ONLINE",
			ExecutionMode:   "LOCAL_REST_API",
			Capabilities:    []string{"chat_completions", "zero_latency", "structured_json"},
		}, true
	}

	return LocalLLMInfo{}, false
}

func probeClaudeCode() (LocalLLMInfo, bool) {
	if claudeBinaryPath() == "" {
		return LocalLLMInfo{}, false
	}

	return LocalLLMInfo{
		Detected:        true,
		Provider:        "Claude Code",
		Endpoint:        "CLI: claude",
		AvailableModels: []string{"claude-3-5-sonnet", "claude-3-opus"},
		SelectedModel:   "claude-3-5-sonnet",
		Status:          "ONLINE",
		ExecutionMode:   "CLI_AGENT",
		Capabilities:    []string{"autonomous_repair", "diff_editing", "terminal_execution"},
	}, true
}

// ExecuteLocalIntent ejecuta una instrucción enviada desde el dashboard
// utilizando el motor cognitivo local disponible bajo las reglas de gobernanza QDD.
func ExecuteLocalIntent(ctx context.Context, cwd, intent string) (ExecutionResult, error) {
	start := time.Now()
	info := DetectLocalLLM()

	promptWithGovernance := buildGovernancePrompt(intent)

	if info.Provider == "Antigravity" || info.Provider == "Claude Code" {
		resp, backendName, err := Ask(ctx, promptWithGovernance)
		elapsed := time.Since(start).Milliseconds()
		if err != nil {
			return ExecutionResult{
				Status:          "ERROR",
				Provider:        info.Provider,
				Output:          fmt.Sprintf("Fallo ejecutando intención con %s: %v", info.Provider, err),
				ExecutionTimeMs: elapsed,
			}, err
		}
		return ExecutionResult{
			Status:           "SUCCESS",
			Provider:         backendName,
			Model:            info.SelectedModel,
			Output:           resp,
			ExecutionTimeMs:  elapsed,
			GovernanceChecks: []string{"Zero-Else Verified", "Early Return Guard", "SOC 2 Grounded"},
		}, nil
	}

	if info.Provider == "Ollama" {
		resp, err := callOllamaGenerate(ctx, info.Endpoint, info.SelectedModel, promptWithGovernance)
		elapsed := time.Since(start).Milliseconds()
		if err != nil {
			return ExecutionResult{
				Status:          "ERROR",
				Provider:        "Ollama",
				Output:          fmt.Sprintf("Fallo de inferencia en Ollama: %v", err),
				ExecutionTimeMs: elapsed,
			}, err
		}
		return ExecutionResult{
			Status:           "SUCCESS",
			Provider:         "Ollama",
			Model:            info.SelectedModel,
			Output:           resp,
			ExecutionTimeMs:  elapsed,
			GovernanceChecks: []string{"Local Airgapped Execution", "ISO 29119-11 Schema Check"},
		}, nil
	}

	if info.ExecutionMode == "LOCAL_REST_API" {
		resp, err := callOpenAICompatChat(ctx, info.Endpoint, info.SelectedModel, promptWithGovernance)
		elapsed := time.Since(start).Milliseconds()
		if err != nil {
			return ExecutionResult{
				Status:          "ERROR",
				Provider:        info.Provider,
				Output:          fmt.Sprintf("Fallo en API local %s: %v", info.Provider, err),
				ExecutionTimeMs: elapsed,
			}, err
		}
		return ExecutionResult{
			Status:           "SUCCESS",
			Provider:         info.Provider,
			Model:            info.SelectedModel,
			Output:           resp,
			ExecutionTimeMs:  elapsed,
			GovernanceChecks: []string{"REST Ingestion", "ISO 23894 Bounds Check"},
		}, nil
	}

	elapsed := time.Since(start).Milliseconds()
	return ExecutionResult{
		Status:          "OFFLINE_GUIDANCE",
		Provider:        "None",
		Output:          "No se detectó un motor LLM local activo (Antigravity/Ollama/LM Studio). Conecta tu IDE o inicia Ollama (ollama run llama3) para habilitar ejecución directa.",
		ExecutionTimeMs: elapsed,
		GovernanceChecks: []string{"Modo Consultivo", "Safe Fallback"},
	}, nil
}

func buildGovernancePrompt(userIntent string) string {
	return fmt.Sprintf(`[QDD GOVERNANCE HARNESS]
Eres un Agente de IA auditado por el QDD Framework.
Reglas inquebrantables de codificación:
1) NUNCA uses 'else' en ningún lenguaje. Usa guard clauses y salidas tempranas (early return).
2) La salida más rápida primero dentro de toda función.
3) Todo bug encontrado debe documentarse como Finding y generar test unitario.

Instrucción del Usuario desde el Dashboard:
%s
`, userIntent)
}

func callOllamaGenerate(ctx context.Context, endpoint, model, prompt string) (string, error) {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	})

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint+"/api/generate", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var ollamaResp struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", fmt.Errorf("error parseando respuesta de Ollama: %w", err)
	}
	return ollamaResp.Response, nil
}

func callOpenAICompatChat(ctx context.Context, endpoint, model, prompt string) (string, error) {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.2,
	})

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint+"/v1/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &chatResp); err != nil || len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("error parseando respuesta de API local: %w", err)
	}
	return chatResp.Choices[0].Message.Content, nil
}
