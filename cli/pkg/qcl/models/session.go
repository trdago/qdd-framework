package models

type IntentModel struct {
	Type        string
	Objectives  []string
	Constraints []string
}

type ProjectContext struct {
	HasState       bool
	Plugins        []string
	KnownFindings  int
}

type Risk struct {
	Type        string
	Description string
	Severity    string
}

type StrategyPlan struct {
	Steps           []string
	TargetArtifacts []string
}

// ApprovalRequest se devuelve a la CLI cuando se requiere intervención humana.
type ApprovalRequest struct {
	Reason string
	Risks  []Risk
}

type ClarificationRequest struct {
	Message string
	Options []string
}

// CognitiveSession representa el estado del razonamiento a medida que fluye por el pipeline.
type CognitiveSession struct {
	RawInput             string
	Intent               *IntentModel
	Context              *ProjectContext
	Risks                []Risk
	Strategy             *StrategyPlan
	ExecutionPlan        *ExecutionPlan
	ApprovalRequest      *ApprovalRequest
	ClarificationRequest *ClarificationRequest
}
