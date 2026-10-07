package platform

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/adams100111/agentic-learning-partner/internal/domain"
	"github.com/adams100111/agentic-learning-partner/internal/state"
	"github.com/adams100111/agentic-learning-partner/internal/workspace"
)

// Analogy bases: how a competency's analogy sources were chosen.
const (
	AnalogySemanticOverride = "semantic-override"
	AnalogyDefaultPriority  = "default-priority"
	AnalogyNone             = "none"
)

// defaultPace is the teaching pace when no persona layer or profile
// preference sets one.
const defaultPace = "standard"

var personaPaces = map[string]bool{"gentle": true, "standard": true, "senior-dense": true}

// TeachingShape is the persona-derived, learner-free teaching shape of a unit
// (ADR-0007, ADR-0060, Q26): how its content should be pitched. It names
// analogy stacks and teaching constraints only, never the learner's identity,
// experience levels or risks.
type TeachingShape struct {
	Pace          string              `json:"pace"`
	ActivityTypes []string            `json:"activityTypes"`
	Avoid         []string            `json:"avoid"`
	Emphasis      []string            `json:"emphasis"`
	Feedback      []string            `json:"feedback"`
	Analogies     []CompetencyAnalogy `json:"analogies"`
}

// CompetencyAnalogy names the stacks to draw analogies from for one
// competency, in preference order, and the persona concepts that chose them.
type CompetencyAnalogy struct {
	Domain     string   `json:"domain"`
	Competency string   `json:"competency"`
	Basis      string   `json:"basis"`
	Concepts   []string `json:"concepts"`
	Sources    []string `json:"sources"`
}

// PersonaBasis is the private persona basis of a unit's teaching shape: the
// learner's risks from the unit's personas, the persona's reasons for each analogy override,
// and the learner's experience in each analogy source. It never leaves the
// learner workspace.
type PersonaBasis struct {
	Risks            []UnitRisk         `json:"risks"`
	AnalogyReasons   []AnalogyReason    `json:"analogyReasons"`
	SourceExperience []SourceExperience `json:"sourceExperience"`
}

// UnitRisk is one learner risk from the unit's Global or Domain Persona, with
// the unit concepts it names (none for a general risk).
type UnitRisk struct {
	Risk     string   `json:"risk"`
	Concepts []string `json:"concepts"`
}

// AnalogyReason is a persona's stated reason for a semantic override applied
// to a competency.
type AnalogyReason struct {
	Domain     string `json:"domain"`
	Competency string `json:"competency"`
	Concept    string `json:"concept"`
	Layer      string `json:"layer"`
	Reason     string `json:"reason"`
}

// SourceExperience is the learner's profile experience behind one analogy
// source: the stack it belongs to, its level and relative rank.
type SourceExperience struct {
	Source       string `json:"source"`
	Stack        string `json:"stack"`
	Level        string `json:"level"`
	RelativeRank int    `json:"relativeRank,omitempty"`
}

// PersonaSource identifies one persona or profile document by path and
// content hash.
type PersonaSource struct {
	Path        string `json:"path"`
	ContentHash string `json:"contentHash"`
}

// DomainPersonaSource identifies the Domain Persona of one domain.
type DomainPersonaSource struct {
	Domain string `json:"domain"`
	PersonaSource
}

// PersonaProvenance records the persona and profile documents a teaching
// shape was derived from; null or absent documents were missing.
type PersonaProvenance struct {
	Profile *PersonaSource        `json:"profile"`
	Global  *PersonaSource        `json:"global"`
	Domains []DomainPersonaSource `json:"domains"`
}

// PersonaReport tells the learner which persona documents shaped teaching and
// which documented defaults stood in for missing ones.
type PersonaReport struct {
	Documents []PersonaDocumentStatus `json:"documents"`
	Defaults  []PersonaDefault        `json:"defaults"`
}

// PersonaDocumentStatus reports one persona or profile document.
type PersonaDocumentStatus struct {
	Path        string `json:"path"`
	Domain      string `json:"domain,omitempty"`
	Present     bool   `json:"present"`
	ContentHash string `json:"contentHash,omitempty"`
}

