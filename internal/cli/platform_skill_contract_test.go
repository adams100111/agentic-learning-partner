package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/adams100111/agentic-learning-partner/internal/platform"
)

// The portable skills drive ALP only through `alp platform` (#74). These
// tests keep them from drifting from the CLI: every invocation a skill names
// must be a real command with flags that command takes and, where the usage
// enumerates values, a value it accepts.

// usageCommand is one `alp platform` command as platformUsage documents it.
type usageCommand struct {
	flags map[string][]string // flag → enumerated values (nil when free-form)
}

// platformUsageCommands parses platformUsage into its commands.
func platformUsageCommands(t *testing.T) map[string]usageCommand {
	t.Helper()
	commands := map[string]usageCommand{}
	for _, line := range strings.Split(platformUsage, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "usage:"))
		tokens := strings.Fields(line)
		if len(tokens) < 3 || tokens[0] != "alp" || tokens[1] != "platform" {
			t.Fatalf("platformUsage line %q is not an alp platform command", line)
		}
		var words []string
		rest := tokens[2:]
		for len(rest) > 0 && !strings.HasPrefix(strings.TrimLeft(rest[0], "["), "-") {
			words, rest = append(words, rest[0]), rest[1:]
		}
		name := strings.Join(words, " ")
		if _, duplicate := commands[name]; duplicate {
			t.Fatalf("platformUsage documents %q twice", name)
		}
		command := usageCommand{flags: map[string][]string{}}
		for i := 0; i < len(rest); i++ {
			token := strings.Trim(rest[i], "[]")
			if !strings.HasPrefix(token, "--") {
				t.Fatalf("platformUsage %q: unexpected token %q", name, rest[i])
			}
			var values []string
			if i+1 < len(rest) && !strings.HasPrefix(strings.TrimLeft(rest[i+1], "["), "-") {
				value := strings.Trim(rest[i+1], "[]")
				if strings.Contains(value, "|") {
					values = strings.Split(value, "|")
				}
				i++
			}
			command.flags[strings.TrimPrefix(token, "--")] = values
		}
		commands[name] = command
	}
	return commands
}

func TestPlatformUsageDocumentsExactlyTheCommandsAndFlagsTheCLIDefines(t *testing.T) {
	usage := platformUsageCommands(t)
	for name := range platformCommands {
		if _, ok := usage[name]; !ok {
			t.Errorf("platform command %q is missing from platformUsage", name)
		}
	}
	defined := map[string]bool{}
	newPlatformFlagSet("usage", &platformArgs{}).VisitAll(func(f *flag.Flag) { defined[f.Name] = true })
	for name, command := range usage {
		if _, ok := platformCommands[name]; !ok {
			t.Errorf("platformUsage documents unknown command %q", name)
		}
		for flagName := range command.flags {
			if !defined[flagName] {
				t.Errorf("platformUsage %q documents flag --%s, which the CLI does not define", name, flagName)
			}
		}
	}
}

// skillInvocation is one `alp platform …` invocation found in a skill.
type skillInvocation struct {
	skill  string
	tokens []string
}

var inlineCode = regexp.MustCompile("`([^`\n]+)`")

