package research

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDedupeURLs(t *testing.T) {
	urls := []string{
		"https://example.com/a",
		"https://example.com/b",
		"https://example.com/a",     // duplicate
		"  https://example.com/c  ", // whitespace
		"",                          // empty
	}
	result := dedupeURLs(urls)
	if len(result) != 3 {
		t.Fatalf("expected 3 unique URLs, got %d: %v", len(result), result)
	}
}

func TestPrioritizeSourceURLs(t *testing.T) {
	hits := prioritizeSourceURLs(
		[]string{"https://primary.example/paper"},
		[]searchHit{
			{URL: "https://search.example/result"},
			{URL: "https://primary.example/paper", Title: "duplicate discovered result"},
		},
	)
	if len(hits) != 2 {
		t.Fatalf("expected two unique hits, got %#v", hits)
	}
	if hits[0].URL != "https://primary.example/paper" {
		t.Fatalf("explicit source URL must be probed first, got %#v", hits)
	}
}

func TestFanoutQueries(t *testing.T) {
	cfg := Config{MaxQueries: 8}
	plugin := &PluginInfo{ID: "d-research", Path: t.TempDir()}
	runner := NewRunner(cfg, plugin)

	req := Request{
		Goal:      "nuclear fusion reactor design",
		Freshness: "2024-01-01",
	}

	queries := runner.fanoutQueries(req)
	if len(queries) == 0 {
		t.Fatal("expected at least 1 query")
	}

	// Should include base goal
	if queries[0] != "nuclear fusion reactor design" {
		t.Fatalf("expected first query to be goal, got %q", queries[0])
	}

	// Should include contradiction query
	hasContradiction := false
	for _, q := range queries {
		if contains(q, "limitations") || contains(q, "criticism") {
			hasContradiction = true
			break
		}
	}
	if !hasContradiction {
		t.Error("expected contradiction/limitation query in fanout")
	}

	// Should include freshness query
	hasFreshness := false
	for _, q := range queries {
		if contains(q, "after:2024-01-01") {
			hasFreshness = true
			break
		}
	}
	if !hasFreshness {
		t.Error("expected freshness query in fanout")
	}
}

func TestFanoutQueriesWithExplicit(t *testing.T) {
	cfg := Config{MaxQueries: 8}
	plugin := &PluginInfo{ID: "d-research", Path: t.TempDir()}
	runner := NewRunner(cfg, plugin)

	req := Request{
		Goal:    "test",
		Queries: []string{"explicit query 1", "explicit query 2"},
	}

	queries := runner.fanoutQueries(req)
	if len(queries) != 2 {
		t.Fatalf("expected 2 explicit queries, got %d", len(queries))
	}
	if queries[0] != "explicit query 1" {
		t.Fatalf("expected explicit query 1, got %q", queries[0])
	}
}

func TestBuildCoverage(t *testing.T) {
	sources := []SourceEvidence{
		{URL: "https://example.com/a", AccessStatus: "accessible"},
		{URL: "https://example.com/b", AccessStatus: "blocked"},
		{URL: "https://other.com/c", AccessStatus: "partial"},
	}
	c := buildCoverage(sources)
	if c.TotalSources != 3 {
		t.Fatalf("expected 3 total sources, got %d", c.TotalSources)
	}
	if c.Accessed != 2 {
		t.Fatalf("expected 2 accessed, got %d", c.Accessed)
	}
	if c.Blocked != 1 {
		t.Fatalf("expected 1 blocked source, got %d", c.Blocked)
	}
	if len(c.Domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(c.Domains))
	}
}

func TestIdentifyContradictions(t *testing.T) {
	sources := []SourceEvidence{
		{URL: "https://example.com/a", Snippet: "Fusion is 10 years away", Contradiction: true},
		{URL: "https://example.com/b", Snippet: "Fusion is 30 years away", Contradiction: true},
		{URL: "https://example.com/c", Snippet: "No contradiction here", Contradiction: false},
	}

	contradictions := identifyContradictions(sources)
	if len(contradictions) != 1 {
		t.Fatalf("expected 1 contradiction pair, got %d", len(contradictions))
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://example.com/page", "example.com"},
		{"http://sub.example.com/path", "sub.example.com"},
		{"https://example.com", "example.com"},
		{"not-a-url", "not-a-url"},
	}
	for _, tt := range tests {
		got := extractDomain(tt.url)
		if got != tt.expected {
			t.Errorf("extractDomain(%q) = %q, want %q", tt.url, got, tt.expected)
		}
	}
}

