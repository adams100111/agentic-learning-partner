package context

type Bundle struct {
	SchemaVersion   int               `yaml:"schemaVersion" json:"schemaVersion"`
	Task            string            `yaml:"task" json:"task"`
	Domain          *string           `yaml:"domain" json:"domain"`
	Learner         map[string]any    `yaml:"learner" json:"learner"`
	Focus           map[string]any    `yaml:"focus" json:"focus"`
	Persona         map[string]any    `yaml:"persona,omitempty" json:"persona,omitempty"`
	Competencies    map[string]any    `yaml:"competencies,omitempty" json:"competencies,omitempty"`
	Reinforcement   []string          `yaml:"reinforcement,omitempty" json:"reinforcement,omitempty"`
	Project         map[string]any    `yaml:"project,omitempty" json:"project,omitempty"`
	IncludedSources []SourceSelection `yaml:"includedSources" json:"includedSources"`
	OmittedSources  []SourceSelection `yaml:"omittedSources,omitempty" json:"omittedSources,omitempty"`
	EstimatedTokens *int              `yaml:"estimatedTokens" json:"estimatedTokens"`
}

type SourceSelection struct {
	Ref    string `yaml:"ref" json:"ref"`
	Reason string `yaml:"reason" json:"reason"`
}

type Request struct {
	Task       string
	Domain     string
	Competency string
	Inspect    bool
}
