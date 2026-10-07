package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/adams100111/agentic-learning-partner/internal/platform"
	"github.com/adams100111/agentic-learning-partner/internal/platform/pylearn"
)

const platformUsage = "usage: alp platform inspect --adapter ID --target ID --curriculum FILE\n" +
	"       alp platform mapping validate --adapter ID --target ID --curriculum FILE --mapping FILE\n" +
	"       alp platform account link --adapter ID --instance ID --user ID --confirm [--workspace PATH]\n" +
	"       alp platform import --adapter ID --target ID --curriculum FILE --mapping FILE --export FILE [--cursor CURSOR] [--workspace PATH]\n" +
	"       alp platform plan --adapter ID --target ID --curriculum FILE --mapping FILE [--constraints FILE] [--intent target-skeleton|curriculum|unit|activity|patch] [--unit ID] [--workspace PATH]\n" +
	"       alp platform decision accept --adapter ID --target ID --curriculum FILE --mapping FILE [--constraints FILE] --unit ITEM --mode skip|challenge|skim|full --basis REVISION --confirm [--reason TEXT] [--workspace PATH]\n" +
	"       alp platform decision revoke --adapter ID --target ID --curriculum FILE --mapping FILE [--constraints FILE] --decision ID --basis REVISION --confirm [--reason TEXT] [--workspace PATH]\n" +
	"       alp platform gates record --adapter ID --target ID --curriculum FILE --plan PLAN --result FILE [--realization FILE] [--workspace PATH]"

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

// platformCommand is one `alp platform` subcommand and the capabilities it
// needs, in the order they are checked.
type platformCommand struct {
	capabilities []platform.Capability
	// targetless commands act on a platform account rather than a Learning
	// Target, so they take no --target.
	targetless bool
	run        func(a App, adapter platform.Adapter, target string, flags platformFlags) int
}

type platformFlags struct {
	curriculum string
	mapping    string
	export     string
	cursor     string
	cursorSet  bool
	instance   string
	user       string
	confirm    bool
	workspace  string
	// Target adaptation flags.
	constraints string
	unit        string
	mode        string
	basis       string
	decision    string
	reason      string
	// Authoring flags.
	intent string
	// Gate flags.
	plan        string
	result      string
	realization string
}

// platformCommands is keyed by the command words, e.g. "mapping validate".
var platformCommands = map[string]platformCommand{
	"inspect": {capabilities: []platform.Capability{platform.CurriculumReader}, run: runPlatformInspect},
	// Mapping validation needs the curriculum: it is the only source of which
	// items are declared-stable (ADR-0057).
	"mapping validate": {capabilities: []platform.Capability{platform.ContentMapper, platform.CurriculumReader}, run: runPlatformMappingValidate},
	// Account links exist only so activity can be imported for a learner.
	"account link": {capabilities: []platform.Capability{platform.ActivitySource}, targetless: true, run: runPlatformAccountLink},
	// Import grades activity through the target's mapping, which is validated
	// against the curriculum export.
	"import": {capabilities: []platform.Capability{platform.ActivitySource, platform.ContentMapper, platform.CurriculumReader}, run: runPlatformImport},
	// Planning projects learner state onto the target's mapped curriculum and
	// derives Curriculum/Learning Unit Specifications and an Authoring Plan
	// realized by the platform-declared authoring target skill.
	"plan": {capabilities: []platform.Capability{platform.ContentMapper, platform.CurriculumReader, platform.AuthoringTarget}, run: runPlatformPlan},
	// Decisions are confirmed against the projection plan rebuilds, so they
	// need the same inputs.
	"decision accept": {capabilities: []platform.Capability{platform.ContentMapper, platform.CurriculumReader}, run: runPlatformDecisionAccept},
	"decision revoke": {capabilities: []platform.Capability{platform.ContentMapper, platform.CurriculumReader}, run: runPlatformDecisionRevoke},
	// Gate results attach to Authoring Plans the authoring target realized;
	// Realization Links are checked against the curriculum export, the only
	// source of which items are declared-stable.
	"gates record": {capabilities: []platform.Capability{platform.PlatformValidator, platform.AuthoringTarget, platform.CurriculumReader}, run: runPlatformGatesRecord},
}

// platformCommandGroups are first words that take a second command word.
var platformCommandGroups = map[string]bool{"mapping": true, "account": true, "decision": true, "gates": true}

