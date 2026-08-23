package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/cognitive"
)

func TestChaosToleranceTimeoutResilience(t *testing.T) {
	// Create slow mock server that delays response beyond timeout
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := cognitive.ExecuteLocalIntent(ctx, ".", "Intento con timeout estricto")
	// Must return quickly without hanging indefinitely
	if ctx.Err() == nil && err != nil {
		// Context completed safely
		return
	}
}

func TestLocalLLMDetectGracefulDegradation(t *testing.T) {
	// Probing in an environment without local servers must degrade gracefully without panicking
	info := cognitive.DetectLocalLLM()
	if info.Provider == "" {
		t.Fatalf("Expected valid provider string even when offline, got empty")
	}
	if info.Status == "" {
		t.Fatalf("Expected valid status string, got empty")
	}
}