// PersonaDefault is a documented default applied for one teaching-shape
// field of a domain because no persona layer or profile sets it.
type PersonaDefault struct {
	Domain string `json:"domain"`
	Field  string `json:"field"`
	Value  string `json:"value"`
	Reason string `json:"reason"`
}

// TeachingPersona is the parsed Learner Profile, Global Persona and Domain
// Personas that shape unit teaching.
type TeachingPersona struct {
	Provenance PersonaProvenance
	Report     PersonaReport
	stacks     map[string]SourceExperience
	ranked     []string
	effective  map[string]effectivePersona
}

type profileDocument struct {
	Experience map[string]struct {
		Level        string            `json:"level"`
		RelativeRank int               `json:"relativeRank"`
		Frameworks   map[string]string `json:"frameworks"`
	} `json:"experience"`
	Preferences map[string]any `json:"preferences"`
}

type personaDocument struct {
	Teaching struct {
		Pace                   string   `json:"pace"`
		PreferredActivityTypes []string `json:"preferredActivityTypes"`
		Avoid                  []string `json:"avoid"`
		Emphasis               []string `json:"emphasis"`
		Feedback               []string `json:"feedback"`
	} `json:"teaching"`
	AnalogyPolicy struct {
		DefaultPriority   []string `json:"defaultPriority"`
		SemanticOverrides map[string]struct {
			Prefer []string `json:"prefer"`
			Reason string   `json:"reason"`
		} `json:"semanticOverrides"`
	} `json:"analogyPolicy"`
	Risks []string `json:"risks"`
}

type analogyOverride struct {
	concept, layer, reason string
	prefer                 []string
}

// effectivePersona is the Domain Persona layered over the Global Persona and
// the profile for one domain.
type effectivePersona struct {
	pace                                     string
	activityTypes, avoid, emphasis, feedback []string
	defaultPriority                          []string
	overrides                                []analogyOverride
	risks                                    []string
}

// ReadTeachingPersona parses the persona documents and resolves the effective
// persona of each domain. Missing documents yield documented defaults, which
// the report lists.
func ReadTeachingPersona(documents state.PersonaDocuments) (TeachingPersona, error) {
	persona := TeachingPersona{
		Provenance: PersonaProvenance{Domains: []DomainPersonaSource{}},
		Report:     PersonaReport{Documents: []PersonaDocumentStatus{}, Defaults: []PersonaDefault{}},
		stacks:     map[string]SourceExperience{},
		effective:  map[string]effectivePersona{},
	}
	var profile *profileDocument
	if documents.Profile.Present {
		profile = &profileDocument{}
		if err := decodePersonaDocument(documents.Profile, profile); err != nil {
			return TeachingPersona{}, err
		}
		persona.Provenance.Profile = personaSource(documents.Profile)
	}
	persona.Report.Documents = append(persona.Report.Documents, documentStatus(documents.Profile, ""))
	var global *personaDocument
	if documents.Global.Present {
		global = &personaDocument{}
		if err := decodePersonaDocument(documents.Global, global); err != nil {
			return TeachingPersona{}, err
		}
		persona.Provenance.Global = personaSource(documents.Global)
	}
	persona.Report.Documents = append(persona.Report.Documents, documentStatus(documents.Global, ""))

	if profile != nil {
		persona.indexStacks(*profile)
	}
	for _, document := range documents.Domains {
		var layer *personaDocument
		if document.Present {
			layer = &personaDocument{}
			if err := decodePersonaDocument(document.PersonaDocument, layer); err != nil {
				return TeachingPersona{}, err
			}
			persona.Provenance.Domains = append(persona.Provenance.Domains, DomainPersonaSource{Domain: document.Domain, PersonaSource: *personaSource(document.PersonaDocument)})
		}
		persona.Report.Documents = append(persona.Report.Documents, documentStatus(document.PersonaDocument, document.Domain))
		persona.effective[document.Domain] = persona.resolve(document.Domain, profile, global, layer)
	}
	sort.SliceStable(persona.Report.Defaults, func(i, j int) bool {
		left, right := persona.Report.Defaults[i], persona.Report.Defaults[j]
		if left.Domain != right.Domain {
			return left.Domain < right.Domain
		}
		return left.Field < right.Field
	})
	return persona, nil
}