func (a App) runPlatform(args []string) int {
	if len(args) == 0 {
		return a.platformUsageError("a platform command is required")
	}
	name, rest := args[0], args[1:]
	if platformCommandGroups[name] {
		if len(rest) == 0 || strings.HasPrefix(rest[0], "-") {
			return a.platformUsageError(fmt.Sprintf("platform %s requires a subcommand", name))
		}
		name, rest = name+" "+rest[0], rest[1:]
	}
	command, ok := platformCommands[name]
	if !ok {
		return a.platformUsageError(fmt.Sprintf("unknown platform command %q", name))
	}
	var parsed platformArgs
	flags := newPlatformFlagSet(name, &parsed)
	if err := flags.Parse(rest); err != nil {
		return a.platformUsageError(err.Error())
	}
	if flags.NArg() != 0 {
		return a.platformUsageError(fmt.Sprintf("unexpected argument %q", flags.Arg(0)))
	}
	flags.Visit(func(set *flag.Flag) {
		if set.Name == "cursor" {
			parsed.cursorSet = true
		}
	})
	switch {
	case command.targetless && parsed.target != "":
		return a.platformUsageError(fmt.Sprintf("platform %s takes no --target", name))
	case command.targetless && parsed.adapter == "":
		return a.platformUsageError("--adapter is required")
	case !command.targetless && (parsed.adapter == "" || parsed.target == ""):
		return a.platformUsageError("--adapter and --target are required")
	}
	var adapter platform.Adapter
	for _, capability := range command.capabilities {
		required, err := a.platforms().Require(parsed.adapter, capability)
		if err != nil {
			return a.platformFailure(err)
		}
		adapter = required
	}
	return command.run(a, adapter, parsed.target, parsed.platformFlags)
}

// platformArgs are the parsed flags of one `alp platform` command.
type platformArgs struct {
	adapter string
	target  string
	platformFlags
}

// newPlatformFlagSet defines every `alp platform` flag on a new flag set that
// parses into args. platformUsage documents which flags each command takes.
func newPlatformFlagSet(name string, args *platformArgs) *flag.FlagSet {
	flags := flag.NewFlagSet("platform "+name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&args.adapter, "adapter", "", "platform adapter ID")
	flags.StringVar(&args.target, "target", "", "platform Learning Target ID")
	flags.StringVar(&args.curriculum, "curriculum", "", "curriculum export file")
	flags.StringVar(&args.mapping, "mapping", "", "platform content mapping file")
	flags.StringVar(&args.export, "export", "", "platform activity export file")
	flags.StringVar(&args.cursor, "cursor", "", "activity cursor returned by the previous import")
	flags.StringVar(&args.instance, "instance", "", "platform instance ID")
	flags.StringVar(&args.user, "user", "", "platform user ID")
	flags.BoolVar(&args.confirm, "confirm", false, "the learner explicitly confirms this action")
	flags.StringVar(&args.constraints, "constraints", "", "target constraints file")
	flags.StringVar(&args.unit, "unit", "", "unit item ID of the Learning Target")
	flags.StringVar(&args.mode, "mode", "", "adaptation mode: skip, challenge, skim or full")
	flags.StringVar(&args.basis, "basis", "", "Target Adaptation Projection revision the learner confirmed against")
	flags.StringVar(&args.decision, "decision", "", "Accepted Adaptation Decision ID")
	flags.StringVar(&args.reason, "reason", "", "the learner's reason for the decision")
	flags.StringVar(&args.intent, "intent", "", "Authoring Intent: target-skeleton, curriculum, unit, activity or patch")
	flags.StringVar(&args.plan, "plan", "", "Authoring Plan ID the gate result is attached to")
	flags.StringVar(&args.result, "result", "", "platform gate result file")
	flags.StringVar(&args.realization, "realization", "", "realization report from the authoring target")
	flags.StringVar(&args.workspace, "workspace", "", "learner workspace path")
	return flags
}

