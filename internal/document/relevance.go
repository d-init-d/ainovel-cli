package document

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type scoredChunk struct {
	index int
	text  string
	score int
}

func SelectRelevantExcerpt(text, query string, maxRunes int) string {
	if maxRunes <= 0 || utf8.RuneCountInString(text) <= maxRunes {
		return strings.TrimSpace(text)
	}

	chunks := chunkText(text, 1800)
	terms := queryTerms(query)
	for i := range chunks {
		lower := strings.ToLower(chunks[i].text)
		for _, term := range terms {
			chunks[i].score += strings.Count(lower, term)
		}
	}

	ranked := append([]scoredChunk(nil), chunks...)
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			return ranked[i].index < ranked[j].index
		}
		return ranked[i].score > ranked[j].score
	})

	selected := make([]scoredChunk, 0, len(ranked))
	used := 0
	for _, chunk := range ranked {
		length := utf8.RuneCountInString(chunk.text)
		if used > 0 && used+length > maxRunes {
			continue
		}
		selected = append(selected, chunk)
		used += length
		if used >= maxRunes {
			break
		}
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].index < selected[j].index })

	var b strings.Builder
	for i, chunk := range selected {
		if i > 0 {
			b.WriteString("\n\n[… unrelated sections omitted …]\n\n")
		}
		b.WriteString(strings.TrimSpace(chunk.text))
	}
	return b.String()
}

func chunkText(text string, targetRunes int) []scoredChunk {
	paragraphs := strings.FieldsFunc(text, func(r rune) bool { return r == '\n' || r == '\r' })
	chunks := make([]scoredChunk, 0, len(paragraphs)/4+1)
	var b strings.Builder
	flush := func() {
		value := strings.TrimSpace(b.String())
		if value != "" {
			chunks = append(chunks, scoredChunk{index: len(chunks), text: value})
		}
		b.Reset()
	}
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		if utf8.RuneCountInString(paragraph) > targetRunes {
			flush()
			runes := []rune(paragraph)
			for start := 0; start < len(runes); start += targetRunes {
				end := start + targetRunes
				if end > len(runes) {
					end = len(runes)
				}
				chunks = append(chunks, scoredChunk{index: len(chunks), text: string(runes[start:end])})
			}
			continue
		}
		if b.Len() > 0 && utf8.RuneCountInString(b.String())+utf8.RuneCountInString(paragraph) > targetRunes {
			flush()
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(paragraph)
	}
	flush()
	return chunks
}

func queryTerms(query string) []string {
	seen := make(map[string]bool)
	var terms []string
	for _, term := range strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if utf8.RuneCountInString(term) < 3 || seen[term] {
			continue
		}
		seen[term] = true
		terms = append(terms, term)
	}
	return terms
}
