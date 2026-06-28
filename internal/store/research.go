package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ResearchStore persists d-research reports and compact context packs.
type ResearchStore struct {
	io *IO
	mu sync.RWMutex
}

// NewResearchStore creates a research store.
func NewResearchStore(io *IO) *ResearchStore {
	return &ResearchStore{io: io}
}

// SaveReport stores the full research report.
func (s *ResearchStore) SaveReport(reportID string, data any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	relDir := filepath.Join("meta", "research", reportID)
	if err := s.io.EnsureDirs([]string{relDir}); err != nil {
		return fmt.Errorf("create research dir: %w", err)
	}

	reportPath := filepath.Join(relDir, "report.json")
	if err := s.io.WriteJSONUnlocked(reportPath, data); err != nil {
		return fmt.Errorf("save report: %w", err)
	}

	return nil
}

// SaveEvidenceLedger stores source evidence for a report.
func (s *ResearchStore) SaveEvidenceLedger(reportID string, data any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	relDir := filepath.Join("meta", "research", reportID)
	ledgerPath := filepath.Join(relDir, "evidence-ledger.json")
	return s.io.WriteJSONUnlocked(ledgerPath, data)
}

// SaveCompactReport stores the compact report used by novel_context.
func (s *ResearchStore) SaveCompactReport(reportID string, data any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	relDir := filepath.Join("meta", "research", reportID)
	if err := s.io.EnsureDirs([]string{relDir}); err != nil {
		return fmt.Errorf("create research dir: %w", err)
	}
	return s.io.WriteJSONUnlocked(filepath.Join(relDir, "compact.json"), data)
}

// SaveRawOutput stores a raw research output file.
func (s *ResearchStore) SaveRawOutput(reportID, filename string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	relDir := filepath.Join("meta", "research", reportID)
	if err := s.io.EnsureDirs([]string{relDir}); err != nil {
		return err
	}
	return s.io.WriteFileUnlocked(filepath.Join(relDir, filename), data)
}

// UpdateLatest points to the latest research report.
func (s *ResearchStore) UpdateLatest(reportID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	latest := map[string]string{"latest_report": reportID}
	return s.io.WriteJSONUnlocked(filepath.Join("meta", "research", "latest.json"), latest)
}

// LoadLatestReportID returns the latest research report ID.
func (s *ResearchStore) LoadLatestReportID() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var latest struct {
		LatestReport string `json:"latest_report"`
	}
	if err := s.io.ReadJSONUnlocked(filepath.Join("meta", "research", "latest.json"), &latest); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return latest.LatestReport, nil
}

// LoadCompactReport loads a compact research report.
func (s *ResearchStore) LoadCompactReport(reportID string, v any) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	compactPath := filepath.Join("meta", "research", reportID, "compact.json")
	return s.io.ReadJSONUnlocked(compactPath, v)
}

// LoadLatestCompactReport loads the latest compact research report.
func (s *ResearchStore) LoadLatestCompactReport(v any) error {
	reportID, err := s.LoadLatestReportID()
	if err != nil || reportID == "" {
		return err
	}
	return s.LoadCompactReport(reportID, v)
}

// SaveRawJSON stores a JSON artifact in the report directory.
func (s *ResearchStore) SaveRawJSON(reportID, name string, data any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	relDir := filepath.Join("meta", "research", reportID)
	if err := s.io.EnsureDirs([]string{relDir}); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return s.io.WriteFileUnlocked(filepath.Join(relDir, name+".json"), raw)
}