func decodePersonaDocument(document state.PersonaDocument, into any) error {
	value, err := workspace.DecodeDocument(document.Path, document.Data)
	if err != nil {
		return fmt.Errorf("read %s: %w", document.Path, err)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("read %s: %w", document.Path, err)
	}
	if err := json.Unmarshal(encoded, into); err != nil {
		return fmt.Errorf("read %s: %w", document.Path, err)
	}
	return nil
}

func personaSource(document state.PersonaDocument) *PersonaSource {
	sum := sha256.Sum256(document.Data)
	return &PersonaSource{Path: document.Path, ContentHash: "sha256:" + hex.EncodeToString(sum[:])}
}

func documentStatus(document state.PersonaDocument, domainName string) PersonaDocumentStatus {
	status := PersonaDocumentStatus{Path: document.Path, Domain: domainName, Present: document.Present}
	if document.Present {
		status.ContentHash = personaSource(document).ContentHash
	}
	return status
}

// indexStacks indexes the profile's stacks and frameworks as analogy sources,
// and ranks the stacks by relative rank.
func (p *TeachingPersona) indexStacks(profile profileDocument) {
	type rankedStack struct {
		name string
		rank int
	}
	var ranked []rankedStack
	for name, experience := range profile.Experience {
		stack := strings.ToLower(name)
		p.stacks[stack] = SourceExperience{Source: stack, Stack: stack, Level: experience.Level, RelativeRank: experience.RelativeRank}
		for framework, level := range experience.Frameworks {
			key := strings.ToLower(framework)
			if _, isStack := profile.Experience[framework]; !isStack {
				p.stacks[key] = SourceExperience{Source: key, Stack: stack, Level: level, RelativeRank: experience.RelativeRank}
			}
		}
		rank := experience.RelativeRank
		if rank == 0 {
			rank = int(^uint(0) >> 1)
		}
		ranked = append(ranked, rankedStack{name: stack, rank: rank})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].rank != ranked[j].rank {
			return ranked[i].rank < ranked[j].rank
		}
		return ranked[i].name < ranked[j].name
	})
	for _, stack := range ranked {
		p.ranked = append(p.ranked, stack.name)
	}
}

