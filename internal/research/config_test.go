package research

import (
	"testing"
	"time"

	"github.com/voocel/ainovel-cli/internal/bootstrap"
)

func TestConfigFromBootstrap(t *testing.T) {
	enabled := true
	auto := true
	headless := true
	extract := true

	cfg := bootstrap.Config{
		Research: bootstrap.ResearchConfig{
			Enabled:            &enabled,
			Plugin:             "d-research",
			PluginPath:         "/custom/path",
			Auto:               &auto,
			MaxQueries:         10,
			MaxResultsPerQuery: 5,
			MaxSources:         20,
			TimeoutSeconds:     60,
			Browser: bootstrap.ResearchBrowserConfig{
				Enabled:        &enabled,
				Headless:       &headless,
				TimeoutSeconds: 15,
				Extract:        &extract,
			},
		},
	}

	rc := ConfigFromBootstrap(cfg)
	if !rc.Enabled {
		t.Error("expected enabled")
	}
	if rc.Plugin != "d-research" {
		t.Errorf("expected plugin d-research, got %q", rc.Plugin)
	}
	if rc.PluginPath != "/custom/path" {
		t.Errorf("expected plugin path /custom/path, got %q", rc.PluginPath)
	}
	if !rc.Auto {
		t.Error("expected auto")
	}
	if rc.MaxQueries != 10 {
		t.Errorf("expected max_queries 10, got %d", rc.MaxQueries)
	}
	if rc.MaxResultsPerQuery != 5 {
		t.Errorf("expected max_results_per_query 5, got %d", rc.MaxResultsPerQuery)
	}
	if rc.MaxSources != 20 {
		t.Errorf("expected max_sources 20, got %d", rc.MaxSources)
	}
	if rc.Timeout != 60*time.Second {
		t.Errorf("expected timeout 60s, got %v", rc.Timeout)
	}
	if !rc.Browser.Enabled {
		t.Error("expected browser enabled")
	}
	if !rc.Browser.Headless {
		t.Error("expected browser headless")
	}
	if rc.Browser.Timeout != 15*time.Second {
		t.Errorf("expected browser timeout 15s, got %v", rc.Browser.Timeout)
	}
}

func TestConfigFromBootstrapDisabled(t *testing.T) {
	disabled := false
	cfg := bootstrap.Config{
		Research: bootstrap.ResearchConfig{
			Enabled: &disabled,
		},
	}

	rc := ConfigFromBootstrap(cfg)
	if rc.Enabled {
		t.Error("expected disabled")
	}
}

func TestConfigFromBootstrapDefaults(t *testing.T) {
	cfg := bootstrap.Config{}
	rc := ConfigFromBootstrap(cfg)
	if rc.Enabled {
		t.Error("expected research disabled by default")
	}
	if rc.Plugin != "d-research" {
		t.Errorf("expected default plugin d-research, got %q", rc.Plugin)
	}
	if rc.MaxQueries != 8 {
		t.Errorf("expected default max_queries 8, got %d", rc.MaxQueries)
	}
	if rc.MaxResultsPerQuery != 6 {
		t.Errorf("expected default max_results_per_query 6, got %d", rc.MaxResultsPerQuery)
	}
	if rc.MaxSources != 12 {
		t.Errorf("expected default max_sources 12, got %d", rc.MaxSources)
	}
	if rc.Timeout != 120*time.Second {
		t.Errorf("expected default timeout 120s, got %v", rc.Timeout)
	}
}
