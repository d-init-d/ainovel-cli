package research

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/voocel/ainovel-cli/internal/document"
)

// Request defines a research request.
type Request struct {
	Goal       string
	Questions  []string
	Queries    []string
	SourceURLs []string
	FilePaths  []string
	Domain     string
	Freshness  string
	MaxSources int
}

// Runner executes research requests using the d-research plugin.
type Runner struct {
	cfg    Config
	plugin *PluginInfo
}

type searchHit struct {
	URL           string
	Title         string
	Snippet       string
	Engine        string
	Contradiction bool
}

// NewRunner creates a new research runner.
func NewRunner(cfg Config, plugin *PluginInfo) *Runner {
	return &Runner{cfg: cfg, plugin: plugin}
}

// Run executes a research request and returns a report.
func (r *Runner) Run(ctx context.Context, req Request) (*Report, error) {
	if r == nil || r.plugin == nil {
		return nil, fmt.Errorf("d-research plugin is not available")
	}
	report := &Report{
		ID:         generateID(),
		CreatedAt:  time.Now(),
		PluginID:   r.plugin.ID,
		PluginPath: r.plugin.Path,
		Goal:       req.Goal,
		Questions:  req.Questions,
		Files:      make(map[string]string),
	}

	maxSources := req.MaxSources
	if maxSources <= 0 {
		maxSources = r.cfg.MaxSources
	}
	if maxSources <= 0 {
		maxSources = 12
	}

	queries := r.fanoutQueries(req)
	report.Queries = queries

	var allHits []searchHit
	for _, q := range queries {
		select {
		case <-ctx.Done():
			report.Warnings = append(report.Warnings, "research cancelled during search phase")
			return report, ctx.Err()
		default:
		}

		results, err := r.runSearch(ctx, q)
		if err != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("search query %q failed: %v", truncate(q, 60), err))
			continue
		}
		contradiction := isContradictionQuery(q)
		for i := range results {
			results[i].Contradiction = contradiction
		}
		allHits = append(allHits, results...)
	}

	allHits = prioritizeSourceURLs(req.SourceURLs, allHits)

	sourceCount := 0
	researchTerms := []string{req.Goal, req.Domain}
	researchTerms = append(researchTerms, req.Questions...)
	researchTerms = append(researchTerms, req.Queries...)
	researchFocus := strings.Join(researchTerms, "\n")
	for _, filePath := range dedupeURLs(req.FilePaths) {
		if sourceCount >= maxSources {
			break
		}
		evidence := r.loadLocalFile(filePath, researchFocus)
		report.Sources = append(report.Sources, evidence)
		if isBlockedStatus(evidence.AccessStatus) {
			report.Blockers = append(report.Blockers, Blocker{
				SourceURL: evidence.URL,
				Reason:    strings.Join(evidence.Blockers, ", "),
				Severity:  "source",
			})
		}
		sourceCount++
	}

	for _, hit := range allHits {
		if sourceCount >= maxSources {
			break
		}
		select {
		case <-ctx.Done():
			report.Warnings = append(report.Warnings, "research cancelled during probe phase")
			return report, ctx.Err()
		default:
		}

		evidence := r.probeSource(ctx, hit)
		report.Sources = append(report.Sources, evidence)
		if isBlockedStatus(evidence.AccessStatus) {
			report.Blockers = append(report.Blockers, Blocker{
				SourceURL: evidence.URL,
				Reason:    strings.Join(evidence.Blockers, ", "),
				Severity:  "source",
			})
		}
		sourceCount++
	}

	report.Contradictions = identifyContradictions(report.Sources)
	report.Coverage = buildCoverage(report.Sources)
	report.Coverage.Contradictions = len(report.Contradictions)

	return report, nil
}

