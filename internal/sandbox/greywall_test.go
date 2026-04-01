package sandbox

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildSettings_ProjectRoot(t *testing.T) {
	cfg := DefaultSandboxConfig()
	s := buildSettings(
		"/home/user/src/proj-feat",
		"/home/user/src/proj", // different from worktree → must be in allowRead
		"/home/user",
		cfg,
	)
	if !containsStr(s.Filesystem.AllowRead, "/home/user/src/proj") {
		t.Error("project root must be in allowRead")
	}
}

func TestBuildSettings_ProjectRootEqualsWorktree(t *testing.T) {
	cfg := DefaultSandboxConfig()
	s := buildSettings("/home/user/src/proj", "/home/user/src/proj", "/home/user", cfg)
	// When they're the same, no duplicate entry needed (greywall adds CWD automatically)
	for _, p := range s.Filesystem.AllowRead {
		if p == "/home/user/src/proj" {
			t.Error("no explicit allowRead entry needed when projectRoot == worktree")
		}
	}
}

func TestBuildSettings_ContextFiles(t *testing.T) {
	root := t.TempDir()
	mid := filepath.Join(root, "mid")
	wt := filepath.Join(mid, "project-feat")
	if err := os.MkdirAll(wt, 0o755); err != nil {
		t.Fatal(err)
	}

	// Place AGENTS.md at two ancestor levels
	agentsMid := filepath.Join(mid, "AGENTS.md")
	agentsRoot := filepath.Join(root, "AGENTS.md")
	os.WriteFile(agentsMid, []byte("# mid"), 0o644)
	os.WriteFile(agentsRoot, []byte("# root"), 0o644)

	cfg := DefaultSandboxConfig()
	s := buildSettings(wt, wt, root, cfg) // homeDir = root, stops before root

	if !containsStr(s.Filesystem.AllowRead, agentsMid) {
		t.Errorf("expected AGENTS.md at mid level in allowRead, got %v", s.Filesystem.AllowRead)
	}
	// root == homeDir so scan stops before it
	if containsStr(s.Filesystem.AllowRead, agentsRoot) {
		t.Error("should not scan homeDir itself")
	}
}

func TestBuildSettings_NoContextFiles(t *testing.T) {
	cfg := DefaultSandboxConfig()
	cfg.ContextFiles = []string{}
	s := buildSettings("/home/user/src/feat", "/home/user/src/proj", "/home/user", cfg)
	// Should still have project root, but no context file entries
	if !containsStr(s.Filesystem.AllowRead, "/home/user/src/proj") {
		t.Error("project root should still be present")
	}
}

func TestBuildSettings_ExtraPaths(t *testing.T) {
	cfg := DefaultSandboxConfig()
	cfg.ExtraROPaths = []string{"/data/shared", "~/.npmrc"}
	cfg.ExtraRWPaths = []string{"/var/run/docker.sock"}
	cfg.ExtraDenyRead = []string{"~/.vault-token"}

	s := buildSettings("/home/user/src/feat", "/home/user/src/proj", "/home/user", cfg)

	if !containsStr(s.Filesystem.AllowRead, "/data/shared") {
		t.Error("extra ro path missing from allowRead")
	}
	if !containsStr(s.Filesystem.AllowRead, "~/.npmrc") {
		t.Error("extra ro path missing from allowRead")
	}
	if !containsStr(s.Filesystem.AllowWrite, "/var/run/docker.sock") {
		t.Error("extra rw path missing from allowWrite")
	}
	if !containsStr(s.Filesystem.DenyRead, "~/.vault-token") {
		t.Error("extra deny path missing from denyRead")
	}
}

func TestWriteSettingsFile(t *testing.T) {
	cfg := DefaultSandboxConfig()
	cfg.ExtraROPaths = []string{"/data/shared"}
	s := buildSettings("/home/user/src/feat", "/home/user/src/proj", "/home/user", cfg)

	path, err := writeSettingsFile(s)
	if err != nil {
		t.Fatalf("writeSettingsFile: %v", err)
	}
	defer os.Remove(path)

	if !strings.HasSuffix(path, ".json") {
		t.Errorf("expected .json extension, got %q", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading settings file: %v", err)
	}

	var parsed greywallSettings
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("parsing settings file: %v", err)
	}
	if !containsStr(parsed.Filesystem.AllowRead, "/home/user/src/proj") {
		t.Error("project root missing from written settings file")
	}
}

func TestScanContextFiles(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a")
	b := filepath.Join(a, "b")
	wt := filepath.Join(b, "project-feat")
	os.MkdirAll(wt, 0o755)

	// AGENTS.md at level b, CLAUDE.md at level a
	os.WriteFile(filepath.Join(b, "AGENTS.md"), nil, 0o644)
	os.WriteFile(filepath.Join(a, "CLAUDE.md"), nil, 0o644)

	names := []string{"AGENTS.md", "CLAUDE.md"}
	found := scanContextFiles(wt, root, names)

	if !containsStr(found, filepath.Join(b, "AGENTS.md")) {
		t.Errorf("expected AGENTS.md in b, got %v", found)
	}
	if !containsStr(found, filepath.Join(a, "CLAUDE.md")) {
		t.Errorf("expected CLAUDE.md in a, got %v", found)
	}
	// root is homeDir — must not be scanned
	for _, p := range found {
		if filepath.Dir(p) == root {
			t.Errorf("should not scan homeDir itself, found %q", p)
		}
	}
}

func TestKnownProfiles(t *testing.T) {
	for _, tool := range []string{"claude", "pi", "opencode", "aider"} {
		if !knownProfiles[tool] {
			t.Errorf("expected %q in knownProfiles", tool)
		}
	}
}

func containsStr(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
