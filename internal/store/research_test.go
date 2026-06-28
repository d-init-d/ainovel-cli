package store

import "testing"

func TestResearchStoreSavesAndLoadsLatestCompactReport(t *testing.T) {
	st := NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	compact := map[string]any{
		"id":      "report-1",
		"goal":    "fusion drive constraints",
		"sources": float64(2),
	}
	if err := st.Research.SaveCompactReport("report-1", compact); err != nil {
		t.Fatalf("SaveCompactReport failed: %v", err)
	}
	if err := st.Research.UpdateLatest("report-1"); err != nil {
		t.Fatalf("UpdateLatest failed: %v", err)
	}

	var loaded map[string]any
	if err := st.Research.LoadLatestCompactReport(&loaded); err != nil {
		t.Fatalf("LoadLatestCompactReport failed: %v", err)
	}
	if loaded["id"] != "report-1" {
		t.Fatalf("expected report-1, got %#v", loaded["id"])
	}
	if loaded["goal"] != "fusion drive constraints" {
		t.Fatalf("expected saved goal, got %#v", loaded["goal"])
	}
}