func (r *Runner) fanoutQueries(req Request) []string {
	if len(req.Queries) > 0 {
		return req.Queries
	}

	queries := make([]string, 0, 6)
	goal := req.Goal

	queries = append(queries, goal)
	queries = append(queries, goal+" official documentation OR primary source")
	queries = append(queries, goal+" scientific OR technical OR research")
	queries = append(queries, goal+" filetype:pdf OR dataset")
	queries = append(queries, goal+" limitations OR criticism OR controversy OR debate")

	if req.Freshness != "" {
		queries = append(queries, goal+" after:"+req.Freshness)
	}

	max := r.cfg.MaxQueries
	if max > 0 && len(queries) > max {
		queries = queries[:max]
	}

	return queries
}

func (r *Runner) runSearch(ctx context.Context, query string) ([]searchHit, error) {
	scriptPath := r.plugin.ScriptPath("web_search")
	if scriptPath == "" {
		return nil, fmt.Errorf("web_search script not available")
	}

	limit := r.cfg.MaxResultsPerQuery
	if limit <= 0 {
		limit = 6
	}

	outFile := filepath.Join(os.TempDir(), "ainovel-research-"+generateID()+".json")

	args := []string{
		scriptPath,
		"--query", query,
		"--limit", fmt.Sprintf("%d", limit),
		"--out", outFile,
	}

	cmdCtx, cancel := context.WithTimeout(ctx, durationOrDefault(r.cfg.Timeout, 120*time.Second))
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "node", args...)
	cmd.Env = append(os.Environ(), "NODE_NO_WARNINGS=1")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		_ = os.Remove(outFile)
		return nil, fmt.Errorf("web_search failed: %w, stderr: %s", err, truncate(stderr.String(), 200))
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		return nil, fmt.Errorf("read search output: %w", err)
	}
	_ = os.Remove(outFile)

	var items []struct {
		URL     string `json:"url"`
		Title   string `json:"title"`
		Snippet string `json:"snippet"`
		Engine  string `json:"source_engine"`
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parse search output: %w", err)
	}

	hits := make([]searchHit, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.URL) == "" {
			continue
		}
		hits = append(hits, searchHit{
			URL:     item.URL,
			Title:   item.Title,
			Snippet: item.Snippet,
			Engine:  item.Engine,
		})
	}
	return hits, nil
}

func (r *Runner) probeSource(ctx context.Context, hit searchHit) SourceEvidence {
	evidence := SourceEvidence{
		URL:           hit.URL,
		Title:         hit.Title,
		Snippet:       truncate(hit.Snippet, 1000),
		AccessMethod:  "web_search",
		AccessStatus:  "search_only",
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Confidence:    "low",
		Contradiction: hit.Contradiction,
	}

	if !r.cfg.Browser.Enabled || !r.plugin.BrowserAvailable || !r.plugin.HasScript("browser_probe") {
		return evidence
	}

	probeScript := r.plugin.ScriptPath("browser_probe")
	if probeScript == "" {
		return evidence
	}

	outFile := filepath.Join(os.TempDir(), "ainovel-probe-"+generateID()+".json")

	timeout := int(durationOrDefault(r.cfg.Browser.Timeout, 30*time.Second).Milliseconds())

	args := []string{
		probeScript,
		"--url", hit.URL,
		"--out", outFile,
		"--timeout", fmt.Sprintf("%d", timeout),
	}
	if !r.cfg.Browser.Headless {
		args = append(args, "--headful")
	}

	probeCtx, cancel := context.WithTimeout(ctx, durationOrDefault(r.cfg.Browser.Timeout, 30*time.Second)+10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(probeCtx, "node", args...)
	cmd.Env = append(os.Environ(), "NODE_NO_WARNINGS=1")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		_ = os.Remove(outFile)
		evidence.AccessStatus = "blocked"
		evidence.Confidence = "low"
		blocker := strings.TrimSpace(stderr.String())
		if blocker == "" {
			blocker = err.Error()
		}
		evidence.Blockers = []string{truncate(blocker, 200)}
		return evidence
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		_ = os.Remove(outFile)
		evidence.AccessStatus = "blocked"
		return evidence
	}
	_ = os.Remove(outFile)

	var probeResult struct {
		FinalURL     string   `json:"finalUrl"`
		Status       int      `json:"status"`
		AccessStatus string   `json:"accessStatus"`
		Blockers     []string `json:"blockers"`
		Title        string   `json:"title"`
		TextSample   string   `json:"textSample"`
	}
	if err := json.Unmarshal(data, &probeResult); err != nil {
		evidence.AccessStatus = "partial"
		return evidence
	}

	if probeResult.FinalURL != "" {
		evidence.URL = probeResult.FinalURL
	}
	if probeResult.Title != "" {
		evidence.Title = probeResult.Title
	}
	if probeResult.TextSample != "" {
		evidence.Snippet = truncate(probeResult.TextSample, 1000)
	}
	evidence.AccessMethod = "browser_probe"
	evidence.AccessStatus = probeResult.AccessStatus
	if evidence.AccessStatus == "" {
		evidence.AccessStatus = "partial"
	}
	evidence.Blockers = probeResult.Blockers

	if isBlockedStatus(evidence.AccessStatus) || len(probeResult.Blockers) > 0 {
		evidence.Confidence = "low"
		return evidence
	}

	evidence.Confidence = "medium"

	if r.cfg.Browser.Extract && r.plugin.HasScript("browser_extract") {
		r.extractContent(ctx, evidence.URL, &evidence)
	}

	return evidence
}

