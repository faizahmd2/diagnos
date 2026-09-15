package aiclient

import "github.com/faizahmd2/diagnos/internal/catalog"

type InitialAnalysisRequest struct {
	Context      string
	Hint         string
	Capabilities []Capability
}

type Capability struct {
	ID          string
	Area        string
	Description string
	Cost        string
}

type InitialAnalysisResponse struct {
	Investigations          []InvestigationSelection `json:"investigations"`
	UnsupportedObservations []string                 `json:"unsupported_observations,omitempty"`
}

type InvestigationSelection struct {
	CapabilityID                   string   `json:"capability_id"`
	Confidence                     float64  `json:"confidence"`
	Evidence                       []string `json:"evidence"`
	MetricObservations             []string `json:"metric_observations"`
	ConnectedTelemetryObservations []string `json:"connected_telemetry_observations"`
}

type ModelConfig struct {
	BaseURL   string `yaml:"base_url"`
	APIKeyEnv string `yaml:"api_key_env"`
}

type Model struct {
	Name   string
	Config ModelConfig
}

type FinalAnalysisRequest struct {
	Context        string
	Investigations []InvestigationSelection
	Unsupported    []string
	DiagnosticData []catalog.DiagnosticResult
	LogEvidence    string
}

type PlannedInvestigation struct {
	CapabilityID string
	Confidence   float64
	Evidence     []string
}