// resolve layers the Domain Persona over the Global Persona and the profile
// for one domain, recording each documented default it falls back to.
func (p *TeachingPersona) resolve(domainName string, profile *profileDocument, global, layer *personaDocument) effectivePersona {
	defaulted := func(field, value, reason string) {
		p.Report.Defaults = append(p.Report.Defaults, PersonaDefault{Domain: domainName, Field: field, Value: value, Reason: reason})
	}
	var effective effectivePersona
	layers := []*personaDocument{}
	for _, candidate := range []*personaDocument{global, layer} {
		if candidate != nil {
			layers = append(layers, candidate)
		}
	}

	// Pace: the most specific layer that sets it, then the profile.
	for _, candidate := range layers {
		if candidate.Teaching.Pace != "" {
			effective.pace = candidate.Teaching.Pace
		}
	}
	if effective.pace == "" && profile != nil {
		if pace, ok := profile.Preferences["teachingPace"].(string); ok && personaPaces[pace] {
			effective.pace = pace
		}
	}
	if effective.pace == "" {
		effective.pace = defaultPace
		defaulted("pace", defaultPace, "no persona layer sets teaching.pace and the profile has no teachingPace preference")
	}

	// Lists: the Global Persona's, extended by the Domain Persona's.
	for _, candidate := range layers {
		effective.activityTypes = appendUnique(effective.activityTypes, candidate.Teaching.PreferredActivityTypes...)
		effective.avoid = appendUnique(effective.avoid, candidate.Teaching.Avoid...)
		effective.emphasis = appendUnique(effective.emphasis, candidate.Teaching.Emphasis...)
		effective.feedback = appendUnique(effective.feedback, candidate.Teaching.Feedback...)
		effective.risks = appendUnique(effective.risks, candidate.Risks...)
	}
	if profile != nil {
		if style, ok := profile.Preferences["feedbackStyle"].(string); ok && style != "" {
			effective.feedback = appendUnique(effective.feedback, style)
		}
	}
	for field, values := range map[string][]string{
		"activityTypes": effective.activityTypes, "avoid": effective.avoid, "emphasis": effective.emphasis, "feedback": effective.feedback,
	} {
		if len(values) == 0 {
			defaulted(field, "[]", "no persona layer (or, for feedback, profile feedbackStyle) sets it")
		}
	}
	// Analogy policy: the most specific layer's default priority, else the
	// profile's stacks by relative rank; the Domain Persona's semantic
	// overrides shadow the Global Persona's for the same concept.
	for _, candidate := range layers {
		if len(candidate.AnalogyPolicy.DefaultPriority) != 0 {
			effective.defaultPriority = candidate.AnalogyPolicy.DefaultPriority
		}
	}
	switch {
	case len(p.stacks) == 0:
		defaulted("analogies", "none", "the learner profile lists no stacks to draw analogies from")
	case len(effective.defaultPriority) == 0:
		effective.defaultPriority = p.ranked
		defaulted("analogyPolicy.defaultPriority", strings.Join(p.ranked, ","), "no persona layer sets analogyPolicy.defaultPriority; the profile's stacks by relative rank apply")
	}
	seen := map[string]bool{}
	for index := len(layers) - 1; index >= 0; index-- {
		name := "global"
		if layers[index] == layer {
			name = "domain"
		}
		concepts := make([]string, 0, len(layers[index].AnalogyPolicy.SemanticOverrides))
		for concept := range layers[index].AnalogyPolicy.SemanticOverrides {
			concepts = append(concepts, concept)
		}
		sort.Strings(concepts)
		for _, concept := range concepts {
			if seen[concept] {
				continue
			}
			seen[concept] = true
			override := layers[index].AnalogyPolicy.SemanticOverrides[concept]
			effective.overrides = append(effective.overrides, analogyOverride{concept: concept, layer: name, reason: override.Reason, prefer: override.Prefer})
		}
	}
	return effective
}

func appendUnique(values []string, more ...string) []string {
	for _, value := range more {
		if !contains(values, value) {
			values = append(values, value)
		}
	}
	return values
}

// competencyTerms are the concept terms of a competency: the segments of its
// ID after the domain, the leaf of each shared scaffold, and the hyphenated
// parts of each.
func competencyTerms(definition domain.Competency) []string {
	var segments []string
	if parts := strings.Split(definition.ID, "."); len(parts) > 1 {
		segments = append(segments, parts[1:]...)
	}
	for _, scaffold := range definition.SharedScaffolds {
		parts := strings.Split(scaffold, ".")
		segments = append(segments, parts[len(parts)-1])
	}
	var terms []string
	for _, segment := range segments {
		terms = appendUnique(terms, strings.ToLower(segment))
		if strings.Contains(segment, "-") {
			terms = appendUnique(terms, strings.Split(strings.ToLower(segment), "-")...)
		}
	}
	return terms
}

// termMatches reports whether a word names a term, allowing a plural.
func termMatches(word, term string) bool {
	return word == term || word+"s" == term || term+"s" == word
}

func matchesAny(word string, terms []string) bool {
	for _, term := range terms {
		if termMatches(word, term) {
			return true
		}
	}
	return false
}

func riskWords(risk string) []string {
	return strings.FieldsFunc(strings.ToLower(risk), func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})
}