func (r *Runner) loadLocalFile(path string, focus ...string) SourceEvidence {
	absPath, err := filepath.Abs(path)
	if err == nil {
		path = absPath
	}
	evidence := SourceEvidence{
		URL:          localFileURL(path),
		Title:        filepath.Base(path),
		SourceType:   "local_file",
		AccessMethod: "local_file",
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Confidence:   "high",
	}

	result, err := document.ReadText(path, document.Options{})
	if err != nil {
		evidence.AccessStatus = "not_found"
		if errors.Is(err, document.ErrUnsupportedFormat) {
			evidence.AccessStatus = "unsupported_format"
		}
		evidence.Confidence = "low"
		evidence.Blockers = []string{err.Error()}
		return evidence
	}

	text := strings.TrimSpace(result.Text)
	if text == "" {
		evidence.AccessStatus = "partial_or_empty"
		evidence.Confidence = "low"
		evidence.Blockers = []string{"local file is empty or not readable as text"}
		return evidence
	}

	evidence.AccessStatus = "accessible"
	if result.Truncated {
		evidence.AccessStatus = "partial"
		evidence.Confidence = "medium"
		evidence.Blockers = []string{"local document exceeded the safe extraction budget; only the extracted prefix was inspected"}
	}
	extracted := document.SelectRelevantExcerpt(text, strings.Join(focus, "\n"), 8000)
	evidence.Snippet = truncate(extracted, 1000)
	evidence.ExtractedContent = extracted
	return evidence
}

func (r *Runner) extractContent(ctx context.Context, url string, evidence *SourceEvidence) {
	extractScript := r.plugin.ScriptPath("browser_extract")
	if extractScript == "" {
		return
	}

	outFile := filepath.Join(os.TempDir(), "ainovel-extract-"+generateID()+".json")

	timeout := int(durationOrDefault(r.cfg.Browser.Timeout, 30*time.Second).Milliseconds())
	args := []string{
		extractScript,
		"--url", url,
		"--format", "json",
		"--out", outFile,
		"--timeout", fmt.Sprintf("%d", timeout),
	}

	extractCtx, cancel := context.WithTimeout(ctx, durationOrDefault(r.cfg.Browser.Timeout, 30*time.Second)+10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(extractCtx, "node", args...)
	cmd.Env = append(os.Environ(), "NODE_NO_WARNINGS=1")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		_ = os.Remove(outFile)
		evidence.AccessStatus = "partial"
		return
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		_ = os.Remove(outFile)
		return
	}
	_ = os.Remove(outFile)

	var extractResult struct {
		FinalURL string `json:"finalUrl"`
		Text     string `json:"text"`
		Title    string `json:"title"`
	}
	if err := json.Unmarshal(data, &extractResult); err != nil {
		return
	}

	if extractResult.FinalURL != "" {
		evidence.URL = extractResult.FinalURL
	}
	if extractResult.Text != "" {
		evidence.ExtractedContent = truncate(extractResult.Text, 8000)
		evidence.Snippet = truncate(extractResult.Text, 1000)
		evidence.AccessMethod = "browser_extract"
		evidence.Confidence = "high"
	}
	if extractResult.Title != "" {
		evidence.Title = extractResult.Title
	}
}

