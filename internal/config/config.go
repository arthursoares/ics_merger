package config

import (
	"encoding/json"
	"os"
)

// Calendar represents a single calendar source
type Calendar struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Config holds the application configuration
type Config struct {
	Calendars          []Calendar `json:"calendars"`
	OutputPath         string     `json:"outputPath"`
	SyncIntervalMinutes int       `json:"syncIntervalMinutes"`
	OutputTimezone     string     `json:"outputTimezone"`
}

// Load reads configuration from the config file
func Load() (*Config, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "/app/config.json"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Set default interval if not specified
	if cfg.SyncIntervalMinutes <= 0 {
		cfg.SyncIntervalMinutes = 15
	}

	// Set default output path if not specified
	if cfg.OutputPath == "" {
		cfg.OutputPath = "/app/output/merged.ics"
	}
	
	// Set default timezone if not specified or check environment variable
	if cfg.OutputTimezone == "" {
		// Check if environment variable is set
		if envTz := os.Getenv("OUTPUT_TIMEZONE"); envTz != "" {
			cfg.OutputTimezone = envTz
		} else {
			cfg.OutputTimezone = "Europe/Berlin" // Default timezone
		}
	}

	return &cfg, nil
}