// skillPlatformInvocations returns every `alp platform` invocation in the
// skills' inline code and code blocks, with code-block line continuations
// joined.
func skillPlatformInvocations(t *testing.T, skillsDir string) []skillInvocation {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(skillsDir, "*", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatalf("no skills under %s", skillsDir)
	}
	var invocations []skillInvocation
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		skill := filepath.Base(filepath.Dir(path))
		var snippets []string
		inBlock := false
		var pending strings.Builder
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "```") {
				inBlock = !inBlock
				continue
			}
			if inBlock {
				trimmed := strings.TrimSpace(line)
				if strings.HasSuffix(trimmed, `\`) {
					pending.WriteString(strings.TrimSuffix(trimmed, `\`) + " ")
					continue
				}
				pending.WriteString(trimmed)
				snippets = append(snippets, pending.String())
				pending.Reset()
				continue
			}
			for _, match := range inlineCode.FindAllStringSubmatch(line, -1) {
				snippets = append(snippets, match[1])
			}
		}
		for _, snippet := range snippets {
			fields := shellFields(snippet)
			for i := 0; i+1 < len(fields); i++ {
				if fields[i] == "alp" && fields[i+1] == "platform" {
					end := len(fields)
					for j := i + 2; j < len(fields); j++ {
						// A shell operator or comment ends the invocation.
						if fields[j] == "|" || fields[j] == ">" || fields[j] == "&&" || fields[j] == ";" || fields[j] == "#" {
							end = j
							break
						}
					}
					invocations = append(invocations, skillInvocation{skill: skill, tokens: fields[i+2 : end]})
				}
			}
		}
	}
	return invocations
}

// shellFields splits a command line on whitespace, keeping a <placeholder>
// or a "quoted value" that contains spaces as one field.
func shellFields(line string) []string {
	var fields []string
	var current strings.Builder
	closing := rune(0)
	for _, r := range line {
		switch {
		case closing != 0:
			current.WriteRune(r)
			if r == closing {
				closing = 0
			}
		case r == ' ' || r == '\t':
			if current.Len() > 0 {
				fields = append(fields, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
			if r == '<' {
				closing = '>'
			} else if r == '"' {
				closing = '"'
			}
		}
	}
	if current.Len() > 0 {
		fields = append(fields, current.String())
	}
	return fields
}

// placeholder reports whether a skill names a value to fill in rather than
// a literal one.
func placeholder(value string) bool {
	return strings.Contains(value, "<") || strings.Contains(value, "…")
}

// checkSkillInvocations reports every invocation that names an unknown
// command or a flag or enumerated value the command does not take, and which
// commands each skill invokes.
func checkSkillInvocations(usage map[string]usageCommand, invocations []skillInvocation) (problems []string, used map[string]bool) {
	booleans := map[string]bool{}
	newPlatformFlagSet("check", &platformArgs{}).VisitAll(func(f *flag.Flag) {
		if b, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && b.IsBoolFlag() {
			booleans[f.Name] = true
		}
	})
	used = map[string]bool{}
	for _, invocation := range invocations {
		tokens := invocation.tokens
		if len(tokens) == 0 {
			continue // prose naming the `alp platform` family itself
		}
		words := []string{tokens[0]}
		tokens = tokens[1:]
		if platformCommandGroups[words[0]] {
			if len(tokens) == 0 || strings.HasPrefix(tokens[0], "-") {
				problems = append(problems, fmt.Sprintf("skill %s: `alp platform %s` names no subcommand", invocation.skill, words[0]))
				continue
			}
			words, tokens = append(words, tokens[0]), tokens[1:]
		}
		name := strings.Join(words, " ")
		command, ok := usage[name]
		if !ok {
			problems = append(problems, fmt.Sprintf("skill %s: `alp platform %s` is not an alp platform command", invocation.skill, name))
			continue
		}
		used[invocation.skill+": "+name] = true
		for i := 0; i < len(tokens); i++ {
			token := strings.Trim(tokens[i], "[]")
			if !strings.HasPrefix(token, "--") {
				problems = append(problems, fmt.Sprintf("skill %s: `alp platform %s`: unexpected argument %q", invocation.skill, name, tokens[i]))
				continue
			}
			flagName := strings.TrimPrefix(token, "--")
			values, ok := command.flags[flagName]
			if !ok {
				problems = append(problems, fmt.Sprintf("skill %s: `alp platform %s` takes no --%s", invocation.skill, name, flagName))
				if i+1 < len(tokens) && !strings.HasPrefix(strings.TrimLeft(tokens[i+1], "["), "--") {
					i++ // the unknown flag's value
				}
				continue
			}
			if booleans[flagName] {
				continue
			}
			if i+1 >= len(tokens) || strings.HasPrefix(strings.TrimLeft(tokens[i+1], "["), "--") {
				problems = append(problems, fmt.Sprintf("skill %s: `alp platform %s --%s` has no value", invocation.skill, name, flagName))
				continue
			}
			i++
			value := strings.Trim(tokens[i], "[]\"")
			if values == nil || placeholder(value) {
				continue
			}
			for _, part := range strings.Split(value, "|") {
				if !contains(values, part) {
					problems = append(problems, fmt.Sprintf("skill %s: `alp platform %s --%s %s`: %q is not one of %v", invocation.skill, name, flagName, value, part, values))
				}
			}
		}
	}
	return problems, used
}

func TestSkillInvocationCheckCatchesDrift(t *testing.T) {
	dir := t.TempDir()
	skill := filepath.Join(dir, "drifted")
	if err := os.MkdirAll(skill, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nname: drifted\ndescription: test\n---\n\n" +
		"Run `alp platform inspekt --adapter <a>`.\n\n" +
		"```bash\nalp platform plan --adapter <a> --target <t> \\\n  --intent whole-course --since <c>\n```\n\n" +
		"`alp platform decision accept --unit <u> --mode skip|later --basis <b> --confirm`\n" +
		"`alp platform gates`\n"
	if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	problems, _ := checkSkillInvocations(platformUsageCommands(t), skillPlatformInvocations(t, dir))
	want := []string{
		"`alp platform inspekt` is not an alp platform command",
		"\"whole-course\" is not one of",
		"`alp platform plan` takes no --since",
		"\"later\" is not one of",
		"`alp platform gates` names no subcommand",
	}
	joined := strings.Join(problems, "\n")
	for _, fragment := range want {
		if !strings.Contains(joined, fragment) {
			t.Errorf("drift %q was not reported; problems:\n%s", fragment, joined)
		}
	}
	if len(problems) != len(want) {
		t.Errorf("got %d problems, want %d:\n%s", len(problems), len(want), joined)
	}
}

func TestSkillPlatformInvocationsAreRealCommandsAndFlags(t *testing.T) {
	problems, used := checkSkillInvocations(platformUsageCommands(t), skillPlatformInvocations(t, filepath.Join("..", "..", "skills")))
	for _, problem := range problems {
		t.Error(problem)
	}

	// The platform skills must actually drive the commands their workflows
	// promise, so a parser that silently finds nothing cannot pass.
	for _, want := range []string{
		"adapt-platform-target: inspect",
		"adapt-platform-target: plan",
		"adapt-platform-target: decision accept",
		"adapt-platform-target: decision revoke",
		"orchestrate-platform-authoring: plan",
		"orchestrate-platform-authoring: inspect",
		"orchestrate-platform-authoring: gates record",
		"orchestrate-platform-authoring: account link",
		"orchestrate-platform-authoring: import",
	} {
		if !used[want] {
			found := make([]string, 0, len(used))
			for key := range used {
				found = append(found, key)
			}
			sort.Strings(found)
			t.Errorf("no `alp platform` invocation for %q; found %v", want, found)
		}
	}
}

// Decisions and account links are learner truth: every invocation a skill
// shows for them (a command with flags, not a mention by name) must carry
// --confirm, and decisions the --basis they were
// confirmed against.
func TestSkillsRecordLearnerDecisionsOnlyWithConfirmation(t *testing.T) {
	for _, invocation := range skillPlatformInvocations(t, filepath.Join("..", "..", "skills")) {
		if len(invocation.tokens) < 2 {
			continue
		}
		name := invocation.tokens[0] + " " + invocation.tokens[1]
		if name != "decision accept" && name != "decision revoke" && name != "account link" {
			continue
		}
		flags := map[string]bool{}
		for _, token := range invocation.tokens[2:] {
			if strings.HasPrefix(token, "--") {
				flags[strings.TrimPrefix(token, "--")] = true
			}
		}
		if len(flags) == 0 {
			continue // a reference to the command by name, not an invocation
		}
		if !flags["confirm"] {
			t.Errorf("skill %s: `alp platform %s` is shown without --confirm", invocation.skill, name)
		}
		if strings.HasPrefix(name, "decision") && !flags["basis"] {
			t.Errorf("skill %s: `alp platform %s` is shown without --basis", invocation.skill, name)
		}
	}
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

// ALP's platform skills hold no platform authoring knowledge (ADR-0056, Q24):
// they resolve the platform and its authoring target skill from `alp
// platform` output, so they never name a built-in adapter or its skill.
func TestPlatformSkillsNameNoBuiltInPlatform(t *testing.T) {
	registry := defaultPlatforms()
	var names []string
	for _, id := range registry.IDs() {
		names = append(names, id)
		adapter, err := registry.Lookup(id)
		if err != nil {
			t.Fatal(err)
		}
		if declaration, ok := adapter.(platform.AuthoringTargetDeclaration); ok {
			names = append(names, declaration.AuthoringSkill())
		}
	}
	for _, skill := range []string{"adapt-platform-target", "orchestrate-platform-authoring"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "skills", skill, "SKILL.md"))
		if err != nil {
			t.Fatal(err)
		}
		text := strings.ToLower(string(data))
		for _, name := range names {
			if strings.Contains(text, strings.ToLower(name)) {
				t.Errorf("skill %s names built-in platform %q; resolve it from alp platform output instead", skill, name)
			}
		}
	}
}
