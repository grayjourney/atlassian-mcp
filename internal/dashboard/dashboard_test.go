package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grayjourney/atlassian-mcp/internal/config"
)

func newTestDashboard(t *testing.T) (*Dashboard, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	return New(24285, path, &config.Config{}), path
}

func TestHealthReflectsConfig(t *testing.T) {
	d, _ := newTestDashboard(t)
	srv := httptest.NewServer(d.Handler())
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var h struct {
		Configured bool `json:"configured"`
		Jira       bool `json:"jira"`
		Confluence bool `json:"confluence"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&h); err != nil {
		t.Fatal(err)
	}
	if h.Configured || h.Jira || h.Confluence {
		t.Errorf("fresh dashboard should be unconfigured, got %+v", h)
	}
}

func TestPostConfigPersistsAndReloads(t *testing.T) {
	d, path := newTestDashboard(t)
	srv := httptest.NewServer(d.Handler())
	t.Cleanup(srv.Close)

	body := `{
		"jira_url": "https://acme.atlassian.net",
		"jira_username": "a@acme.com",
		"jira_api_token": "tok"
	}`
	resp, err := http.Post(srv.URL+"/config", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /config status = %d", resp.StatusCode)
	}

	// In-memory config hot-reloaded.
	if !d.Config().JiraConfigured() {
		t.Errorf("dashboard config not reloaded after POST")
	}
	// Persisted to disk.
	saved, err := config.LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if saved.JiraURL != "https://acme.atlassian.net" || saved.JiraAPIToken != "tok" {
		t.Errorf("config not persisted: %+v", saved)
	}
}

func TestIndexServesGuide(t *testing.T) {
	d, _ := newTestDashboard(t)
	srv := httptest.NewServer(d.Handler())
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	buf := make([]byte, 8192)
	n, _ := resp.Body.Read(buf)
	page := string(buf[:n])
	if !strings.Contains(page, "id.atlassian.com/manage-profile/security/api-tokens") {
		t.Errorf("index page missing API-token guide link")
	}
}
