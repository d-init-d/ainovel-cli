package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/voocel/ainovel-cli/internal/research"
	"github.com/voocel/ainovel-cli/internal/store"
)

func TestResearchBriefToolExecuteSavesCompactReport(t *testing.T) {
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	scriptDir := t.TempDir()
	scriptPath := filepath.Join(scriptDir, "web_search.mjs")
	script := `import { writeFileSync } from 'node:fs';
const outIndex = process.argv.indexOf('--out');
if (outIndex < 0) {
  throw new Error('missing --out');
}
writeFileSync(process.argv[outIndex + 1], JSON.stringify([
  {
    url: 'https://example.com/fusion',
    title: 'Fusion constraints',
    snippet: 'Fusion drives need heat rejection and radiation shielding.',
    source_engine: 'fake'
  }
]));
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write fake search script: %v", err)
	}

	runner := research.NewRunner(
		research.Config{
			Enabled:            true,
			MaxQueries:         1,
			MaxResultsPerQuery: 2,
			MaxSources:         2,
			Timeout:            5 * time.Second,
			Browser:            research.BrowserConfig{Enabled: false},
		},
		&research.PluginInfo{
			ID:      "d-research",
			Path:    scriptDir,
			Scripts: map[string]string{"web_search": scriptPath},
		},
	)

	tool := NewResearchBriefTool(st, runner)
	out, err := tool.Execute(context.Background(), json.RawMessage(`{"goal":"fusion drive","queries":["fusion drive"]}`))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result["sources"] != float64(1) {
		t.Fatalf("expected 1 source, got %#v", result["sources"])
	}

	var compact map[string]any
	if err := st.Research.LoadLatestCompactReport(&compact); err != nil {
		t.Fatalf("LoadLatestCompactReport failed: %v", err)
	}
	if compact["goal"] != "fusion drive" {
		t.Fatalf("expected saved goal, got %#v", compact["goal"])
	}
	highlights, ok := compact["highlights"].([]any)
	if !ok || len(highlights) != 1 {
		t.Fatalf("expected one compact highlight, got %#v", compact["highlights"])
	}
}
