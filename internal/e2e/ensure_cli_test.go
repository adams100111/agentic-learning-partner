package e2e

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

// These tests drive scripts/ensure-cli.sh (the plugin's SessionStart hook)
// against a fake plugin root, a fake `alp` and a stub installer. No network.

func repoRootDir(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

type ensureEnv struct {
	t       *testing.T
	root    string // fake plugin root
	bin     string // fake PATH dir
	dest    string // ALP_INSTALL_DIR
	calls   string // installer call log
	home    string
	getSh   string
	extra   []string
	version string
}

func newEnsureEnv(t *testing.T) *ensureEnv {
	t.Helper()
	tmp := t.TempDir()
	e := &ensureEnv{t: t, version: "0.1.1"}
	e.root = filepath.Join(tmp, "plugin")
	e.bin = filepath.Join(tmp, "bin")
	e.dest = filepath.Join(tmp, "dest")
	e.home = filepath.Join(tmp, "home")
	e.calls = filepath.Join(tmp, "calls.log")
	e.getSh = filepath.Join(tmp, "get.sh")
	for _, d := range []string{filepath.Join(e.root, "scripts"), filepath.Join(e.root, ".claude-plugin"), e.bin, e.home} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	script, err := os.ReadFile(filepath.Join(repoRootDir(t), "scripts", "ensure-cli.sh"))
	if err != nil {
		t.Fatal(err)
	}
	e.write(filepath.Join(e.root, "scripts", "ensure-cli.sh"), string(script), 0o755)
	e.write(filepath.Join(e.root, ".claude-plugin", "plugin.json"), "{\n  \"name\": \"x\",\n  \"version\": \""+e.version+"\"\n}\n", 0o644)
	return e
}

func (e *ensureEnv) write(path, body string, mode os.FileMode) {
	e.t.Helper()
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		e.t.Fatal(err)
	}
}

// fakeAlp installs an `alp` reporting version into dir.
func (e *ensureEnv) fakeAlp(dir, version string) {
	e.t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		e.t.Fatal(err)
	}
	e.write(filepath.Join(dir, "alp"),
		"#!/bin/sh\nprintf '{\"version\":\""+version+"\",\"commit\":\"c\"}\\n'\n", 0o755)
}

// stubInstaller records its args and, unless fail, installs a matching alp.
func (e *ensureEnv) stubInstaller(fail bool) {
	e.t.Helper()
	body := "#!/bin/sh\necho \"$@\" >> '" + e.calls + "'\n"
	if fail {
		body += "echo 'get.sh: download failed' >&2\nexit 1\n"
	} else {
		body += "mkdir -p \"$ALP_INSTALL_DIR\"\nprintf '#!/bin/sh\\nprintf \"{\\\\\"version\\\\\":\\\\\"%s\\\\\"}\"\\n' \"$1\" > \"$ALP_INSTALL_DIR/alp\"\nchmod +x \"$ALP_INSTALL_DIR/alp\"\n"
	}
	e.write(e.getSh, body, 0o755)
}

func (e *ensureEnv) run() (stdout, stderr string, code int) {
	e.t.Helper()
	cmd := exec.Command("bash", filepath.Join(e.root, "scripts", "ensure-cli.sh"))
	cmd.Env = append([]string{
		"PATH=" + e.bin + ":/usr/bin:/bin",
		"HOME=" + e.home,
		"TMPDIR=" + e.home,
		"CLAUDE_PLUGIN_ROOT=" + e.root,
		"ALP_INSTALL_DIR=" + e.dest,
	}, e.extra...)
	if e.getSh != "" {
		cmd.Env = append(cmd.Env, "ALP_GET_SH="+e.getSh)
	}
	var out, errb strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		e.t.Fatal(err)
	}
	return out.String(), errb.String(), code
}

func (e *ensureEnv) installerCalls() string {
	b, _ := os.ReadFile(e.calls)
	return strings.TrimSpace(string(b))
}

