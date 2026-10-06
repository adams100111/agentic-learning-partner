package state

type Evidence struct {
	SchemaVersion int            `yaml:"schemaVersion"`
	ID            string         `yaml:"id"`
	RecordedAt    string         `yaml:"recordedAt"`
	Domain        string         `yaml:"domain"`
	Competencies  []string       `yaml:"competencies"`
	Type          string         `yaml:"type"`
	Source        EvidenceSource `yaml:"source"`
	Observation   string         `yaml:"observation"`
	Result        string         `yaml:"result,omitempty"`
	Strength      string         `yaml:"strength"`
	FailureClass  string         `yaml:"failureClassification,omitempty"`
	Supports      []string       `yaml:"supports,omitempty"`
	Contradicts   []string       `yaml:"contradicts,omitempty"`
	Supersedes    []string       `yaml:"supersedes,omitempty"`
	Metadata      map[string]any `yaml:"metadata,omitempty"`
}

type EvidenceSource struct {
	Kind      string `yaml:"kind"`
	Ref       string `yaml:"ref,omitempty"`
	Commit    string `yaml:"commit,omitempty"`
	ContentID string `yaml:"contentId,omitempty"`
}

type Assessment struct {
	SchemaVersion     int       `yaml:"schemaVersion"`
	ID                string    `yaml:"id"`
	RecordedAt        string    `yaml:"recordedAt"`
	Domain            string    `yaml:"domain"`
	Competency        string    `yaml:"competency"`
	Evidence          []string  `yaml:"evidence"`
	Rubric            RubricRef `yaml:"rubric"`
	Assessor          Assessor  `yaml:"assessor"`
	Judgment          Judgment  `yaml:"judgment"`
	Confidence        string    `yaml:"confidence"`
	Rationale         string    `yaml:"rationale"`
	Status            string    `yaml:"status"`
	Supersedes        []string  `yaml:"supersedes,omitempty"`
	WorkspaceRevision string    `yaml:"workspaceRevision,omitempty"`
}

type RubricRef struct {
	ID                string `yaml:"id"`
	Version           string `yaml:"version"`
	DomainPackVersion string `yaml:"domainPackVersion,omitempty"`
}

type Assessor struct {
	Type    string `yaml:"type"`
	ID      string `yaml:"id"`
	Model   string `yaml:"model,omitempty"`
	Harness string `yaml:"harness,omitempty"`
}

type Judgment struct {
	Level     string   `yaml:"level"`
	Direction string   `yaml:"direction,omitempty"`
	Gaps      []string `yaml:"gaps,omitempty"`
}

type Projection struct {
	SchemaVersion int                   `yaml:"schemaVersion"`
	Competencies  []ProjectedCompetency `yaml:"competencies"`
}

type ProjectedCompetency struct {
	ID                string   `yaml:"id"`
	Domain            string   `yaml:"domain"`
	Level             string   `yaml:"level"`
	Confidence        string   `yaml:"confidence"`
	AssessmentID      string   `yaml:"assessmentId"`
	LastVerified      string   `yaml:"lastVerified"`
	NeedsReassessment bool     `yaml:"needsReassessment,omitempty"`
	Gaps              []string `yaml:"gaps,omitempty"`
}
