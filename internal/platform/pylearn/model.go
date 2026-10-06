package pylearn

import "github.com/adams100111/agentic-learning-partner/internal/state"

type Export struct {
	SchemaVersion  int              `yaml:"schemaVersion" json:"schemaVersion"`
	ExportedAt     string           `yaml:"exportedAt" json:"exportedAt"`
	Learner        map[string]any   `yaml:"learner,omitempty" json:"learner,omitempty"`
	Progress       []Progress       `yaml:"progress,omitempty" json:"progress,omitempty"`
	Attempts       []Attempt        `yaml:"attempts,omitempty" json:"attempts,omitempty"`
	ConceptMastery []ConceptMastery `yaml:"conceptMastery,omitempty" json:"conceptMastery,omitempty"`
	QuizAnswers    []QuizAnswer     `yaml:"quizAnswers,omitempty" json:"quizAnswers,omitempty"`
	Reflections    []Reflection     `yaml:"reflections,omitempty" json:"reflections,omitempty"`
	Bookmarks      []Bookmark       `yaml:"bookmarks,omitempty" json:"bookmarks,omitempty"`
}

type Progress struct {
	ID        string `yaml:"id" json:"id"`
	ContentID string `yaml:"contentId" json:"contentId"`
	Status    string `yaml:"status" json:"status"`
	UpdatedAt string `yaml:"updatedAt,omitempty" json:"updatedAt,omitempty"`
}

type Attempt struct {
	ID        string `yaml:"id" json:"id"`
	ContentID string `yaml:"contentId" json:"contentId"`
	Passed    bool   `yaml:"passed" json:"passed"`
	Code      string `yaml:"code,omitempty" json:"code,omitempty"`
	Failure   string `yaml:"failure,omitempty" json:"failure,omitempty"`
	Stdout    string `yaml:"stdout,omitempty" json:"stdout,omitempty"`
	CreatedAt string `yaml:"createdAt,omitempty" json:"createdAt,omitempty"`
}

type ConceptMastery struct {
	ID        string `yaml:"id" json:"id"`
	ContentID string `yaml:"contentId" json:"contentId"`
	Attempts  int    `yaml:"attempts,omitempty" json:"attempts,omitempty"`
	Passes    int    `yaml:"passes,omitempty" json:"passes,omitempty"`
	Failures  int    `yaml:"failures,omitempty" json:"failures,omitempty"`
	UpdatedAt string `yaml:"updatedAt,omitempty" json:"updatedAt,omitempty"`
}

type QuizAnswer struct {
	ID        string `yaml:"id" json:"id"`
	ContentID string `yaml:"contentId" json:"contentId"`
	Correct   bool   `yaml:"correct" json:"correct"`
	CreatedAt string `yaml:"createdAt,omitempty" json:"createdAt,omitempty"`
}

type Reflection struct {
	ID        string `yaml:"id" json:"id"`
	ContentID string `yaml:"contentId" json:"contentId"`
	Text      string `yaml:"text" json:"text"`
	CreatedAt string `yaml:"createdAt,omitempty" json:"createdAt,omitempty"`
}

type Bookmark struct {
	ID        string `yaml:"id" json:"id"`
	ContentID string `yaml:"contentId" json:"contentId"`
}

type DerivedSignal struct {
	Kind      string
	ContentID string
	Ref       string
	Summary   string
}

type ProfileConflict struct {
	Field    string
	Platform any
	Current  any
}

type Result struct {
	Evidence         []NormalizedEvidence
	DerivedSignals   []DerivedSignal
	ProfileConflicts []ProfileConflict
}

type NormalizedEvidence struct {
	StableKey string
	Record    state.Evidence
}
