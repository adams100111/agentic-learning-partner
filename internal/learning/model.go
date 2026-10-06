package learning

type ReviewQueue struct {
	SchemaVersion int          `yaml:"schemaVersion" json:"schemaVersion"`
	AsOf          string       `yaml:"asOf" json:"asOf"`
	Items         []ReviewItem `yaml:"items" json:"items"`
}

type ReviewItem struct {
	Domain       string   `yaml:"domain" json:"domain"`
	Competency   string   `yaml:"competency" json:"competency"`
	Priority     int      `yaml:"priority" json:"priority"`
	Reasons      []string `yaml:"reasons" json:"reasons"`
	LastVerified string   `yaml:"lastVerified,omitempty" json:"lastVerified,omitempty"`
}

type Plan struct {
	SchemaVersion int        `yaml:"schemaVersion" json:"schemaVersion"`
	Domain        string     `yaml:"domain" json:"domain"`
	AsOf          string     `yaml:"asOf" json:"asOf"`
	Items         []PlanItem `yaml:"items" json:"items"`
}

type PlanItem struct {
	Competency string `yaml:"competency" json:"competency"`
	Mode       string `yaml:"mode" json:"mode"`
	Reason     string `yaml:"reason" json:"reason"`
	Priority   int    `yaml:"priority" json:"priority"`
}

type Diagnostic struct {
	SchemaVersion int                  `yaml:"schemaVersion" json:"schemaVersion"`
	Domain        string               `yaml:"domain" json:"domain"`
	Activities    []DiagnosticActivity `yaml:"activities" json:"activities"`
}

type DiagnosticActivity struct {
	ID           string   `yaml:"id" json:"id"`
	Kind         string   `yaml:"kind" json:"kind"`
	Competencies []string `yaml:"competencies" json:"competencies"`
	Prompt       string   `yaml:"prompt" json:"prompt"`
}
