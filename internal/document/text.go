package document

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/voocel/ainovel-cli/internal/utils"
)

const (
	DefaultMaxInputBytes  int64 = 256 << 20
	DefaultMaxOutputBytes int64 = 256 << 20
)

var ErrUnsupportedFormat = errors.New("unsupported document format")

func IsPromptAttachment(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".txt", ".md", ".markdown", ".docx":
		return true
	default:
		return false
	}
}

type Options struct {
	MaxInputBytes  int64
	MaxOutputBytes int64
}

type Result struct {
	Text      string
	Format    string
	Truncated bool
}

func ReadText(path string, opts Options) (Result, error) {
	opts = normalizeOptions(opts)
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".docx":
		return readDOCX(path, opts)
	case ".doc":
		return Result{}, fmt.Errorf("%w: legacy .doc is not supported; save it as .docx, .txt, or .md", ErrUnsupportedFormat)
	case ".pdf", ".epub", ".mobi", ".azw", ".azw3", ".xls", ".xlsx", ".ppt", ".pptx", ".zip", ".rar", ".7z", ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return Result{}, fmt.Errorf("%w: %s", ErrUnsupportedFormat, ext)
	default:
		return readPlainText(path, opts)
	}
}

func normalizeOptions(opts Options) Options {
	if opts.MaxInputBytes <= 0 {
		opts.MaxInputBytes = DefaultMaxInputBytes
	}
	if opts.MaxOutputBytes <= 0 {
		opts.MaxOutputBytes = DefaultMaxOutputBytes
	}
	return opts
}

func readPlainText(path string, opts Options) (Result, error) {
	file, err := os.Open(path)
	if err != nil {
		return Result{}, err
	}
	defer file.Close()

	data, truncated, err := readLimited(file, opts.MaxInputBytes)
	if err != nil {
		return Result{}, err
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return Result{}, fmt.Errorf("%w: file appears to contain binary data", ErrUnsupportedFormat)
	}
	text := utils.DecodeText(data)
	if int64(len(text)) > opts.MaxOutputBytes {
		text = validUTF8Prefix(text, opts.MaxOutputBytes)
		truncated = true
	}
	return Result{Text: text, Format: "text", Truncated: truncated}, nil
}

func readDOCX(path string, opts Options) (Result, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Result{}, err
	}
	if info.Size() > opts.MaxInputBytes {
		return Result{}, fmt.Errorf("DOCX archive exceeds safe input limit of %d bytes", opts.MaxInputBytes)
	}

	archive, err := zip.OpenReader(path)
	if err != nil {
		return Result{}, fmt.Errorf("open DOCX archive: %w", err)
	}
	defer archive.Close()

	var documentXML *zip.File
	for _, entry := range archive.File {
		if filepath.ToSlash(entry.Name) == "word/document.xml" {
			documentXML = entry
			break
		}
	}
	if documentXML == nil {
		return Result{}, fmt.Errorf("invalid DOCX: word/document.xml is missing")
	}
	if int64(documentXML.UncompressedSize64) > opts.MaxOutputBytes*8 {
		return Result{}, fmt.Errorf("DOCX XML exceeds safe expansion limit")
	}

	reader, err := documentXML.Open()
	if err != nil {
		return Result{}, fmt.Errorf("open DOCX document XML: %w", err)
	}
	defer reader.Close()

	text, truncated, err := extractWordText(reader, opts.MaxOutputBytes)
	if err != nil {
		return Result{}, err
	}
	return Result{Text: text, Format: "docx", Truncated: truncated}, nil
}

func extractWordText(reader io.Reader, maxBytes int64) (string, bool, error) {
	decoder := xml.NewDecoder(reader)
	var output strings.Builder
	inText := false
	truncated := false

	appendText := func(value string) {
		if value == "" || truncated {
			return
		}
		remaining := maxBytes - int64(output.Len())
		if remaining <= 0 {
			truncated = true
			return
		}
		if int64(len(value)) > remaining {
			value = validUTF8Prefix(value, remaining)
			truncated = true
		}
		output.WriteString(value)
	}

	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", false, fmt.Errorf("parse DOCX XML: %w", err)
		}
		switch value := token.(type) {
		case xml.StartElement:
			switch value.Name.Local {
			case "t":
				inText = true
			case "tab":
				appendText("\t")
			case "br", "cr":
				appendText("\n")
			}
		case xml.CharData:
			if inText {
				appendText(string(value))
			}
		case xml.EndElement:
			switch value.Name.Local {
			case "t":
				inText = false
			case "p":
				appendText("\n")
			case "tc":
				appendText("\t")
			case "tr":
				appendText("\n")
			}
		}
		if truncated {
			return strings.TrimSpace(output.String()), true, nil
		}
	}

	return strings.TrimSpace(output.String()), truncated, nil
}

func readLimited(reader io.Reader, limit int64) ([]byte, bool, error) {
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, false, err
	}
	if int64(len(data)) <= limit {
		return data, false, nil
	}
	return data[:limit], true, nil
}

func validUTF8Prefix(value string, maxBytes int64) string {
	if maxBytes <= 0 {
		return ""
	}
	if int64(len(value)) <= maxBytes {
		return value
	}
	prefix := value[:maxBytes]
	for prefix != "" && !utf8.ValidString(prefix) {
		prefix = prefix[:len(prefix)-1]
	}
	return prefix
}
