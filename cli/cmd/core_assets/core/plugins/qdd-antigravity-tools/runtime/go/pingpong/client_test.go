package pingpong

import (
	"context"
	"testing"
	"time"
)

func TestClient_PingSuccess(t *testing.T) {
	client := NewClient(ClientConfig{
		SystemInstructions: "Test agent",
		RunCommand: func(name string, arg ...string) ([]byte, error) {
			return []byte("Hello from mock agentapi!"), nil
		},
	})

	session := client.Session("sess_unit_test_1")
	var tokens []string

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pong, err := session.Ping(ctx, "Audit repo", WithOnToken(func(tok string) {
		tokens = append(tokens, tok)
	}))

	if err != nil {
		t.Fatalf("Error inesperado en Ping: %v", err)
	}

	if !pong.IsSuccess() {
		t.Fatalf("Se esperaba status SUCCESS, se obtuvo: %s", pong.Status)
	}

	if pong.SessionID != "sess_unit_test_1" {
		t.Fatalf("SessionID incorrecto: %s", pong.SessionID)
	}

	if pong.TurnID != 1 {
		t.Fatalf("TurnID incorrecto: %d", pong.TurnID)
	}

	if len(tokens) == 0 {
		t.Fatalf("No se recibieron tokens en streaming")
	}
}

func TestClient_EmptyPromptError(t *testing.T) {
	client := NewClient(ClientConfig{})
	session := client.Session("")

	_, err := session.Ping(context.Background(), "")
	if err == nil {
		t.Fatalf("Se esperaba error con prompt vacío pero retornó nil")
	}
}

func TestClient_ExtractJSONBlock(t *testing.T) {
	delimited := "Pensando...\n[VERDICT_START]\n{\"status\":\"PASS\",\"score\":100}\n[VERDICT_END]\nFin."
	res := ExtractJSONBlock(delimited)

	if res["status"] != "PASS" {
		t.Fatalf("Se esperaba status PASS, se obtuvo: %v", res["status"])
	}
	if res["score"] != float64(100) {
		t.Fatalf("Se esperaba score 100, se obtuvo: %v", res["score"])
	}

	markdown := "```json\n{\"alive\":true}\n```"
	resMd := ExtractJSONBlock(markdown)
	if resMd["alive"] != true {
		t.Fatalf("Se esperaba alive true, se obtuvo: %v", resMd["alive"])
	}
}

func TestToolBridge_RegisterAndExecute(t *testing.T) {
	tb := NewToolBridge()
	err := tb.RegisterTool("sum", "Suma dos números", map[string]interface{}{}, func(args map[string]interface{}) (interface{}, error) {
		a := args["a"].(float64)
		b := args["b"].(float64)
		return a + b, nil
	})

	if err != nil {
		t.Fatalf("Error registrando tool: %v", err)
	}

	tools := tb.ListTools()
	if len(tools) != 1 {
		t.Fatalf("Se esperaba 1 tool, se obtuvieron: %d", len(tools))
	}

	res, errExec := tb.ExecuteTool("sum", map[string]interface{}{"a": float64(15), "b": float64(25)})
	if errExec != nil {
		t.Fatalf("Error ejecutando tool: %v", errExec)
	}

	if res != float64(40) {
		t.Fatalf("Se esperaba 40, se obtuvo: %v", res)
	}
}

func TestGetAgentAPIPath(t *testing.T) {
	// Should not panic
	path := GetAgentAPIPath()
	t.Logf("AgentAPI path detected: %s", path)
}
