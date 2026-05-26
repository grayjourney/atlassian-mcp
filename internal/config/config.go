// Package config loads and persists the Atlassian MCP server's credentials.
//
// The source of truth is a JSON file (default ~/.atlassian-mcp/config.json),
// written by the local setup dashboard. Environment variables of the same names
// used by the Python mcp-atlassian project (JIRA_URL, JIRA_API_TOKEN, ...)
// override the file when set, so the server still works in a plain .mcp.json
// "env" block.
//
// MVP scope: Atlassian Cloud + API-token (basic auth) only. Server/DC, PAT and
// OAuth are intentionally out of scope and tracked as future work.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Config holds Cloud basic-auth credentials for Jira and Confluence.
type Config struct {
	JiraURL            string `json:"jira_url"`
	JiraUsername       string `json:"jira_username"`
	JiraAPIToken       string `json:"jira_api_token"`
	ConfluenceURL      string `json:"confluence_url"`
	ConfluenceUsername string `json:"confluence_username"`
	ConfluenceAPIToken string `json:"confluence_api_token"`
}

// Path returns the config file location. It honors ATLASSIAN_MCP_CONFIG for
// overrides (handy in tests and non-standard installs), otherwise defaults to
// ~/.atlassian-mcp/config.json.
func Path() (string, error) {
	if p := os.Getenv("ATLASSIAN_MCP_CONFIG"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".atlassian-mcp", "config.json"), nil
}

// Load reads the config from the default Path with env overrides applied.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	return LoadFrom(path)
}

// LoadFrom reads config from path (a missing file is not an error, it yields an
// empty config) and then applies environment-variable overrides.
func LoadFrom(path string) (*Config, error) {
	cfg := &Config{}

	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config %s: %w", path, err)
		}
	case errors.Is(err, fs.ErrNotExist):
		// First run, no file yet — fall through to env overrides.
	default:
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	cfg.applyEnv()
	return cfg, nil
}

// applyEnv overlays non-empty environment variables onto the config.
func (c *Config) applyEnv() {
	overlay := func(dst *string, key string) {
		if v := os.Getenv(key); v != "" {
			*dst = v
		}
	}
	overlay(&c.JiraURL, "JIRA_URL")
	overlay(&c.JiraUsername, "JIRA_USERNAME")
	overlay(&c.JiraAPIToken, "JIRA_API_TOKEN")
	overlay(&c.ConfluenceURL, "CONFLUENCE_URL")
	overlay(&c.ConfluenceUsername, "CONFLUENCE_USERNAME")
	overlay(&c.ConfluenceAPIToken, "CONFLUENCE_API_TOKEN")
}

// Save writes the config to the default Path.
func (c *Config) Save() error {
	path, err := Path()
	if err != nil {
		return err
	}
	return c.SaveTo(path)
}

// SaveTo writes the config to path, creating parent dirs (0700) and the file
// with owner-only permissions (0600) since it holds API tokens.
func (c *Config) SaveTo(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}

// JiraConfigured reports whether all Jira credentials are present.
func (c *Config) JiraConfigured() bool {
	return c.JiraURL != "" && c.JiraUsername != "" && c.JiraAPIToken != ""
}

// ConfluenceConfigured reports whether all Confluence credentials are present.
func (c *Config) ConfluenceConfigured() bool {
	return c.ConfluenceURL != "" && c.ConfluenceUsername != "" && c.ConfluenceAPIToken != ""
}

// IsConfigured reports whether at least one service is usable.
func (c *Config) IsConfigured() bool {
	return c.JiraConfigured() || c.ConfluenceConfigured()
}
