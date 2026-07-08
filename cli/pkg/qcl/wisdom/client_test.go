package wisdom

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFetchRepairStrategy_OfflineFallback(t *testing.T) {
	tempDir := t.TempDir()

	// Arrange: Create a fake offline server (returns 500)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Offline", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(tempDir)
	client.baseURL = server.URL

	// Pre-seed the cache
	cacheDir := filepath.Join(tempDir, ".qdd", "cache", "wisdom")
	os.MkdirAll(cacheDir, 0755)
	os.WriteFile(filepath.Join(cacheDir, "repair_testcomp.yaml"), []byte("component: testcomp\naction: update\n"), 0644)

	// Act
	strategy, err := client.FetchRepairStrategy(context.Background(), "testcomp")

	// Assert
	if err != nil {
		t.Fatalf("Expected fallback to succeed, got error: %v", err)
	}
	if strategy == nil || strategy.Component != "testcomp" {
		t.Fatalf("Failed to parse fallback strategy, got: %+v", strategy)
	}
}

func TestFetchRulesManifest_Success(t *testing.T) {
	tempDir := t.TempDir()

	// Arrange: Server returns valid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"version":"1.0", "rules":["RULE-01"]}`))
	}))
	defer server.Close()

	client := NewClient(tempDir)
	client.baseURL = server.URL

	// Act
	manifest, err := client.FetchRulesManifest(context.Background())

	// Assert
	assertManifestSuccess(t, manifest, err)
}

func assertManifestSuccess(t *testing.T, manifest *RemoteRulesManifest, err error) {
	if err != nil {
		t.Fatalf("Expected fetch to succeed, got error: %v", err)
	}
	if manifest == nil {
		t.Fatalf("Expected manifest, got nil")
	}
	assertManifestData(t, manifest)
}

func assertManifestData(t *testing.T, manifest *RemoteRulesManifest) {
	if manifest.Version != "1.0" {
		t.Fatalf("Expected version 1.0, got: %+v", manifest)
	}
	if len(manifest.Rules) != 1 {
		t.Fatalf("Expected 1 rule, got: %+v", manifest)
	}
	if manifest.Rules[0] != "RULE-01" {
		t.Fatalf("Expected RULE-01, got: %+v", manifest)
	}
}
