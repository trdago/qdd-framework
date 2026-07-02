package models

// ExecutionPlan es el contrato YAML/JSON independiente del proveedor IA.
type ExecutionPlan struct {
	Goal          string   `json:"goal" yaml:"goal"`
	Intent        string   `json:"intent" yaml:"intent"`
	Strategy      []string `json:"strategy" yaml:"strategy"`
	Artifacts     []string `json:"artifacts" yaml:"artifacts"`
	Quality       []string `json:"quality" yaml:"quality"`
	Compatibility string   `json:"compatibility" yaml:"compatibility"`
}
