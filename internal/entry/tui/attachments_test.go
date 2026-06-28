package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/document"
)

func TestQueueAttachmentsSupportsReferenceFormats(t *testing.T) {
	dir := t.TempDir()
	var paths []string
	for _, name := range []string{"notes.txt", "world.md", "source.markdown", "reference.docx"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}

	queued, err := queueAttachments(nil, paths)
	if err != nil {
		t.Fatal(err)
	}
	if len(queued) != len(paths) {
		t.Fatalf("queued = %#v", queued)
	}
}

func TestQueueAttachmentsRejectsLegacyDOC(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.doc")
	if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := queueAttachments(nil, []string{path}); err == nil {
		t.Fatal("expected unsupported .doc error")
	}
}

func TestBuildPromptWithAttachmentsMarksDataUntrusted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "physics.md")
	if err := os.WriteFile(path, []byte("Radiators reject waste heat in space."), 0o644); err != nil {
		t.Fatal(err)
	}
	prompt, err := buildPromptWithAttachments("Thiết kế tàu vũ trụ", []pendingAttachment{{Path: path}})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Thiết kế tàu vũ trụ",
		"UNTRUSTED REFERENCE DATA",
		"không suy diễn rằng đây là bản thảo cần viết lại",
		"Radiators reject waste heat",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestSelectRelevantExcerptFindsMatchingLateChunk(t *testing.T) {
	text := strings.Repeat("Thông tin chung không liên quan.\n", 400) +
		"Động cơ nhiệt hạch cần bộ tản nhiệt và lớp chắn neutron."
	excerpt := document.SelectRelevantExcerpt(text, "tản nhiệt neutron", 2500)
	if !strings.Contains(excerpt, "lớp chắn neutron") {
		t.Fatalf("relevant late chunk was not selected: %s", excerpt)
	}
	if len([]rune(excerpt)) > 3000 {
		t.Fatalf("excerpt exceeded expected budget: %d runes", len([]rune(excerpt)))
	}
}