func TestEnsureCLIMatchingVersionIsSilentAndOffline(t *testing.T) {
	e := newEnsureEnv(t)
	e.fakeAlp(e.bin, "v0.1.1")
	e.stubInstaller(false)
	stdout, stderr, code := e.run()
	if code != 0 || stdout != "" || stderr != "" {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if got := e.installerCalls(); got != "" {
		t.Fatalf("installer ran for a matching version: %q", got)
	}
}

func TestEnsureCLIDevBuildIsLeftAlone(t *testing.T) {
	e := newEnsureEnv(t)
	e.fakeAlp(e.bin, "dev")
	e.stubInstaller(false)
	if _, stderr, code := e.run(); code != 0 || stderr != "" || e.installerCalls() != "" {
		t.Fatalf("dev build was replaced: code=%d stderr=%q calls=%q", code, stderr, e.installerCalls())
	}
}

func TestEnsureCLIMismatchedVersionRunsPinnedInstaller(t *testing.T) {
	e := newEnsureEnv(t)
	e.fakeAlp(e.bin, "v0.1.0")
	e.stubInstaller(false)
	_, stderr, code := e.run()
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	if got := e.installerCalls(); got != "v0.1.1" {
		t.Fatalf("installer args = %q, want v0.1.1", got)
	}
	if !strings.Contains(stderr, "installed alp v0.1.1") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestEnsureCLIMissingAlpInstallsAndWarnsOnceAboutPath(t *testing.T) {
	e := newEnsureEnv(t)
	e.stubInstaller(false)
	stdout, stderr, code := e.run()
	if code != 0 || e.installerCalls() != "v0.1.1" {
		t.Fatalf("code=%d calls=%q stderr=%q", code, e.installerCalls(), stderr)
	}
	if !strings.Contains(stderr, "not on PATH") || !strings.Contains(stdout, "additionalContext") {
		t.Fatalf("expected a PATH warning: stdout=%q stderr=%q", stdout, stderr)
	}
	// The next session finds the install dir's alp (current) and stays quiet;
	// force a reinstall to prove the PATH warning is only given once.
	e.fakeAlp(e.dest, "v0.0.1")
	_, stderr, _ = e.run()
	if strings.Contains(stderr, "not on PATH") {
		t.Fatalf("PATH warning repeated: %q", stderr)
	}
}

func TestEnsureCLIInstallerFailureNeverFailsTheSession(t *testing.T) {
	e := newEnsureEnv(t)
	e.stubInstaller(true)
	stdout, stderr, code := e.run()
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	for _, out := range []string{stderr, stdout} {
		if !strings.Contains(out, "could not install alp v0.1.1") || !strings.Contains(out, "raw.githubusercontent.com/adams100111/agentic-learning-partner/v0.1.1/scripts/get.sh | bash -s -- v0.1.1") {
			t.Fatalf("missing failure message / pinned manual command in %q", out)
		}
	}
	if !strings.Contains(stdout, `"hookEventName":"SessionStart"`) {
		t.Fatalf("stdout is not hook context: %q", stdout)
	}
}

func TestEnsureCLIOptOut(t *testing.T) {
	e := newEnsureEnv(t)
	e.stubInstaller(false)
	e.extra = []string{"ALP_SKIP_CLI_INSTALL=1"}
	stdout, stderr, code := e.run()
	if code != 0 || stdout != "" || stderr != "" || e.installerCalls() != "" {
		t.Fatalf("opt-out ignored: code=%d out=%q err=%q calls=%q", code, stdout, stderr, e.installerCalls())
	}
}

func TestEnsureCLIConcurrentSessionsInstallOnce(t *testing.T) {
	e := newEnsureEnv(t)
	e.stubInstaller(false)
	done := make(chan struct{}, 3)
	for i := 0; i < 3; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			cmd := exec.Command("bash", filepath.Join(e.root, "scripts", "ensure-cli.sh"))
			cmd.Env = []string{"PATH=" + e.bin + ":/usr/bin:/bin", "HOME=" + e.home, "TMPDIR=" + e.home,
				"CLAUDE_PLUGIN_ROOT=" + e.root, "ALP_INSTALL_DIR=" + e.dest, "ALP_GET_SH=" + e.getSh}
			_ = cmd.Run()
		}()
	}
	for i := 0; i < 3; i++ {
		<-done
	}
	if n := len(strings.Fields(e.installerCalls())); n != 1 {
		t.Fatalf("installer ran %d times, want 1: %q", n, e.installerCalls())
	}
}

