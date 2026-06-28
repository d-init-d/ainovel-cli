package research

import (
	"fmt"
	"time"
)

// SourceEvidence represents a single source with evidence.
type SourceEvidence struct {
	URL              string   `json:"url"`
	Title            string   `json:"title"`
	Snippet          string   `json:"snippet,omitempty"`
	SourceType       string   `json:"source_type,omitempty"`
	AccessMethod     string   `json:"access_method"`
	AccessStatus     string   `json:"access_status"`
	Timestamp        string   `json:"timestamp"`
	Confidence       string   `json:"confidence,omitempty"`
	Contradiction    bool     `json:"contradiction,omitempty"`
	ExtractedContent string   `json:"extracted_content,omitempty"`
	Blockers         []string `json:"blockers,omitempty"`
}

// Blocker represents a research blocker.
type Blocker struct {
	SourceURL string `json:"source_url,omitempty"`
	Reason    string `json:"reason"`
	Severity  string `json:"severity"`
}

// Contradiction represents a contradiction between sources.
type Contradiction struct {
	ClaimA  string `json:"claim_a"`
	SourceA string `json:"source_a"`
	ClaimB  string `json:"claim_b"`
	SourceB string `json:"source_b"`
	Note    string `json:"note,omitempty"`
}

// Coverage represents research coverage assessment.
type Coverage struct {
	TotalSources   int      `json:"total_sources"`
	Accessed       int      `json:"accessed"`
	Blocked        int      `json:"blocked"`
	Contradictions int      `json:"contradictions"`
	Domains        []string `json:"domains,omitempty"`
}

// Report is the complete research report.
type Report struct {
	ID             string            `json:"id"`
	CreatedAt      time.Time         `json:"created_at"`
	PluginID       string            `json:"plugin_id"`
	PluginPath     string            `json:"plugin_path"`
	Goal           string            `json:"goal"`
	Questions      []string          `json:"questions,omitempty"`
	Queries        []string          `json:"queries,omitempty"`
	Sources        []SourceEvidence  `json:"sources,omitempty"`
	Blockers       []Blocker         `json:"blockers,omitempty"`
	Contradictions []Contradiction   `json:"contradictions,omitempty"`
	Coverage       Coverage          `json:"coverage"`
	Files          map[string]string `json:"files,omitempty"`
	Warnings       []string          `json:"warnings,omitempty"`
}

// CompactSource keeps the source detail that is useful inside LLM context.
type CompactSource struct {
	Title         string   `json:"title,omitempty"`
	URL           string   `json:"url"`
	Snippet       string   `json:"snippet,omitempty"`
	AccessStatus  string   `json:"access_status,omitempty"`
	Confidence    string   `json:"confidence,omitempty"`
	Contradiction bool     `json:"contradiction,omitempty"`
	Blockers      []string `json:"blockers,omitempty"`
}

// CompactReport is a lightweight version for context injection.
type CompactReport struct {
	ID             string            `json:"id"`
	Goal           string            `json:"goal"`
	Summary        string            `json:"summary"`
	Sources        int               `json:"sources"`
	Blockers       int               `json:"blockers"`
	Contradictions int               `json:"contradictions"`
	Coverage       Coverage          `json:"coverage"`
	Highlights     []CompactSource   `json:"highlights,omitempty"`
	Warnings       []string          `json:"warnings,omitempty"`
	Files          map[string]string `json:"files,omitempty"`
	CreatedAt      string            `json:"created_at"`
}

// ToCompact creates a compact version of the report.
func (r *Report) ToCompact() CompactReport {
	summary := r.Goal
	if len(r.Sources) > 0 {
		summary += " | " + pluralize(len(r.Sources), "source") + " reached"
	}
	if len(r.Blockers) > 0 {
		summary += " | " + pluralize(len(r.Blockers), "blocker")
	}
	if len(r.Contradictions) > 0 {
		summary += " | " + pluralize(len(r.Contradictions), "contradiction")
	}

	highlights := make([]CompactSource, 0, 6)
	for _, source := range r.Sources {
		if len(highlights) >= 6 {
			break
		}
		if source.URL == "" && source.Snippet == "" {
			continue
		}
		highlights = append(highlights, CompactSource{
			Title:         source.Title,
			URL:           source.URL,
			Snippet:       truncate(source.Snippet, 350),
			AccessStatus:  source.AccessStatus,
			Confidence:    source.Confidence,
			Contradiction: source.Contradiction,
			Blockers:      append([]string(nil), source.Blockers...),
		})
	}

	return CompactReport{
		ID:             r.ID,
		Goal:           r.Goal,
		Summary:        summary,
		Sources:        len(r.Sources),
		Blockers:       len(r.Blockers),
		Contradictions: len(r.Contradictions),
		Coverage:       r.Coverage,
		Highlights:     highlights,
		Warnings:       append([]string(nil), r.Warnings...),
		Files:          cloneStringMap(r.Files),
		CreatedAt:      r.CreatedAt.Format(time.RFC3339),
	}
}

func pluralize(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

func cloneStringMap(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	clone := make(map[string]string, len(m))
	for key, value := range m {
		clone[key] = value
	}
	return clone
}
