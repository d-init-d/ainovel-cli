package research

import (
	"time"

	"github.com/voocel/ainovel-cli/internal/bootstrap"
)

// ConfigFromBootstrap extracts a normalized ResearchConfig from bootstrap config.
func ConfigFromBootstrap(cfg bootstrap.Config) Config {
	r := cfg.Research
	c := Config{
		Enabled:            r.IsEnabled(),
		Plugin:             r.Plugin,
		PluginPath:         r.PluginPath,
		Auto:               r.IsAuto(),
		MaxQueries:         r.MaxQueries,
		MaxResultsPerQuery: r.MaxResultsPerQuery,
		MaxSources:         r.MaxSources,
		Timeout:            time.Duration(r.TimeoutSeconds) * time.Second,
	}
	if c.Plugin == "" {
		c.Plugin = "d-research"
	}
	if c.MaxQueries <= 0 {
		c.MaxQueries = 8
	}
	if c.MaxResultsPerQuery <= 0 {
		c.MaxResultsPerQuery = 6
	}
	if c.MaxSources <= 0 {
		c.MaxSources = 12
	}
	if c.Timeout <= 0 {
		c.Timeout = 120 * time.Second
	}
	if r.Browser.Enabled == nil || *r.Browser.Enabled {
		c.Browser.Enabled = true
	}
	if r.Browser.Headless == nil || *r.Browser.Headless {
		c.Browser.Headless = true
	}
	if r.Browser.Extract == nil || *r.Browser.Extract {
		c.Browser.Extract = true
	}
	c.Browser.Timeout = time.Duration(r.Browser.TimeoutSeconds) * time.Second
	if c.Browser.Timeout <= 0 {
		c.Browser.Timeout = 30 * time.Second
	}
	return c
}

// Config holds the normalized research configuration.
type Config struct {
	Enabled            bool
	Plugin             string
	PluginPath         string
	Auto               bool
	MaxQueries         int
	MaxResultsPerQuery int
	MaxSources         int
	Timeout            time.Duration
	Browser            BrowserConfig
}

// BrowserConfig holds browser-specific research configuration.
type BrowserConfig struct {
	Enabled  bool
	Headless bool
	Timeout  time.Duration
	Extract  bool
}
