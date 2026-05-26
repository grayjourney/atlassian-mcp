package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromMissingFileReturnsEmpty(t *testing.T) {
	clearAtlassianEnv(t)
	path := filepath.Join(t.TempDir(), "does-not-exist.json")

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom missing file: unexpected error %v", err)
	}
	if cfg.IsConfigured() {
		t.Fatalf("empty config should not be configured")
	}
}

func TestLoadFromReadsFile(t *testing.T) {
	clearAtlassianEnv(t)
	path := filepath.Join(t.TempDir(), "config.json")
	const body = `{
		"jira_url": "https://acme.atlassian.net",
		"jira_username": "a@acme.com",
		"jira_api_token": "tok-jira",
		"confluence_url": "https://acme.atlassian.net/wiki",
		"confluence_username": "a@acme.com",
		"confluence_api_token": "tok-conf"
	}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.JiraURL != "https://acme.atlassian.net" {
		t.Errorf("JiraURL = %q", cfg.JiraURL)
	}
	if cfg.ConfluenceAPIToken != "tok-conf" {
		t.Errorf("ConfluenceAPIToken = %q", cfg.ConfluenceAPIToken)
	}
	if !cfg.JiraConfigured() || !cfg.ConfluenceConfigured() {
		t.Errorf("expected both services configured")
	}
}

func TestEnvOverridesFile(t *testing.T) {
	clearAtlassianEnv(t)
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"jira_url":"https://from-file.net"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JIRA_URL", "https://from-env.net")
	t.Setenv("JIRA_USERNAME", "env@acme.com")
	t.Setenv("JIRA_API_TOKEN", "env-tok")

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.JiraURL != "https://from-env.net" {
		t.Errorf("env should override file: JiraURL = %q", cfg.JiraURL)
	}
	if !cfg.JiraConfigured() {
		t.Errorf("expected Jira configured from env")
	}
}

func TestIsConfigured(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		jira bool
		conf bool
		any  bool
	}{
		{"empty", Config{}, false, false, false},
		{
			"jira only",
			Config{JiraURL: "u", JiraUsername: "n", JiraAPIToken: "t"},
			true, false, true,
		},
		{
			"jira missing token",
			Config{JiraURL: "u", JiraUsername: "n"},
			false, false, false,
		},
		{
			"confluence only",
			Config{ConfluenceURL: "u", ConfluenceUsername: "n", ConfluenceAPIToken: "t"},
			false, true, true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.JiraConfigured(); got != tt.jira {
				t.Errorf("JiraConfigured = %v, want %v", got, tt.jira)
			}
			if got := tt.cfg.ConfluenceConfigured(); got != tt.conf {
				t.Errorf("ConfluenceConfigured = %v, want %v", got, tt.conf)
			}
			if got := tt.cfg.IsConfigured(); got != tt.any {
				t.Errorf("IsConfigured = %v, want %v", got, tt.any)
			}
		})
	}
}

func TestSaveToRoundTripAndPerms(t *testing.T) {
	clearAtlassianEnv(t)
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	in := &Config{JiraURL: "https://acme.atlassian.net", JiraUsername: "a@acme.com", JiraAPIToken: "t"}

	if err := in.SaveTo(path); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config file perm = %o, want 600", perm)
	}

	out, err := LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if *out != *in {
		t.Errorf("round trip mismatch: got %+v want %+v", out, in)
	}
}

// clearAtlassianEnv ensures env overrides don't leak between tests.
func clearAtlassianEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"JIRA_URL", "JIRA_USERNAME", "JIRA_API_TOKEN",
		"CONFLUENCE_URL", "CONFLUENCE_USERNAME", "CONFLUENCE_API_TOKEN",
	} {
		t.Setenv(k, "")
	}
}
