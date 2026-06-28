package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/voocel/agentcore/schema"
	"github.com/voocel/ainovel-cli/internal/errs"
	"github.com/voocel/ainovel-cli/internal/research"
	"github.com/voocel/ainovel-cli/internal/store"
)

// ResearchBriefTool runs web research through the d-research plugin.
// It is intended for Architect agents before save_foundation when real-world grounding is needed.
type ResearchBriefTool struct {
	store  *store.Store
	runner *research.Runner
}

// NewResearchBriefTool creates the research tool.
func NewResearchBriefTool(st *store.Store, runner *research.Runner) *ResearchBriefTool {
	return &ResearchBriefTool{store: st, runner: runner}
}

func (t *ResearchBriefTool) Name() string { return "research_brief" }
func (t *ResearchBriefTool) Description() string {
	return "Thực hiện nghiên cứu web nhanh cho bối cảnh thực tế (khoa học/kỹ thuật/lịch sử/pháp lý/y tế/kinh tế). " +
		"Gọi trước save_foundation khi câu chuyện cần căn cứ thế giới thực. " +
		"Trả về báo cáo cô đọng với nguồn dẫn, bằng chứng, và cảnh báo."
}
func (t *ResearchBriefTool) Label() string { return "Nghiên cứu nhanh" }

func (t *ResearchBriefTool) ReadOnly(_ json.RawMessage) bool        { return false }
func (t *ResearchBriefTool) ConcurrencySafe(_ json.RawMessage) bool { return false }

func (t *ResearchBriefTool) Schema() map[string]any {
	return schema.Object(
		schema.Property("goal", schema.String("Mục tiêu nghiên cứu (bắt buộc). Ví dụ: 'cơ chế hoạt động của lò phản ứng nhiệt hạch'")).Required(),
		schema.Property("questions", schema.Array("Các câu hỏi phụ cần trả lời", schema.String(""))),
		schema.Property("queries", schema.Array("Truy vấn tìm kiếm tường minh; nếu không cung cấp sẽ tự động sinh từ goal", schema.String(""))),
		schema.Property("source_urls", schema.Array("URL cụ thể cần thăm dò", schema.String(""))),
		schema.Property("file_paths", schema.Array("Đường dẫn tệp local do người dùng cung cấp làm tài liệu nghiên cứu", schema.String(""))),
		schema.Property("domain", schema.String("Lĩnh vực (khoa học/kỹ thuật/lịch sử/v.v.)")),
		schema.Property("freshness", schema.String("Yêu cầu độ mới (YYYY-MM-DD hoặc relative như '2024-01-01')")),
		schema.Property("max_sources", schema.Int("Số nguồn tối đa (mặc định 12)")),
	)
}

func (t *ResearchBriefTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var a struct {
		Goal       string   `json:"goal"`
		Questions  []string `json:"questions"`
		Queries    []string `json:"queries"`
		SourceURLs []string `json:"source_urls"`
		FilePaths  []string `json:"file_paths"`
		Domain     string   `json:"domain"`
		Freshness  string   `json:"freshness"`
		MaxSources int      `json:"max_sources"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("invalid args: %w: %w", errs.ErrToolArgs, err)
	}
	if a.Goal == "" {
		return nil, fmt.Errorf("goal is required: %w", errs.ErrToolArgs)
	}
	if a.MaxSources < 0 {
		return nil, fmt.Errorf("max_sources must be >= 0: %w", errs.ErrToolArgs)
	}

	if t.runner == nil {
		return json.Marshal(map[string]any{
			"error":    "research plugin not available",
			"goal":     a.Goal,
			"blocked":  true,
			"warnings": []string{"d-research plugin is not configured or not found. Install the d-research plugin and enable research in config."},
		})
	}

	req := research.Request{
		Goal:       a.Goal,
		Questions:  a.Questions,
		Queries:    a.Queries,
		SourceURLs: a.SourceURLs,
		FilePaths:  a.FilePaths,
		Domain:     a.Domain,
		Freshness:  a.Freshness,
		MaxSources: a.MaxSources,
	}

	report, err := t.runner.Run(ctx, req)
	if err != nil {
		slog.Warn("research_brief failed", "module", "tool", "goal", truncateStr(a.Goal, 60), "err", err)
		return json.Marshal(map[string]any{
			"error":    err.Error(),
			"goal":     a.Goal,
			"blocked":  true,
			"warnings": []string{err.Error()},
		})
	}

	// Save report to store
	reportID := report.ID
	report.Files["report"] = filepath.Join(t.store.Dir(), "meta", "research", reportID, "report.json")
	report.Files["evidence_ledger"] = filepath.Join(t.store.Dir(), "meta", "research", reportID, "evidence-ledger.json")
	report.Files["compact"] = filepath.Join(t.store.Dir(), "meta", "research", reportID, "compact.json")
	compact := report.ToCompact()

	if err := t.store.Research.SaveReport(reportID, report); err != nil {
		slog.Warn("research_brief: save report failed", "module", "tool", "id", reportID, "err", err)
	}
	if err := t.store.Research.SaveCompactReport(reportID, compact); err != nil {
		slog.Warn("research_brief: save compact report failed", "module", "tool", "id", reportID, "err", err)
	}
	if err := t.store.Research.SaveEvidenceLedger(reportID, report.Sources); err != nil {
		slog.Warn("research_brief: save evidence ledger failed", "module", "tool", "id", reportID, "err", err)
	}
	if err := t.store.Research.UpdateLatest(reportID); err != nil {
		slog.Warn("research_brief: update latest failed", "module", "tool", "id", reportID, "err", err)
	}

	result := map[string]any{
		"report_id":  compact.ID,
		"goal":       compact.Goal,
		"summary":    compact.Summary,
		"sources":    compact.Sources,
		"blockers":   compact.Blockers,
		"coverage":   compact.Coverage,
		"highlights": compact.Highlights,
		"warnings":   compact.Warnings,
		"created_at": compact.CreatedAt,
		"file_paths": report.Files,
	}

	return json.Marshal(result)
}

func truncateStr(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}
