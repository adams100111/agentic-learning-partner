package platform_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// platformNativeTerm matches PyLearn-native content concepts that must stay
// inside the PyLearn adapter (ADR-0054, ADR-0056).
var platformNativeTerm = regexp.MustCompile(`(?i:\b(reel|scene|mdx))|[a-z0-9](Reel|Scene|Mdx|MDX)`)

func TestALPCoreHasNoPlatformNativeContentConcepts(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	adapters := filepath.Join(root, "internal", "platform", "pylearn")
	var violations []string
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			name := entry.Name()
			if path == adapters || name == "testdata" || name == ".git" || name == ".devenv" || name == "docs" {
				return filepath.SkipDir
			}
			return nil
		}
		isCode := strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go")
		isSchema := strings.HasSuffix(path, ".schema.json")
		if !isCode && !isSchema {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for number, line := range strings.Split(string(data), "\n") {
			if platformNativeTerm.MatchString(line) {
				relative, _ := filepath.Rel(root, path)
				violations = append(violations, relative+":"+strconv.Itoa(number+1)+": "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("platform-native content concepts in ALP core:\n%s", strings.Join(violations, "\n"))
	}
}
