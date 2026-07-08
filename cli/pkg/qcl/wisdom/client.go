package wisdom

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultRegistryURL = "https://raw.githubusercontent.com/trdago/qdd-framework/main/registry"
	cacheTTL           = 24 * time.Hour
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	cacheDir   string
}

func NewClient(cwd string) *Client {
	cacheDir := filepath.Join(cwd, ".qdd", "cache", "wisdom")
	os.MkdirAll(cacheDir, 0755)

	return &Client{
		baseURL: defaultRegistryURL,
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
		cacheDir: cacheDir,
	}
}

type RepairStrategy struct {
	Component string   `yaml:"component"`
	Action    string   `yaml:"action"` // "create", "update"
	Files     []string `yaml:"files"`
	Source    string   `yaml:"source"`
}

func (c *Client) FetchRepairStrategy(ctx context.Context, componentName string) (*RepairStrategy, error) {
	cachePath := filepath.Join(c.cacheDir, fmt.Sprintf("repair_%s.yaml", componentName))
	
	if c.isCacheValid(cachePath) {
		return c.loadLocalRepair(cachePath)
	}

	return c.fetchAndCacheRepairStrategy(ctx, componentName, cachePath)
}

func (c *Client) isCacheValid(cachePath string) bool {
	info, err := os.Stat(cachePath)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) < cacheTTL
}

func (c *Client) fetchRemoteData(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) fetchAndCacheRepairStrategy(ctx context.Context, componentName, cachePath string) (*RepairStrategy, error) {
	url := fmt.Sprintf("%s/repairs/%s.yaml", c.baseURL, componentName)
	body, err := c.fetchRemoteData(ctx, url)
	if err != nil {
		return c.loadLocalRepair(cachePath)
	}

	_ = os.WriteFile(cachePath, body, 0644)

	var strategy RepairStrategy
	if err := yaml.Unmarshal(body, &strategy); err != nil {
		return nil, err
	}
	return &strategy, nil
}

func (c *Client) loadLocalRepair(path string) (*RepairStrategy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("no cache and remote failed: %w", err)
	}
	var strategy RepairStrategy
	if err := yaml.Unmarshal(data, &strategy); err != nil {
		return nil, err
	}
	return &strategy, nil
}

type RemoteRulesManifest struct {
	Version string   `json:"version"`
	Rules   []string `json:"rules"`
}

func (c *Client) FetchRulesManifest(ctx context.Context) (*RemoteRulesManifest, error) {
	cachePath := filepath.Join(c.cacheDir, "manifest.json")
	
	if c.isCacheValid(cachePath) {
		return c.loadLocalManifest(cachePath)
	}

	return c.fetchAndCacheManifest(ctx, cachePath)
}

func (c *Client) fetchAndCacheManifest(ctx context.Context, cachePath string) (*RemoteRulesManifest, error) {
	url := fmt.Sprintf("%s/manifest.json", c.baseURL)
	body, err := c.fetchRemoteData(ctx, url)
	if err != nil {
		return c.loadLocalManifest(cachePath)
	}

	_ = os.WriteFile(cachePath, body, 0644)

	var manifest RemoteRulesManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func (c *Client) loadLocalManifest(path string) (*RemoteRulesManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("no cache and remote failed")
	}
	var manifest RemoteRulesManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}
