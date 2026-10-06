package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	alpschemas "github.com/adams100111/agentic-learning-partner/schemas"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v3"
)

type ValidationIssue struct {
	File   string
	Path   string
	Reason string
}

func (i ValidationIssue) Error() string {
	if i.Path == "" {
		return fmt.Sprintf("%s: %s", i.File, i.Reason)
	}
	return fmt.Sprintf("%s:%s: %s", i.File, i.Path, i.Reason)
}

type Validator struct {
	compiled map[string]*jsonschema.Schema
}

func NewValidator() (*Validator, error) {
	compiler := jsonschema.NewCompiler()

	entries, err := fs.Glob(alpschemas.Files, "*.schema.json")
	if err != nil {
		return nil, fmt.Errorf("list embedded schemas: %w", err)
	}

	resourceIDs := make(map[string]string, len(entries))
	for _, name := range entries {
		data, err := alpschemas.Files.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("read embedded schema %s: %w", name, err)
		}
		var document map[string]any
		if err := json.Unmarshal(data, &document); err != nil {
			return nil, fmt.Errorf("parse embedded schema %s: %w", name, err)
		}
		id, ok := document["$id"].(string)
		if !ok || id == "" {
			return nil, fmt.Errorf("embedded schema %s: $id is required", name)
		}
		if err := compiler.AddResource(id, document); err != nil {
			return nil, fmt.Errorf("register embedded schema %s: %w", name, err)
		}
		resourceIDs[name] = id
	}

	compiled := make(map[string]*jsonschema.Schema, len(entries))
	for _, name := range entries {
		schema, err := compiler.Compile(resourceIDs[name])
		if err != nil {
			return nil, fmt.Errorf("compile embedded schema %s: %w", name, err)
		}
		compiled[name] = schema
	}
	return &Validator{compiled: compiled}, nil
}

type documentRule struct {
	Pattern string
	Schema  string
}

var workspaceDocumentRules = []documentRule{
	{Pattern: "workspace.yaml", Schema: "workspace.schema.json"},
	{Pattern: "workspace.json", Schema: "workspace.schema.json"},
	{Pattern: "profile/profile.yaml", Schema: "profile.schema.json"},
	{Pattern: "profile/profile.json", Schema: "profile.schema.json"},
	{Pattern: "personas/global.yaml", Schema: "persona.schema.json"},
	{Pattern: "personas/global.json", Schema: "persona.schema.json"},
	{Pattern: "personas/domains/*.yaml", Schema: "persona.schema.json"},
	{Pattern: "personas/domains/*.json", Schema: "persona.schema.json"},
	{Pattern: "evidence/*.yaml", Schema: "evidence.schema.json"},
	{Pattern: "evidence/*.json", Schema: "evidence.schema.json"},
	{Pattern: "assessments/*.yaml", Schema: "assessment.schema.json"},
	{Pattern: "assessments/*.json", Schema: "assessment.schema.json"},
	{Pattern: "state/competencies.yaml", Schema: "projection.schema.json"},
	{Pattern: "state/competencies.json", Schema: "projection.schema.json"},
	{Pattern: "state/review-queue.yaml", Schema: "review-queue.schema.json"},
	{Pattern: "state/review-queue.json", Schema: "review-queue.schema.json"},
	{Pattern: "state/learning-plan.yaml", Schema: "learning-plan.schema.json"},
	{Pattern: "state/learning-plan.json", Schema: "learning-plan.schema.json"},
}

func (v *Validator) ValidateWorkspace(root string) []ValidationIssue {
	var issues []ValidationIssue

	if _, err := os.Stat(filepath.Join(root, "workspace.yaml")); errors.Is(err, os.ErrNotExist) {
		if _, jsonErr := os.Stat(filepath.Join(root, "workspace.json")); errors.Is(jsonErr, os.ErrNotExist) {
			issues = append(issues, ValidationIssue{File: "workspace.yaml", Reason: "workspace manifest is required"})
		}
	}

	for _, rule := range workspaceDocumentRules {
		matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(rule.Pattern)))
		if err != nil {
			issues = append(issues, ValidationIssue{File: rule.Pattern, Reason: err.Error()})
			continue
		}
		sort.Strings(matches)
		for _, path := range matches {
			if issue := v.validateFile(root, path, rule.Schema); issue != nil {
				issues = append(issues, *issue)
			}
		}
	}
	return issues
}

func (v *Validator) validateFile(root, path, schemaName string) *ValidationIssue {
	data, err := os.ReadFile(path)
	if err != nil {
		return &ValidationIssue{File: relative(root, path), Reason: err.Error()}
	}
	return v.ValidateDocument(schemaName, relative(root, path), data)
}

func (v *Validator) ValidateDocument(schemaName, file string, data []byte) *ValidationIssue {
	document, err := decodeDocument(file, data)
	if err != nil {
		return &ValidationIssue{File: file, Reason: err.Error()}
	}

	schema, ok := v.compiled[schemaName]
	if !ok {
		return &ValidationIssue{File: file, Reason: "unknown schema " + schemaName}
	}
	if err := schema.Validate(document); err != nil {
		var validationErr *jsonschema.ValidationError
		if errors.As(err, &validationErr) {
			return &ValidationIssue{
				File:   file,
				Path:   pointer(validationErr.InstanceLocation),
				Reason: validationErr.Error(),
			}
		}
		return &ValidationIssue{File: file, Reason: err.Error()}
	}
	return nil
}

func decodeDocument(path string, data []byte) (any, error) {
	var value any
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		if err := json.Unmarshal(data, &value); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
	case ".yaml", ".yml":
		var yamlValue any
		if err := yaml.Unmarshal(data, &yamlValue); err != nil {
			return nil, fmt.Errorf("invalid YAML: %w", err)
		}
		normalized, err := json.Marshal(yamlValue)
		if err != nil {
			return nil, fmt.Errorf("normalize YAML: %w", err)
		}
		if err := json.Unmarshal(normalized, &value); err != nil {
			return nil, fmt.Errorf("normalize YAML: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported document extension %q", filepath.Ext(path))
	}
	return value, nil
}

func relative(root, path string) string {
	value, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(value)
}

func pointer(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	return "/" + strings.Join(parts, "/")
}