type inspectOutput struct {
	SchemaVersion         int                   `json:"schemaVersion"`
	Adapter               string                `json:"adapter"`
	Target                platform.ExternalID   `json:"target"`
	Title                 string                `json:"title,omitempty"`
	Capabilities          []platform.Capability `json:"capabilities"`
	StableIdentifierKinds []string              `json:"stableIdentifierKinds"`
	// AuthoringTarget names the platform-declared authoring target skill
	// (ADR-0056, Q24); null when the adapter declares no Authoring Target.
	AuthoringTarget  *platform.AuthoringTargetRef `json:"authoringTarget"`
	CurriculumExport inspectSource                `json:"curriculumExport"`
	Mapping          *platform.MappingRef         `json:"mapping"`
	Phases           []platform.Phase             `json:"phases"`
	Items            []inspectItem                `json:"items"`
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

// readCurriculum reads the target from the --curriculum export; on failure it
// has already written the error and returns the exit code.
func (a App) readCurriculum(adapter platform.Adapter, target string, flags platformFlags) (platform.Curriculum, int, bool) {
	if flags.curriculum == "" {
		return platform.Curriculum{}, a.platformUsageError("--curriculum is required: pass the platform's versioned curriculum export"), false
	}
	reader, ok := adapter.(platform.CurriculumSource)
	if !ok {
		return platform.Curriculum{}, a.platformFailure(fmt.Errorf("platform adapter %q declares %s but does not implement it", adapter.ID(), platform.CurriculumReader)), false
	}
	data, err := os.ReadFile(flags.curriculum)
	if err != nil {
		return platform.Curriculum{}, a.platformFailure(fmt.Errorf("read curriculum export: %w", err)), false
	}
	curriculum, err := reader.ReadCurriculum(data, target)
	if err != nil {
		return platform.Curriculum{}, a.platformFailure(err), false
	}
	return curriculum, 0, true
}

func runPlatformInspect(a App, adapter platform.Adapter, target string, flags platformFlags) int {
	curriculum, code, ok := a.readCurriculum(adapter, target, flags)
	if !ok {
		return code
	}

	var authoringTarget *platform.AuthoringTargetRef
	if platform.Declares(adapter, platform.AuthoringTarget) {
		declaration, ok := adapter.(platform.AuthoringTargetDeclaration)
		if !ok {
			return a.platformFailure(fmt.Errorf("platform adapter %q declares %s but does not implement it", adapter.ID(), platform.AuthoringTarget))
		}
		authoringTarget = &platform.AuthoringTargetRef{Platform: adapter.ID(), Skill: declaration.AuthoringSkill()}
	}

	output := inspectOutput{
		SchemaVersion:         1,
		Adapter:               adapter.ID(),
		Target:                curriculum.Target,
		Title:                 curriculum.Title,
		Capabilities:          platform.DeclaredCapabilities(adapter),
		StableIdentifierKinds: sortedCopy(adapter.StableIdentifierKinds()),
		AuthoringTarget:       authoringTarget,
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

type mappingValidateOutput struct {
	SchemaVersion    int                       `json:"schemaVersion"`
	Adapter          string                    `json:"adapter"`
	Target           platform.ExternalID       `json:"target"`
	Mapping          platform.MappingSource    `json:"mapping"`
	CurriculumExport inspectSource             `json:"curriculumExport"`
	Packs            []platform.PackStatus     `json:"packs"`
	Valid            bool                      `json:"valid"`
	Summary          platform.MappingSummary   `json:"summary"`
	Entries          []platform.MappedItem     `json:"entries"`
	Problems         []platform.MappingProblem `json:"problems"`
}

func runPlatformMappingValidate(a App, adapter platform.Adapter, target string, flags platformFlags) int {
	if flags.mapping == "" {
		return a.platformUsageError("--mapping is required: pass the platform's content mapping file")
	}
	validator, ok := adapter.(platform.ContentMappingValidator)
	if !ok {
		return a.platformFailure(fmt.Errorf("platform adapter %q declares %s but does not implement it", adapter.ID(), platform.ContentMapper))
	}
	curriculum, code, ok := a.readCurriculum(adapter, target, flags)
	if !ok {
		return code
	}
	data, err := os.ReadFile(flags.mapping)
	if err != nil {
		return a.platformFailure(fmt.Errorf("read mapping: %w", err))
	}
	report, err := validator.ValidateContentMapping(data, filepath.Base(flags.mapping), curriculum)
	if err != nil {
		return a.platformFailure(err)
	}
	output := mappingValidateOutput{
		SchemaVersion:    1,
		Adapter:          adapter.ID(),
		Target:           curriculum.Target,
		CurriculumExport: inspectSource{SchemaVersion: curriculum.SchemaVersion, ContentHash: curriculum.ContentHash},
		Mapping:          report.Mapping,
		Packs:            report.Packs,
		Valid:            report.Valid,
		Summary:          report.Summary,
		Entries:          report.Entries,
		Problems:         report.Problems,
	}
	for _, problem := range report.Problems {
		location := problem.Path
		if location == "" {
			location = "/"
		}
		fmt.Fprintf(a.ErrOut, "%s %s at %s: %s\n", problem.Severity, problem.Code, location, problem.Message)
	}
	if !report.Valid {
		return a.writePlatformJSON(output, 1)
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
