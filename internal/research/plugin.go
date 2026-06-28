package research

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// PluginManifest describes a d-research plugin bundle.
type PluginManifest struct {
	SchemaVersion int               `json:"schema_version"`
	ID            string            `json:"id"`
	DisplayName   string            `json:"display_name"`
	Version       string            `json:"version"`
	Capabilities  []string          `json:"capabilities"`
	Scripts       map[string]string `json:"scripts"`
}

// PluginInfo holds resolved plugin information.
type PluginInfo struct {
	ID               string
	Path             string
	Manifest         PluginManifest
	Capabilities     map[string]bool
	Scripts          map[string]string
	Warnings         []string
	BrowserAvailable bool
}

// ResolvePlugin finds the d-research plugin by checking paths in priority order:
// 1. Explicitly configured plugin_path
// 2. CWD-local plugins/d-research/
// 3. Executable-adjacent plugins/d-research/
// 4. %USERPROFILE%\.codex\skills\d-research
// 5. %USERPROFILE%\.agents\skills\d-research
func ResolvePlugin(cfg Config) (*PluginInfo, error) {
	candidates := candidatePaths(cfg.PluginPath)
	allowRepoFallback := strings.TrimSpace(cfg.PluginPath) == ""

	for _, dir := range candidates {
		info, err := loadPlugin(dir, allowRepoFallback && samePath(dir, filepath.Join("plugins", "d-research")))
		if err == nil {
			return info, nil
		}
	}

	return nil, fmt.Errorf("d-research plugin not found at any candidate path")
}

// ResolvePluginWithWarnings resolves the plugin and returns warnings instead of errors for missing capabilities.
func ResolvePluginWithWarnings(cfg Config) (*PluginInfo, []string) {
	var warnings []string

	candidates := candidatePaths(cfg.PluginPath)
	allowRepoFallback := strings.TrimSpace(cfg.PluginPath) == ""
	for _, dir := range candidates {
		info, err := loadPlugin(dir, allowRepoFallback && samePath(dir, filepath.Join("plugins", "d-research")))
		if err == nil {
			info.Warnings = checkCapabilities(info)
			return info, info.Warnings
		}
	}

	warnings = append(warnings, "d-research plugin not found at any candidate path")
	return nil, warnings
}

func candidatePaths(explicit string) []string {
	var paths []string

	if explicit = strings.TrimSpace(explicit); explicit != "" {
		paths = append(paths, explicit)
		return paths
	}

	paths = append(paths, filepath.Join("plugins", "d-research"))
	if executable, err := os.Executable(); err == nil {
		if executable, err = filepath.EvalSymlinks(executable); err == nil {
			paths = append(paths, filepath.Join(filepath.Dir(executable), "plugins", "d-research"))
		}
	}

	if home := userHome(); home != "" {
		paths = append(paths,
			filepath.Join(home, ".codex", "skills", "d-research"),
			filepath.Join(home, ".agents", "skills", "d-research"),
		)
	}

	seen := make(map[string]bool, len(paths))
	unique := paths[:0]
	for _, path := range paths {
		key := PluginPathKey(path)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, path)
	}
	return unique
}