func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func dedupeURLs(urls []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(urls))
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" || seen[u] {
			continue
		}
		seen[u] = true
		result = append(result, u)
	}
	return result
}

func dedupeSearchHits(hits []searchHit) []searchHit {
	seen := make(map[string]bool)
	result := make([]searchHit, 0, len(hits))
	for _, hit := range hits {
		key := normalizeURLKey(hit.URL)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, hit)
	}
	return result
}

func prioritizeSourceURLs(sourceURLs []string, discovered []searchHit) []searchHit {
	prioritized := make([]searchHit, 0, len(sourceURLs)+len(discovered))
	for _, sourceURL := range sourceURLs {
		prioritized = append(prioritized, searchHit{URL: sourceURL})
	}
	prioritized = append(prioritized, discovered...)
	return dedupeSearchHits(prioritized)
}

func normalizeURLKey(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return strings.ToLower(strings.TrimRight(raw, "/"))
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Fragment = ""
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return parsed.String()
}

func buildCoverage(sources []SourceEvidence) Coverage {
	c := Coverage{
		TotalSources: len(sources),
	}
	domains := make(map[string]bool)
	for _, s := range sources {
		switch s.AccessStatus {
		case "accessible", "partial", "partial_or_empty", "search_only":
			c.Accessed++
		default:
			if isBlockedStatus(s.AccessStatus) {
				c.Blocked++
			}
		}
		if d := extractDomain(s.URL); d != "" {
			domains[d] = true
		}
	}
	for d := range domains {
		c.Domains = append(c.Domains, d)
	}
	return c
}

func identifyContradictions(sources []SourceEvidence) []Contradiction {
	var contradictions []Contradiction
	candidates := make([]SourceEvidence, 0)
	for _, s := range sources {
		if s.Contradiction {
			candidates = append(candidates, s)
		}
	}
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[i].URL != candidates[j].URL {
				contradictions = append(contradictions, Contradiction{
					ClaimA:  truncate(candidates[i].Snippet, 100),
					SourceA: candidates[i].URL,
					ClaimB:  truncate(candidates[j].Snippet, 100),
					SourceB: candidates[j].URL,
					Note:    "contradiction candidate — requires human verification",
				})
			}
		}
	}
	return contradictions
}

func isContradictionQuery(query string) bool {
	q := strings.ToLower(query)
	return strings.Contains(q, "limitation") ||
		strings.Contains(q, "criticism") ||
		strings.Contains(q, "controversy") ||
		strings.Contains(q, "contradiction") ||
		strings.Contains(q, "debate")
}

func isBlockedStatus(status string) bool {
	switch status {
	case "blocked", "broken", "captcha", "login_required", "paywalled", "forbidden", "geo_blocked", "rate_limited", "server_error", "not_found", "unsupported_format":
		return true
	default:
		return false
	}
}

func durationOrDefault(value, fallback time.Duration) time.Duration {
	if value <= 0 {
		return fallback
	}
	return value
}

func extractDomain(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	parts := strings.SplitN(url, "/", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func localFileURL(path string) string {
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}
