package pylearn

import (
	"github.com/adams100111/agentic-learning-partner/internal/domain"
)

// Adapter is the PyLearn Platform Adapter.
type Adapter struct {
	Domains domain.Registry
}

func NewAdapter() Adapter {
	return Adapter{Domains: domain.NewRegistry()}
}
