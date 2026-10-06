package domain

import (
	"fmt"
	"sort"

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
	sort.Strings(result)
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

func (r Registry) CheckCompatibility(name, versionRange string) error {
	pack, err := r.Load(name)
	if err != nil {
		return err
	}
	ok, err := pack.Supports(versionRange)
	if err != nil {
		return fmt.Errorf("domain pack %q compatibility: %w", name, err)
	}
	if !ok {
		return fmt.Errorf("domain pack %q version %s does not satisfy %q", name, pack.Version, versionRange)
	}
	return nil
}

func (r Registry) HasCompetency(domainName, id string) bool {
	pack, err := r.Load(domainName)
	if err != nil {
		return false
	}
	return pack.HasCompetency(id)
}

// NewRegistryFrom builds a registry over the given pack sources (domain name to
// competencies YAML), for composition roots that supply packs other than the
// built-in ones.
func NewRegistryFrom(sources map[string][]byte) Registry {
	loaders := make(map[string]func() ([]byte, error), len(sources))
	for name, data := range sources {
		data := append([]byte(nil), data...)
		loaders[name] = func() ([]byte, error) { return data, nil }
	}
	return Registry{loaders: loaders}
}