// shapeRequest holds what one unit's teaching shape is derived from.
type shapeRequest struct {
	competencies []SpecCompetency
	definitions  map[string]domain.Competency
	domains      []string
}

// shape derives a unit's teaching shape and its private persona basis.
func (p TeachingPersona) shape(request shapeRequest) (TeachingShape, PersonaBasis) {
	shape := TeachingShape{
		ActivityTypes: []string{}, Avoid: []string{}, Emphasis: []string{}, Feedback: []string{}, Analogies: []CompetencyAnalogy{},
	}
	basis := PersonaBasis{Risks: []UnitRisk{}, AnalogyReasons: []AnalogyReason{}, SourceExperience: []SourceExperience{}}
	unitTerms := []string{}
	for _, domainName := range request.domains {
		effective := p.effectiveFor(domainName)
		if shape.Pace == "" {
			shape.Pace = effective.pace
		}
		shape.ActivityTypes = appendUnique(shape.ActivityTypes, effective.activityTypes...)
		shape.Avoid = appendUnique(shape.Avoid, effective.avoid...)
		shape.Emphasis = appendUnique(shape.Emphasis, effective.emphasis...)
		shape.Feedback = appendUnique(shape.Feedback, effective.feedback...)
	}
	if shape.Pace == "" {
		shape.Pace = defaultPace
	}

	experienced := map[string]bool{}
	for _, competency := range request.competencies {
		effective := p.effectiveFor(competency.Domain)
		terms := competencyTerms(request.definitions[competencyKey(competency.Domain, competency.ID)])
		analogy := CompetencyAnalogy{Domain: competency.Domain, Competency: competency.ID, Basis: AnalogyNone, Concepts: []string{}, Sources: []string{}}
		for _, override := range effective.overrides {
			if !matchesAny(override.concept, terms) {
				continue
			}
			sources := p.listed(override.prefer)
			if len(sources) == 0 {
				continue
			}
			analogy.Basis = AnalogySemanticOverride
			analogy.Concepts = append(analogy.Concepts, override.concept)
			analogy.Sources = appendUnique(analogy.Sources, sources...)
			terms = appendUnique(terms, override.concept)
			if override.reason != "" {
				basis.AnalogyReasons = append(basis.AnalogyReasons, AnalogyReason{
					Domain: competency.Domain, Competency: competency.ID, Concept: override.concept, Layer: override.layer, Reason: override.reason,
				})
			}
		}
		if analogy.Basis == AnalogyNone {
			if sources := p.listed(effective.defaultPriority); len(sources) != 0 {
				analogy.Basis, analogy.Sources = AnalogyDefaultPriority, sources
			}
		}
		for _, source := range analogy.Sources {
			if !experienced[source] {
				experienced[source] = true
				basis.SourceExperience = append(basis.SourceExperience, p.stacks[source])
			}
		}
		unitTerms = appendUnique(unitTerms, terms...)
		shape.Analogies = append(shape.Analogies, analogy)
	}

	seen := map[string]bool{}
	for _, domainName := range request.domains {
		for _, risk := range p.effectiveFor(domainName).risks {
			if seen[risk] {
				continue
			}
			seen[risk] = true
			named := UnitRisk{Risk: risk, Concepts: []string{}}
			words := riskWords(risk)
			for _, term := range unitTerms {
				for _, word := range words {
					if termMatches(word, term) {
						named.Concepts = appendUnique(named.Concepts, term)
					}
				}
			}
			basis.Risks = append(basis.Risks, named)
		}
	}
	return shape, basis
}

func (p TeachingPersona) effectiveFor(domainName string) effectivePersona {
	if effective, ok := p.effective[domainName]; ok {
		return effective
	}
	return effectivePersona{pace: defaultPace}
}

// listed keeps the analogy sources the learner profile lists, in order.
func (p TeachingPersona) listed(sources []string) []string {
	result := []string{}
	for _, source := range sources {
		key := strings.ToLower(source)
		if _, ok := p.stacks[key]; ok {
			result = appendUnique(result, key)
		}
	}
	return result
}
