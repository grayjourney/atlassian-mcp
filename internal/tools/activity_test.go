package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grayjourney/atlassian-mcp/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// resultText decodes the JSON text payload of a tool result into a map.
func resultText(t *testing.T, res *mcp.CallToolResult) map[string]any {
	t.Helper()
	if res == nil || len(res.Content) == 0 {
		t.Fatal("empty tool result")
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content[0] is %T, want *mcp.TextContent", res.Content[0])
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(tc.Text), &m); err != nil {
		t.Fatalf("decode result text: %v", err)
	}
	return m
}

func newToolServer(t *testing.T, srv *httptest.Server) *Server {
	t.Helper()
	cfg := &config.Config{JiraURL: srv.URL, JiraUsername: "u", JiraAPIToken: "t"}
	s := NewServer(func() *config.Config { return cfg }, "http://x")
	s.httpClient = srv.Client()
	return s
}

// TestAddCommentSendsADF verifies the Markdown comment is converted to an ADF
// body before being posted.
func TestAddCommentSendsADF(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"77","author":{"displayName":"Me"},"created":"2026-01-01"}`)
	}))
	defer srv.Close()
	s := newToolServer(t, srv)

	res, _, err := s.jiraAddComment(context.Background(), nil, jiraAddCommentInput{
		IssueKey: "KAN-1", Comment: "looks **good**",
	})
	if err != nil {
		t.Fatalf("jiraAddComment: %v", err)
	}
	adf, ok := body["body"].(map[string]any)
	if !ok || adf["type"] != "doc" {
		t.Fatalf("comment body is not an ADF doc: %v", body["body"])
	}
	if res == nil {
		t.Fatal("nil result")
	}
}

// TestGetIssueDatesCollectsCustomDates verifies date/datetime custom fields are
// discovered and surfaced under their human names.
func TestGetIssueDatesCollectsCustomDates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/api/3/field":
			b, _ := json.Marshal(sampleFields()) // includes "Start date" (customfield_10015? no)
			_, _ = w.Write(b)
		default: // GET issue
			_, _ = io.WriteString(w, `{"key":"KAN-1","fields":{"created":"2026-01-01","duedate":"2026-06-01"}}`)
		}
	}))
	defer srv.Close()
	s := newToolServer(t, srv)

	res, _, err := s.jiraGetIssueDates(context.Background(), nil, jiraGetIssueDatesInput{IssueKey: "KAN-1"})
	if err != nil {
		t.Fatalf("jiraGetIssueDates: %v", err)
	}
	m := resultText(t, res)
	dates, ok := m["dates"].(map[string]any)
	if !ok {
		t.Fatalf("no dates map: %v", m)
	}
	if dates["Due date"] != "2026-06-01" || dates["Created"] != "2026-01-01" {
		t.Errorf("dates = %v", dates)
	}
}
