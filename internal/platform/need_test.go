package platform

import (
	"testing"

	"github.com/adams100111/agentic-learning-partner/internal/state"
)

// The need signal is part of the projection read, so sequencing and mode
// proposals cannot disagree about a competency.
func TestLearnerReadExposesNeedForEveryLevel(t *testing.T) {
	cases := []struct {
		name       string
		competency state.ProjectedCompetency
		want       Need
	}{
		{"unknown", state.ProjectedCompetency{Level: "unknown"}, NeedUrgent},
		{"rusty", state.ProjectedCompetency{Level: "rusty"}, NeedUrgent},
		{"functional", state.ProjectedCompetency{Level: "functional"}, NeedSettled},
		{"strong", state.ProjectedCompetency{Level: "strong", Confidence: "high"}, NeedSettled},
		{"strong, low confidence", state.ProjectedCompetency{Level: "strong", Confidence: "low"}, NeedUnconfirmed},
		{"strong, stale", state.ProjectedCompetency{Level: "strong", Confidence: "high", NeedsReassessment: true}, NeedUnconfirmed},
		{"functional, stale", state.ProjectedCompetency{Level: "functional", Confidence: "high", NeedsReassessment: true}, NeedUnconfirmed},
	}
	for _, test := range cases {
		test.competency.Domain, test.competency.ID = "go", "go.x"
		view := newLearnerView(state.Projection{Competencies: []state.ProjectedCompetency{test.competency}})
		if got := view.read("go", "go.x").Need; got != test.want {
			t.Errorf("%s: need = %d, want %d", test.name, got, test.want)
		}
	}
	if got := newLearnerView(state.Projection{}).read("go", "go.x").Need; got != NeedUnconfirmed {
		t.Errorf("unassessed need = %d, want %d", got, NeedUnconfirmed)
	}
}