func userHome() string {
	if runtime.GOOS == "windows" {
		if h := os.Getenv("USERPROFILE"); h != "" {
			return h
		}
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

func loadPlugin(dir string, allowExternalScriptFallback bool) (*PluginInfo, error) {
	manifestPath := filepath.Join(dir, "ainovel-plugin.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var manifest PluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest at %s: %w", manifestPath, err)
	}

	if manifest.ID != "d-research" {
		return nil, fmt.Errorf("plugin at %s has id %q, expected d-research", dir, manifest.ID)
	}

	capMap := make(map[string]bool)
	for _, c := range manifest.Capabilities {
		capMap[c] = true
	}

	scripts := make(map[string]string)
	for name, relPath := range manifest.Scripts {
		fullPath := filepath.Join(dir, relPath)
		if _, err := os.Stat(fullPath); err == nil {
			scripts[name] = fullPath
		}
	}

	// The repo-local bundle may be a thin adapter. When auto-resolving it, borrow
	// script implementations from the user's full d-research skill installation.
	// Explicit plugin paths stay authoritative so misconfigured plugins fail loud.
	if len(scripts) == 0 && allowExternalScriptFallback && !isExternalPlugin(dir) {
		external := findExternalScripts()
		for name, path := range external {
			scripts[name] = path
		}
	}

	return &PluginInfo{
		ID:               manifest.ID,
		Path:             dir,
		Manifest:         manifest,
		Capabilities:     capMap,
		Scripts:          scripts,
		BrowserAvailable: true,
	}, nil
}

func isExternalPlugin(dir string) bool {
	if home := userHome(); home != "" {
		codexPath := filepath.Join(home, ".codex", "skills", "d-research")
		agentsPath := filepath.Join(home, ".agents", "skills", "d-research")
		return samePath(dir, codexPath) || samePath(dir, agentsPath)
	}
	return false
}

func findExternalScripts() map[string]string {
	scripts := make(map[string]string)
	candidates := []string{
		"scripts/web_search.mjs",
		"scripts/playwright_probe.mjs",
		"scripts/playwright_extract.mjs",
		"scripts/playwright_crawl.mjs",
	}

	scriptNames := []string{"web_search", "browser_probe", "browser_extract", "browser_crawl"}

	for _, dir := range candidatePaths("") {
		if samePath(dir, filepath.Join("plugins", "d-research")) {
			continue
		}
		for i, rel := range candidates {
			if _, ok := scripts[scriptNames[i]]; ok {
				continue
			}
			fullPath := filepath.Join(dir, rel)
			if _, err := os.Stat(fullPath); err == nil {
				scripts[scriptNames[i]] = fullPath
			}
		}
		if len(scripts) == len(scriptNames) {
			break
		}
	}
	return scripts
}

func samePath(a, b string) bool {
	return PluginPathKey(a) == PluginPathKey(b)
}

func checkCapabilities(info *PluginInfo) []string {
	var warnings []string
	nodeAvailable := true

	if _, ok := info.Scripts["web_search"]; !ok {
		warnings = append(warnings, "web_search script not found")
	}
	if _, ok := info.Scripts["browser_probe"]; !ok {
		warnings = append(warnings, "browser_probe script not found (Playwright may not be installed)")
	}
	if _, ok := info.Scripts["browser_extract"]; !ok {
		warnings = append(warnings, "browser_extract script not found")
	}

	// Check if node is available
	if _, err := exec.LookPath("node"); err != nil {
		warnings = append(warnings, "node.js not found in PATH; research scripts require Node.js >= 18")
		nodeAvailable = false
		info.BrowserAvailable = false
	}

	if nodeAvailable && (info.HasScript("browser_probe") || info.HasScript("browser_extract") || info.HasScript("browser_crawl")) {
		if err := canImportNodePackage(scriptPackageRoot(info), "playwright"); err != nil {
			warnings = append(warnings, "playwright package not found for d-research browser scripts; run npm install in the plugin directory and npx playwright install chromium")
			info.BrowserAvailable = false
		}
	}

	return warnings
}

func scriptPackageRoot(info *PluginInfo) string {
	for _, name := range []string{"browser_probe", "browser_extract", "browser_crawl", "web_search"} {
		if scriptPath := info.ScriptPath(name); scriptPath != "" {
			return filepath.Dir(filepath.Dir(scriptPath))
		}
	}
	return info.Path
}

func canImportNodePackage(dir, pkg string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "node", "-e", fmt.Sprintf("import(%q).then(()=>process.exit(0)).catch(()=>process.exit(1))", pkg))
	cmd.Dir = dir
	return cmd.Run()
}

// HasCapability checks if the plugin has a specific capability.
func (p *PluginInfo) HasCapability(name string) bool {
	if p == nil {
		return false
	}
	return p.Capabilities[name]
}

// HasScript checks if a specific script is available.
func (p *PluginInfo) HasScript(name string) bool {
	if p == nil {
		return false
	}
	_, ok := p.Scripts[name]
	return ok
}

// ScriptPath returns the full path to a named script.
func (p *PluginInfo) ScriptPath(name string) string {
	if p == nil {
		return ""
	}
	return p.Scripts[name]
}

// PluginPathKey returns a normalized key for the plugin path (for dedup).
func PluginPathKey(path string) string {
	return strings.ToLower(filepath.Clean(path))
}
