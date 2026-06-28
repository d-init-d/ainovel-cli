package document

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadTextPlainAndMarkdown(t *testing.T) {
	for _, name := range []string{"notes.txt", "notes.md", "notes.markdown"} {
		path := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(path, []byte("Chapter 1\nReference material"), 0o644); err != nil {
			t.Fatal(err)
		}
		result, err := ReadText(path, Options{})
		if err != nil {
			t.Fatalf("ReadText(%s): %v", name, err)
		}
		if result.Text != "Chapter 1\nReference material" || result.Format != "text" {
			t.Fatalf("unexpected result for %s: %#v", name, result)
		}
	}
}

func TestReadTextDOCX(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reference.docx")
	writeTestDOCX(t, path, `<w:document xmlns:w="urn:test"><w:body>`+
		`<w:p><w:r><w:t>Chapter 1</w:t></w:r></w:p>`+
		`<w:p><w:r><w:t>Fusion reference</w:t><w:tab/><w:t>verified</w:t></w:r></w:p>`+
		`</w:body></w:document>`)

	result, err := ReadText(path, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Format != "docx" {
		t.Fatalf("format = %q", result.Format)
	}
	for _, want := range []string{"Chapter 1", "Fusion reference", "verified"} {
		if !strings.Contains(result.Text, want) {
			t.Fatalf("DOCX text missing %q: %q", want, result.Text)
		}
	}
}

func TestReadTextRejectsLegacyDOC(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.doc")
	if err := os.WriteFile(path, []byte("legacy"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ReadText(path, Options{})
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("expected unsupported format, got %v", err)
	}
}

func TestReadTextTruncatesWithoutBreakingUTF8(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unicode.txt")
	if err := os.WriteFile(path, []byte("nghiên cứu khoa học"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ReadText(path, Options{MaxInputBytes: 12, MaxOutputBytes: 12})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Truncated || !strings.Contains(result.Text, "nghi") {
		t.Fatalf("unexpected truncated result: %#v", result)
	}
	if strings.ContainsRune(result.Text, '\uFFFD') {
		t.Fatalf("truncation broke UTF-8: %q", result.Text)
	}
}

func writeTestDOCX(t *testing.T, path, documentXML string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	entry, err := archive.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte(documentXML)); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
