package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Tendency enum for the trend
type Tendency string

const (
	TendencyImproving Tendency = "Mejorando"
	TendencyWorsening Tendency = "Empeorando"
	TendencyStable    Tendency = "Estable"
)

// Certificate represents a snapshot of the framework's quality
type Certificate struct {
	Timestamp      string     `json:"timestamp"`
	Score          int        `json:"score"`
	TotalViolations int        `json:"total_violations"`
	Tendency       Tendency   `json:"tendency"`
}

// GenerateCertificate creates a certificate based on violations and saves it to history.
func GenerateCertificate(cwd string, violations []Violation) (*Certificate, error) {
	score := 100 - (len(violations) * 2)
	if score < 0 {
		score = 0
	}

	cert := &Certificate{
		Timestamp:      time.Now().Format(time.RFC3339),
		Score:          score,
		TotalViolations: len(violations),
		Tendency:       TendencyStable,
	}

	historyPath := filepath.Join(cwd, ".qdd", "project", "metrics", "certificate_history.json")
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(historyPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("error creating metrics directory: %v", err)
	}

	var history []Certificate
	
	// Read previous history
	data, err := os.ReadFile(historyPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error reading certificate history: %v", err)
		}
	}
	
	if err == nil {
		if parseErr := json.Unmarshal(data, &history); parseErr != nil {
			return nil, fmt.Errorf("error parsing certificate history: %v", parseErr)
		}
	}

	// Calculate tendency based on the last certificate
	if len(history) > 0 {
		lastCert := history[len(history)-1]
		if cert.Score > lastCert.Score {
			cert.Tendency = TendencyImproving
		}
		if cert.Score < lastCert.Score {
			cert.Tendency = TendencyWorsening
		}
	}

	// Append to history and save
	history = append(history, *cert)
	
	outData, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshaling certificate history: %v", err)
	}

	if err := os.WriteFile(historyPath, outData, 0644); err != nil {
		return nil, fmt.Errorf("error writing certificate history: %v", err)
	}

	return cert, nil
}
