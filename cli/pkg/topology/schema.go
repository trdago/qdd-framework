package topology

// TopologyNode represents any entity in the project ecosystem.
type TopologyNode struct {
	ID            string          `json:"id" yaml:"id"`
	Name          string          `json:"name" yaml:"name"`
	Type          string          `json:"type" yaml:"type"` // App, Module, Endpoint, Function, Operation
	Path          string          `json:"path" yaml:"path"`
	Certified     bool            `json:"certified" yaml:"certified"`
	RequiredCerts []string        `json:"required_certs" yaml:"required_certs"`
	MissingCerts  []string        `json:"missing_certs" yaml:"missing_certs"`
	Children      []*TopologyNode `json:"children" yaml:"children"`
}

// ProjectTopology represents the root of the project map
type ProjectTopology struct {
	Application *TopologyNode `json:"application" yaml:"application"`
	GlobalScore int           `json:"global_score" yaml:"global_score"`
}