func TestPluginHookAndVersionsAreConsistent(t *testing.T) {
	root := repoRootDir(t)
	hooks, err := os.ReadFile(filepath.Join(root, "hooks", "hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"SessionStart"`, `${CLAUDE_PLUGIN_ROOT}/scripts/ensure-cli.sh`} {
		if !strings.Contains(string(hooks), want) {
			t.Errorf("hooks.json lacks %s", want)
		}
	}
	re := regexp.MustCompile(`"version"\s*:\s*"([^"]+)"`)
	var versions []string
	for _, p := range []string{"plugin.json", ".claude-plugin/plugin.json", ".claude-plugin/marketplace.json"} {
		b, err := os.ReadFile(filepath.Join(root, p))
		if err != nil {
			t.Fatal(err)
		}
		m := re.FindSubmatch(b)
		if m == nil {
			t.Fatalf("%s has no version", p)
		}
		versions = append(versions, string(m[1]))
	}
	if versions[0] != versions[1] || versions[1] != versions[2] {
		t.Fatalf("manifest versions disagree: %v", versions)
	}
}

// --- shadowing, downgrade, locking, JSON ------------------------------------

func TestEnsureCLIAcceptsMatchingInstallDirWithoutRedownloadAndWarnsOnceAboutShadowing(t *testing.T) {
	e := newEnsureEnv(t)
	e.fakeAlp(e.bin, "v0.1.0")  // older, earlier on PATH
	e.fakeAlp(e.dest, "v0.1.1") // already installed and current
	e.stubInstaller(false)
	stdout, stderr, code := e.run()
	if code != 0 || e.installerCalls() != "" {
		t.Fatalf("re-downloaded: code=%d calls=%q", code, e.installerCalls())
	}
	if !strings.Contains(stderr, "shadowed by "+filepath.Join(e.bin, "alp")+" (version 0.1.0)") || !strings.Contains(stdout, "additionalContext") {
		t.Fatalf("no shadow warning: stdout=%q stderr=%q", stdout, stderr)
	}
	if stdout, stderr, _ = e.run(); stdout != "" || stderr != "" {
		t.Fatalf("shadow warning repeated: stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestEnsureCLIWarnsWhenAFreshInstallIsShadowed(t *testing.T) {
	e := newEnsureEnv(t)
	e.fakeAlp(e.bin, "v0.1.0")
	e.stubInstaller(false)
	_, stderr, _ := e.run()
	if e.installerCalls() != "v0.1.1" || !strings.Contains(stderr, "shadowed by") {
		t.Fatalf("calls=%q stderr=%q", e.installerCalls(), stderr)
	}
}

func TestEnsureCLINeverDowngradesANewerCLI(t *testing.T) {
	for _, where := range []string{"path", "dest"} {
		t.Run(where, func(t *testing.T) {
			e := newEnsureEnv(t)
			if where == "path" {
				e.fakeAlp(e.bin, "v0.2.0")
			} else {
				e.fakeAlp(e.dest, "v0.10.0")
			}
			e.stubInstaller(false)
			stdout, stderr, code := e.run()
			if code != 0 || e.installerCalls() != "" {
				t.Fatalf("downgraded: code=%d calls=%q", code, e.installerCalls())
			}
			if !strings.Contains(stderr, "older than the installed alp") || !strings.Contains(stdout, "additionalContext") {
				t.Fatalf("no warning: stdout=%q stderr=%q", stdout, stderr)
			}
			if stdout, stderr, _ = e.run(); stdout != "" || stderr != "" {
				t.Fatalf("warning repeated: %q %q", stdout, stderr)
			}
		})
	}
}

func (e *ensureEnv) lockDir() string {
	return filepath.Join(e.home, ".cache", "alp", "ensure-cli.lock")
}

func TestEnsureCLIReclaimsAStaleLock(t *testing.T) {
	e := newEnsureEnv(t)
	e.stubInstaller(false)
	if err := os.MkdirAll(e.lockDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(e.lockDir(), old, old); err != nil {
		t.Fatal(err)
	}
	if _, stderr, code := e.run(); code != 0 || e.installerCalls() != "v0.1.1" {
		t.Fatalf("stale lock not reclaimed: code=%d calls=%q stderr=%q", code, e.installerCalls(), stderr)
	}
	if _, err := os.Stat(e.lockDir()); err == nil {
		t.Fatal("lock left behind")
	}
}

func TestEnsureCLIGivesUpOnALiveLockWithoutFailing(t *testing.T) {
	e := newEnsureEnv(t)
	e.stubInstaller(false)
	if err := os.MkdirAll(e.lockDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	e.extra = []string{"ALP_ENSURE_LOCK_WAIT=1"}
	stdout, stderr, code := e.run()
	if code != 0 || e.installerCalls() != "" || !strings.Contains(stderr, "another session is still installing") || !strings.Contains(stdout, "additionalContext") {
		t.Fatalf("code=%d calls=%q stdout=%q stderr=%q", code, e.installerCalls(), stdout, stderr)
	}
}

func TestEnsureCLILockLivesInAPrivateDir(t *testing.T) {
	e := newEnsureEnv(t)
	e.stubInstaller(false)
	e.run()
	fi, err := os.Stat(filepath.Join(e.home, ".cache", "alp"))
	if err != nil || fi.Mode().Perm() != 0o700 {
		t.Fatalf("lock dir: %v %v", fi, err)
	}
}

func TestEnsureCLIContextIsValidJSONWithControlCharacters(t *testing.T) {
	e := newEnsureEnv(t)
	e.write(e.getSh, "#!/bin/sh\nprintf 'bad\\ttab \"quoted\" back\\\\slash\\001ctl\\n' >&2\nexit 1\n", 0o755)
	stdout, _, code := e.run()
	if code != 0 {
		t.Fatalf("code=%d", code)
	}
	var v struct {
		HookSpecificOutput struct{ AdditionalContext string } `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(stdout), &v); err != nil {
		t.Fatalf("invalid JSON %q: %v", stdout, err)
	}
	if !strings.Contains(v.HookSpecificOutput.AdditionalContext, `"quoted"`) {
		t.Fatalf("context = %q", v.HookSpecificOutput.AdditionalContext)
	}
}

func TestEnsureCLIUnpublishedReleaseIsQuietAndRetried(t *testing.T) {
	e := newEnsureEnv(t)
	e.write(e.getSh, "#!/bin/sh\necho \"$@\" >> '"+e.calls+"'\necho 'get.sh: release not published (404)' >&2\nexit 44\n", 0o755)
	for i := 0; i < 2; i++ {
		stdout, stderr, code := e.run()
		if code != 0 || stdout != "" || !strings.Contains(stderr, "not published yet; will retry next session") {
			t.Fatalf("run %d: code=%d stdout=%q stderr=%q", i, code, stdout, stderr)
		}
	}
	if n := len(strings.Fields(e.installerCalls())); n != 2 {
		t.Fatalf("installer attempts = %d, want 2 (retry next session)", n)
	}
}

// --- the real get.sh against a fake release --------------------------------

type fakeRelease struct {
	srv      *httptest.Server
	requests []string
}

// newFakeRelease serves /v0.1.1/{asset,SHA256SUMS}. mode: "good", "badsum", "404".
func newFakeRelease(t *testing.T, mode string) *fakeRelease {
	t.Helper()
	var buf strings.Builder
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	body := "#!/bin/sh\nprintf '{\"version\":\"v0.1.1\"}\\n'\n"
	_ = tw.WriteHeader(&tar.Header{Name: "alp", Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg})
	_, _ = tw.Write([]byte(body))
	_ = tw.Close()
	_ = gz.Close()
	archive := buf.String()
	asset := fmt.Sprintf("alp_v0.1.1_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	sum := fmt.Sprintf("%x", sha256.Sum256([]byte(archive)))
	if mode == "badsum" {
		sum = strings.Repeat("0", 64)
	}
	r := &fakeRelease{}
	r.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.requests = append(r.requests, req.URL.Path)
		switch {
		case mode == "404":
			http.NotFound(w, req)
		case req.URL.Path == "/v0.1.1/"+asset:
			_, _ = w.Write([]byte(archive))
		case req.URL.Path == "/v0.1.1/SHA256SUMS":
			fmt.Fprintf(w, "%s  %s\n", sum, asset)
		default:
			http.NotFound(w, req)
		}
	}))
	t.Cleanup(r.srv.Close)
	return r
}

func (e *ensureEnv) useRealGetSh(base string) {
	e.t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRootDir(e.t), "scripts", "get.sh"))
	if err != nil {
		e.t.Fatal(err)
	}
	e.write(filepath.Join(e.root, "scripts", "get.sh"), string(b), 0o755)
	e.getSh = ""
	e.extra = append(e.extra, "ALP_GET_TEST_BASE_URL="+base)
}

func TestRealGetShInstallsFromAFakeRelease(t *testing.T) {
	rel := newFakeRelease(t, "good")
	e := newEnsureEnv(t)
	e.fakeAlp(e.bin, "v0.1.0")
	e.useRealGetSh(rel.srv.URL)
	_, stderr, code := e.run()
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	out, err := exec.Command(filepath.Join(e.dest, "alp"), "version", "--json").Output()
	if err != nil || !strings.Contains(string(out), `"v0.1.1"`) {
		t.Fatalf("installed alp: %q %v (stderr %q)", out, err, stderr)
	}
	if !strings.Contains(stderr, "installed alp v0.1.1") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestRealGetShRefusesAChecksumMismatch(t *testing.T) {
	rel := newFakeRelease(t, "badsum")
	e := newEnsureEnv(t)
	e.useRealGetSh(rel.srv.URL)
	stdout, stderr, code := e.run()
	if code != 0 {
		t.Fatalf("code=%d", code)
	}
	if _, err := os.Stat(filepath.Join(e.dest, "alp")); err == nil {
		t.Fatal("alp installed despite a checksum mismatch")
	}
	if !strings.Contains(stderr, "checksum mismatch") || !strings.Contains(stdout, "checksum mismatch") {
		t.Fatalf("stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestRealGetShTreats404AsUnpublished(t *testing.T) {
	rel := newFakeRelease(t, "404")
	e := newEnsureEnv(t)
	e.useRealGetSh(rel.srv.URL)
	stdout, stderr, code := e.run()
	if code != 0 || stdout != "" || !strings.Contains(stderr, "not published yet") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(e.dest, "alp")); err == nil {
		t.Fatal("alp installed from a 404")
	}
	// get.sh itself reports the distinct status.
	cmd := exec.Command("bash", filepath.Join(e.root, "scripts", "get.sh"), "v0.1.1")
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + e.home, "ALP_INSTALL_DIR=" + e.dest, "ALP_GET_TEST_BASE_URL=" + rel.srv.URL}
	err := cmd.Run()
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 44 {
		t.Fatalf("get.sh exit = %v, want 44", err)
	}
}
