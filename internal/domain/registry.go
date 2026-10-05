package domain

import (
	"fmt"

	godomain "github.com/adams100111/agentic-learning-partner/domains/go"
	"go.yaml.in/yaml/v3"
)

type Registry struct {
	loaders map[string]func() ([]byte, error)
}

func NewRegistry() Registry {
	return Registry{
		loaders: map[string]func() ([]byte, error){
			"go": func() ([]byte, error) {
				return godomain.Files.ReadFile("competencies.yaml")
			},
		},
	}
}

func (r Registry) List() []string {
	result := make([]string, 0, len(r.loaders))
	for name := range r.loaders {
		result = append(result, name)
	}
	if len(result) == 1 {
		return result
	}
	// Keep output deterministic without introducing another public abstraction.
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j] < result[i] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}

func (r Registry) Load(name string) (Pack, error) {
	loader, ok := r.loaders[name]
	if !ok {
		return Pack{}, fmt.Errorf("unknown domain pack %q", name)
	}
	data, err := loader()
	if err != nil {
		return Pack{}, fmt.Errorf("read domain pack %q: %w", name, err)
	}

	var pack Pack
	if err := yaml.Unmarshal(data, &pack); err != nil {
		return Pack{}, fmt.Errorf("parse domain pack %q: %w", name, err)
	}
	if err := pack.Validate(); err != nil {
		return Pack{}, err
	}
	if pack.Domain != name {
		return Pack{}, fmt.Errorf("domain pack %q declares domain %q", name, pack.Domain)
	}
	return pack, nil
}
