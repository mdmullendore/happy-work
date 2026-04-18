package config

import (
	"fmt"
	"os"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	// Server
	Port string

	// Jira
	JiraWebhookSecret    string // optional HMAC secret to verify Jira payloads
	JiraTriggerStatus    string // e.g. "In Progress"

	// Google Gemini
	GeminiAPIKey string
	GeminiModel  string

	// Bitbucket
	BitbucketBaseURL   string // e.g. https://api.bitbucket.org/2.0
	BitbucketWorkspace string
	BitbucketAPIKey    string // Bitbucket API key (repository access token)
}

// Load reads configuration from environment variables and returns an error
// if any required variable is missing.
func Load() (*Config, error) {
	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		JiraTriggerStatus:  getEnv("JIRA_TRIGGER_STATUS", "In Progress"),
		JiraWebhookSecret:  os.Getenv("JIRA_WEBHOOK_SECRET"),
		GeminiAPIKey:       os.Getenv("GEMINI_API_KEY"),
		GeminiModel:        getEnv("GEMINI_MODEL", "gemini-2.0-flash"),
		BitbucketBaseURL:   getEnv("BITBUCKET_BASE_URL", "https://api.bitbucket.org/2.0"),
		BitbucketWorkspace: os.Getenv("BITBUCKET_WORKSPACE"),
		BitbucketAPIKey:    os.Getenv("BITBUCKET_API_KEY"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	required := map[string]string{
		"GEMINI_API_KEY":      c.GeminiAPIKey,
		"BITBUCKET_WORKSPACE": c.BitbucketWorkspace,
		"BITBUCKET_API_KEY":   c.BitbucketAPIKey,
	}
	for key, val := range required {
		if val == "" {
			return fmt.Errorf("required environment variable %q is not set", key)
		}
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
