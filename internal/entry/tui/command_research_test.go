package tui

import (
	"strings"
	"testing"
)

func TestParseSlashCommandKeepsQuotedPaths(t *testing.T) {
	cmd, ok := parseSlashCommand(`/d-research fusion drive file="D:\Docs\research notes.md" max=5`)
	if !ok {
		t.Fatal("expected slash command")
	}
	if cmd.name != "d-research" {
		t.Fatalf("name = %q", cmd.name)
	}
	if len(cmd.args) != 4 {
		t.Fatalf("args = %#v", cmd.args)
	}
	if cmd.args[2] != `file=D:\Docs\research notes.md` {
		t.Fatalf("quoted path was not preserved: %#v", cmd.args)
	}
}

func TestParseSlashCommandAcceptsTabSeparator(t *testing.T) {
	cmd, ok := parseSlashCommand("/d-research\tfusion file=notes.md")
	if !ok || cmd.name != "d-research" {
		t.Fatalf("unexpected command: ok=%v cmd=%#v", ok, cmd)
	}
	if len(cmd.args) != 2 || cmd.args[0] != "fusion" {
		t.Fatalf("unexpected args: %#v", cmd.args)
	}
}

func TestParseDResearchPreservesCommasAndRepeatedFiles(t *testing.T) {
	parsed, err := parseDResearchCommandArgs([]string{
		"orbital", "mechanics",
		`file=D:\Docs\alpha,beta.md`,
		`file=D:\Docs\second.md`,
		"url=https://example.com/search?q=a,b",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.FilePaths) != 2 || parsed.FilePaths[0] != `D:\Docs\alpha,beta.md` {
		t.Fatalf("file paths were corrupted: %#v", parsed.FilePaths)
	}
	if len(parsed.SourceURLs) != 1 || parsed.SourceURLs[0] != "https://example.com/search?q=a,b" {
		t.Fatalf("source URL was corrupted: %#v", parsed.SourceURLs)
	}
}

func TestQuotedImportPathStillParses(t *testing.T) {
	cmd, ok := parseSlashCommand(`/import "D:\Books\source novel.txt" from=2`)
	if !ok {
		t.Fatal("expected slash command")
	}
	opts, err := parseImportArgs(cmd.args)
	if err != nil {
		t.Fatal(err)
	}
	if opts.SourcePath != `D:\Books\source novel.txt` || opts.ResumeFrom != 2 {
		t.Fatalf("unexpected import options: %#v", opts)
	}
}

func TestBuildDResearchCommandPrompt(t *testing.T) {
	prompt, err := buildDResearchCommandPrompt([]string{
		"fusion", "drive",
		`file=D:\Docs\notes.md`,
		"url=https://example.com/paper",
		"freshness=2025-01-01",
		"max=4",
	})
	if err != nil {
		t.Fatalf("buildDResearchCommandPrompt failed: %v", err)
	}
	for _, want := range []string{
		"[D-RESEARCH]",
		"Mục tiêu nghiên cứu: fusion drive",
		`D:\Docs\notes.md`,
		"https://example.com/paper",
		"Số nguồn tối đa: 4",
		"research_brief",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}
