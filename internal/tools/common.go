// Package tools registers the MCP tools and adapts them onto the atlassian
// REST client. Outputs are deliberately compact JSON (mirroring the Python
// project's to_simplified_dict) so the model isn't flooded with raw API noise.
package tools

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/grayjourney/atlassian-mcp/internal/atlassian"
	"github.com/grayjourney/atlassian-mcp/internal/config"
	"github.com/grayjourney/atlassian-mcp/internal/content"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server holds the dependencies shared by all tool handlers. Config is read
// through a provider so dashboard edits take effect without a restart.
type Server struct {
	cfg          func() *config.Config
	dashboardURL string
	httpClient   *http.Client
}

// NewServer builds a tool server. cfgProvider must return the latest config.
func NewServer(cfgProvider func() *config.Config, dashboardURL string) *Server {
	return &Server{
		cfg:          cfgProvider,
		dashboardURL: dashboardURL,
		httpClient:   &http.Client{Timeout: 75 * time.Second},
	}
}

// notConfigured returns a user-facing error pointing at the setup dashboard.
func (s *Server) notConfigured(service string) error {
	return fmt.Errorf(
		"%s is not configured. Open the setup dashboard at %s to add your URL, email and API token",
		service, s.dashboardURL,
	)
}

func (s *Server) jira() (*atlassian.JiraClient, *config.Config, error) {
	cfg := s.cfg()
	if !cfg.JiraConfigured() {
		return nil, nil, s.notConfigured("Jira")
	}
	return atlassian.NewJiraClient(cfg.JiraURL, cfg.JiraUsername, cfg.JiraAPIToken, s.httpClient), cfg, nil
}

func (s *Server) confluence() (*atlassian.ConfluenceClient, *config.Config, error) {
	cfg := s.cfg()
	if !cfg.ConfluenceConfigured() {
		return nil, nil, s.notConfigured("Confluence")
	}
	return atlassian.NewConfluenceClient(cfg.ConfluenceURL, cfg.ConfluenceUsername, cfg.ConfluenceAPIToken, s.httpClient), cfg, nil
}

// textResult wraps a value as a pretty-printed JSON text result. Out is typed
// as any so the SDK skips output-schema inference and returns our text as-is.
func textResult(v any) (*mcp.CallToolResult, any, error) {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("marshal result: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(raw)}},
	}, nil, nil
}

// flattenIssue projects a Jira issue into a compact map, optionally rendering
// the ADF description to plain text. Absent fields are omitted entirely.
func flattenIssue(iss atlassian.Issue, includeDescription bool) map[string]any {
	out := map[string]any{"key": iss.Key}
	f := iss.Fields
	if f == nil {
		return out
	}
	if v, ok := f["summary"].(string); ok {
		out["summary"] = v
	}
	if m, ok := f["status"].(map[string]any); ok {
		out["status"] = m["name"]
	}
	if m, ok := f["issuetype"].(map[string]any); ok {
		out["type"] = m["name"]
	}
	if m, ok := f["assignee"].(map[string]any); ok {
		out["assignee"] = m["displayName"]
	}
	if m, ok := f["priority"].(map[string]any); ok {
		out["priority"] = m["name"]
	}
	if v, ok := f["updated"].(string); ok {
		out["updated"] = v
	}
	if v, ok := f["created"].(string); ok {
		out["created"] = v
	}
	if v, ok := f["duedate"].(string); ok {
		out["due_date"] = v
	}
	if m, ok := f["reporter"].(map[string]any); ok {
		out["reporter"] = m["displayName"]
	}
	if m, ok := f["resolution"].(map[string]any); ok {
		out["resolution"] = m["name"]
	}
	if m, ok := f["parent"].(map[string]any); ok {
		out["parent"] = m["key"]
	}
	if v, ok := f["labels"].([]any); ok && len(v) > 0 {
		out["labels"] = v
	}
	if names := objectNames(f["components"]); len(names) > 0 {
		out["components"] = names
	}
	if names := objectNames(f["fixVersions"]); len(names) > 0 {
		out["fix_versions"] = names
	}
	if includeDescription {
		if d := f["description"]; d != nil {
			out["description"] = content.ADFToText(d)
		}
	}
	return out
}

// objectNames pulls the "name" of each object in a Jira array field (components,
// fixVersions, ...). Returns nil for absent or unexpected shapes.
func objectNames(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, e := range arr {
		if m, ok := e.(map[string]any); ok {
			if n, ok := m["name"].(string); ok {
				out = append(out, n)
			}
		}
	}
	return out
}

// resolveTransition finds a transition by id or case-insensitive name.
func resolveTransition(trs []atlassian.Transition, target string) (atlassian.Transition, bool) {
	for _, tr := range trs {
		if tr.ID == target || strings.EqualFold(tr.Name, target) {
			return tr, true
		}
	}
	return atlassian.Transition{}, false
}
