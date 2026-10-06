package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/adams100111/agentic-learning-partner/internal/platform"
	"github.com/adams100111/agentic-learning-partner/internal/platform/pylearn"
)

const platformUsage = "usage: alp platform inspect --adapter ID --target ID --curriculum FILE"

// defaultPlatforms is the composition root for built-in platform adapters.
func defaultPlatforms() platform.Registry {
	return platform.NewRegistry(pylearn.NewAdapter())
}

func (a App) platforms() platform.Registry {
	if a.Platforms != nil {
		return *a.Platforms
	}
	return defaultPlatforms()
}

// platformCommand is one `alp platform` subcommand and the capability it needs.
type platformCommand struct {
	capability platform.Capability
	run        func(a App, adapter platform.Adapter, target string, flags platformFlags) int
}

type platformFlags struct {
	curriculum string
}

var platformCommands = map[string]platformCommand{
	"inspect": {capability: platform.CurriculumReader, run: runPlatformInspect},
}

func (a App) runPlatform(args []string) int {
	if len(args) == 0 {
		return a.platformUsageError("a platform command is required")
	}
	command, ok := platformCommands[args[0]]
	if !ok {
		return a.platformUsageError(fmt.Sprintf("unknown platform command %q", args[0]))
	}
	flags := flag.NewFlagSet("platform "+args[0], flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	adapterID := flags.String("adapter", "", "platform adapter ID")
	target := flags.String("target", "", "platform Learning Target ID")
	curriculum := flags.String("curriculum", "", "curriculum export file")
	if err := flags.Parse(args[1:]); err != nil {
		return a.platformUsageError(err.Error())
	}
	if flags.NArg() != 0 {
		return a.platformUsageError(fmt.Sprintf("unexpected argument %q", flags.Arg(0)))
	}
	if *adapterID == "" || *target == "" {
		return a.platformUsageError("--adapter and --target are required")
	}
	adapter, err := a.platforms().Require(*adapterID, command.capability)
	if err != nil {
		return a.platformFailure(err)
	}
	return command.run(a, adapter, *target, platformFlags{curriculum: *curriculum})
}

type inspectOutput struct {
	SchemaVersion         int                   `json:"schemaVersion"`
	Adapter               string                `json:"adapter"`
	Target                platform.ExternalID   `json:"target"`
	Title                 string                `json:"title,omitempty"`
	Capabilities          []platform.Capability `json:"capabilities"`
	StableIdentifierKinds []string              `json:"stableIdentifierKinds"`
	CurriculumExport      inspectSource         `json:"curriculumExport"`
	Mapping               *platform.MappingRef  `json:"mapping"`
	Phases                []platform.Phase      `json:"phases"`
	Items                 []inspectItem         `json:"items"`
}

type inspectSource struct {
	SchemaVersion int    `json:"schemaVersion"`
	ContentHash   string `json:"contentHash"`
}

type inspectItem struct {
	Ref    platform.ExternalID  `json:"ref"`
	Kind   string               `json:"kind"`
	Phase  string               `json:"phase,omitempty"`
	Title  string               `json:"title,omitempty"`
	Parent *platform.ExternalID `json:"parent,omitempty"`
}

func runPlatformInspect(a App, adapter platform.Adapter, target string, flags platformFlags) int {
	if flags.curriculum == "" {
		return a.platformUsageError("--curriculum is required: pass the platform's versioned curriculum export")
	}
	reader, ok := adapter.(platform.CurriculumSource)
	if !ok {
		return a.platformFailure(fmt.Errorf("platform adapter %q declares %s but does not implement it", adapter.ID(), platform.CurriculumReader))
	}
	data, err := os.ReadFile(flags.curriculum)
	if err != nil {
		return a.platformFailure(fmt.Errorf("read curriculum export: %w", err))
	}
	curriculum, err := reader.ReadCurriculum(data, target)
	if err != nil {
		return a.platformFailure(err)
	}

	output := inspectOutput{
		SchemaVersion:         1,
		Adapter:               adapter.ID(),
		Target:                curriculum.Target,
		Title:                 curriculum.Title,
		Capabilities:          platform.DeclaredCapabilities(adapter),
		StableIdentifierKinds: sortedCopy(adapter.StableIdentifierKinds()),
		CurriculumExport:      inspectSource{SchemaVersion: curriculum.SchemaVersion, ContentHash: curriculum.ContentHash},
		Mapping:               curriculum.Mapping,
		Phases:                curriculum.Phases,
		Items:                 make([]inspectItem, 0, len(curriculum.Items)),
	}
	for _, item := range curriculum.Items {
		output.Items = append(output.Items, inspectItem{Ref: item.Ref, Kind: item.Kind, Phase: item.Phase, Title: item.Title, Parent: item.Parent})
	}
	return a.writePlatformJSON(output, 0)
}

type platformErrorOutput struct {
	Error *platform.Error `json:"error"`
}

const codeUsage = "usage"

func (a App) platformUsageError(message string) int {
	fmt.Fprintln(a.ErrOut, message)
	fmt.Fprintln(a.ErrOut, platformUsage)
	return a.writePlatformJSON(platformErrorOutput{Error: &platform.Error{Code: codeUsage, Message: message}}, 2)
}

func (a App) platformFailure(err error) int {
	var contractErr *platform.Error
	if !errors.As(err, &contractErr) {
		contractErr = &platform.Error{Code: "error", Message: err.Error()}
	}
	fmt.Fprintln(a.ErrOut, contractErr.Message)
	return a.writePlatformJSON(platformErrorOutput{Error: contractErr}, 1)
}

// writePlatformJSON writes deterministic, indented JSON and returns code.
func (a App) writePlatformJSON(value any, code int) int {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintf(a.ErrOut, "encode platform output: %v\n", err)
		return 1
	}
	if _, err := a.Out.Write(buffer.Bytes()); err != nil {
		fmt.Fprintf(a.ErrOut, "write platform output: %v\n", err)
		return 1
	}
	return code
}

func sortedCopy(values []string) []string {
	sorted := append([]string{}, values...)
	sort.Strings(sorted)
	return sorted
}
