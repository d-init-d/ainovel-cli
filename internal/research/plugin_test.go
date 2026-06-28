package research

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolvePluginExplicitPath(t *testing.T) {
	dir := t.TempDir()
	writePluginManifest(t, dir, "d-research")

	cfg := Config{PluginPath: dir}
	info, err := ResolvePlugin(cfg)
	if err != nil {
		t.Fatalf("ResolvePlugin with explicit path failed: %v", err)
	}
	if info.ID != "d-research" {
		t.Fatalf("expected id d-research, got %q", info.ID)
	}
}

func TestResolvePluginNotFound(t *testing.T) {
	cfg := Config{PluginPath: filepath.Join(os.TempDir(), "nonexistent-plugin-dir-12345")}
	_, err := ResolvePlugin(cfg)
	if err == nil {
		t.Fatal("expected error for nonexistent plugin path")
	}
}

func TestResolvePluginWithWarningsMissingScripts(t *testing.T) {
	dir := t.TempDir()
	writePluginManifest(t, dir, "d-research")

	cfg := Config{PluginPath: dir}
	info, warnings := ResolvePluginWithWarnings(cfg)
	if info == nil {
		t.Fatal("expected plugin info even with warnings")
	}
	// Plugin should be found even without scripts
	if info.ID != "d-research" {
		t.Fatalf("expected id d-research, got %q", info.ID)
	}
	if !warningsContain(warnings, "web_search script not found") {
		t.Fatalf("expected missing web_search warning, got %v", warnings)
	}
	if !warningsContain(warnings, "browser_probe script not found") {
		t.Fatalf("expected missing browser_probe warning, got %v", warnings)
	}
}

func TestPluginCapabilities(t *testing.T) {
	dir := t.TempDir()
	writePluginManifest(t, dir, "d-research")

	cfg := Config{PluginPath: dir}
	info, err := ResolvePlugin(cfg)
	if err != nil {
		t.Fatalf("ResolvePlugin failed: %v", err)
	}

	if !info.HasCapability("web_search") {
		t.Error("expected web_search capability")
	}
	if info.HasCapability("nonexistent") {
		t.Error("expected no nonexistent capability")
	}
}

func TestPluginPathKey(t *testing.T) {
	a := PluginPathKey("C:/Users/test/.codex/skills/d-research")
	b := PluginPathKey("c:\\users\\test\\.codex\\skills\\d-research")
	if a != b {
		t.Fatalf("expected normalized keys to match: %q vs %q", a, b)
	}
}

func writePluginManifest(t *testing.T, dir, id string) {
	t.Helper()
	manifest := map[string]any{
		"schema_version": 1,
		"id":             id,
		"display_name":   "Test Plugin",
		"version":        "1.0.0",
		"capabilities":   []string{"web_search", "browser_probe", "evidence_ledger"},
		"scripts": map[string]string{
			"web_search":    "scripts/web_search.mjs",
			"browser_probe": "scripts/playwright_probe.mjs",
		},
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ainovel-plugin.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func warningsContain(warnings []string, want string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, want) {
			return true
		}
	}
	return false
}