func TestReportToCompact(t *testing.T) {
	report := &Report{
		ID:       "test123",
		Goal:     "Test research",
		Sources:  []SourceEvidence{{URL: "https://example.com/a", Title: "Example", Snippet: "Useful fact", AccessStatus: "accessible"}},
		Blockers: []Blocker{{Reason: "test blocker", Severity: "warning"}},
		Files:    map[string]string{"report": "report.json"},
	}

	compact := report.ToCompact()
	if compact.ID != "test123" {
		t.Fatalf("expected id test123, got %q", compact.ID)
	}
	if compact.Sources != 1 {
		t.Fatalf("expected 1 source, got %d", compact.Sources)
	}
	if compact.Blockers != 1 {
		t.Fatalf("expected 1 blocker, got %d", compact.Blockers)
	}
	if len(compact.Highlights) != 1 || compact.Highlights[0].Snippet != "Useful fact" {
		t.Fatalf("expected compact source highlight, got %#v", compact.Highlights)
	}
	if compact.Files["report"] != "report.json" {
		t.Fatalf("expected compact report file path, got %#v", compact.Files)
	}
}

func TestRunnerIncludesLocalFilesAsEvidence(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "fusion-notes.md")
	if err := os.WriteFile(sourcePath, []byte("Fusion drives need heat rejection and radiation shielding."), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Enabled:    true,
		MaxSources: 3,
		Browser:    BrowserConfig{Enabled: false},
	}
	plugin := &PluginInfo{ID: "d-research", Path: t.TempDir(), Scripts: map[string]string{}}
	runner := NewRunner(cfg, plugin)

	report, err := runner.Run(context.Background(), Request{
		Goal:      "fusion drive",
		Queries:   []string{},
		FilePaths: []string{sourcePath},
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(report.Sources) != 1 {
		t.Fatalf("expected one local source, got %d", len(report.Sources))
	}
	source := report.Sources[0]
	if source.AccessMethod != "local_file" {
		t.Fatalf("expected local_file access, got %q", source.AccessMethod)
	}
	if !strings.Contains(source.Snippet, "heat rejection") {
		t.Fatalf("expected local snippet, got %q", source.Snippet)
	}
}

func TestLoadLocalFileRejectsBinaryInput(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "reference.pdf")
	if err := os.WriteFile(sourcePath, []byte("%PDF-1.7\x00binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := NewRunner(Config{}, &PluginInfo{ID: "d-research", Path: t.TempDir()})
	source := runner.loadLocalFile(sourcePath)
	if source.AccessStatus != "unsupported_format" {
		t.Fatalf("expected unsupported format, got %#v", source)
	}
	if !isBlockedStatus(source.AccessStatus) {
		t.Fatal("unsupported format must be reported as a blocker")
	}
}

func TestLoadLocalFileExtractsDOCX(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "reference.docx")
	file, err := os.Create(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	entry, err := archive.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte(`<w:document xmlns:w="urn:test"><w:body><w:p><w:r><w:t>Fusion shielding evidence</w:t></w:r></w:p></w:body></w:document>`)); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	runner := NewRunner(Config{}, &PluginInfo{ID: "d-research", Path: t.TempDir()})
	source := runner.loadLocalFile(sourcePath)
	if source.AccessStatus != "accessible" || !strings.Contains(source.ExtractedContent, "Fusion shielding evidence") {
		t.Fatalf("unexpected DOCX evidence: %#v", source)
	}
}

func TestRunnerWithFakeScript(t *testing.T) {
	dir := t.TempDir()
	writePluginManifest(t, dir, "d-research")

	// Create a fake web_search script
	scriptsDir := filepath.Join(dir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a simple batch script that outputs JSON
	scriptPath := filepath.Join(scriptsDir, "web_search.mjs")
	scriptContent := `const fs = require('fs');
const idx = process.argv.indexOf('--out');
if (idx >= 0) {
  fs.writeFileSync(process.argv[idx+1], JSON.stringify([
    {
      url: "https://example.com/result",
      title: "Example result",
      snippet: "A short search result snippet",
      source_engine: "fake"
    }
  ]));
}
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Enabled:            true,
		PluginPath:         dir,
		MaxQueries:         2,
		MaxResultsPerQuery: 5,
		MaxSources:         5,
		Browser:            BrowserConfig{Enabled: false},
	}

	plugin, err := ResolvePlugin(cfg)
	if err != nil {
		t.Fatalf("ResolvePlugin failed: %v", err)
	}

	runner := NewRunner(cfg, plugin)
	req := Request{
		Goal:    "test research",
		Queries: []string{"test query"},
	}

	report, err := runner.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if report == nil {
		t.Fatal("expected non-nil report")
	}
	if report.Goal != "test research" {
		t.Fatalf("expected goal 'test research', got %q", report.Goal)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
