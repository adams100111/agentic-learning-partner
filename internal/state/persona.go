package state

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Workspace-relative paths of the Learner Profile and the persona layers.
const (
	ProfilePath       = "profile/profile.yaml"
	GlobalPersonaPath = "personas/global.yaml"
)

// DomainPersonaPath is the workspace-relative path of a Domain Persona.
func DomainPersonaPath(domainName string) string {
	return "personas/domains/" + domainName + ".yaml"
}

// PersonaDocument is one persona or profile document of the learner
// workspace, schema-validated. Present is false when the file does not exist.
type PersonaDocument struct {
	Path    string
	Present bool
	Data    []byte
}

// DomainPersonaDocument is the Domain Persona document of one domain.
type DomainPersonaDocument struct {
	Domain string
	PersonaDocument
}

// PersonaDocuments are the Learner Profile, the Global Persona and the Domain
// Personas of the requested domains (in domain order).
type PersonaDocuments struct {
	Profile PersonaDocument
	Global  PersonaDocument
	Domains []DomainPersonaDocument
}

// PersonaDocuments reads the Learner Profile, the Global Persona and the
// Domain Persona of each domain. A missing document is reported, not an
// error; a present document must be schema-valid.
func (s Store) PersonaDocuments(domains []string) (PersonaDocuments, error) {
	profile, err := s.personaDocument(ProfilePath, "profile.schema.json")
	if err != nil {
		return PersonaDocuments{}, err
	}
	global, err := s.personaDocument(GlobalPersonaPath, "persona.schema.json")
	if err != nil {
		return PersonaDocuments{}, err
	}
	documents := PersonaDocuments{Profile: profile, Global: global, Domains: []DomainPersonaDocument{}}
	sorted := append([]string{}, domains...)
	sort.Strings(sorted)
	for i, domainName := range sorted {
		if i > 0 && sorted[i-1] == domainName {
			continue
		}
		document, err := s.personaDocument(DomainPersonaPath(domainName), "persona.schema.json")
		if err != nil {
			return PersonaDocuments{}, err
		}
		documents.Domains = append(documents.Domains, DomainPersonaDocument{Domain: domainName, PersonaDocument: document})
	}
	return documents, nil
}

func (s Store) personaDocument(path, schema string) (PersonaDocument, error) {
	data, err := os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(path)))
	if os.IsNotExist(err) {
		return PersonaDocument{Path: path}, nil
	}
	if err != nil {
		return PersonaDocument{}, fmt.Errorf("read %s: %w", path, err)
	}
	if err := s.validate(schema, path, data); err != nil {
		return PersonaDocument{}, err
	}
	return PersonaDocument{Path: path, Present: true, Data: data}, nil
}
