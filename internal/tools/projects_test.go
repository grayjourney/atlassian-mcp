package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grayjourney/atlassian-mcp/internal/atlassian"
)

func TestEpicLinkField(t *testing.T) {
	withEpic := newFieldResolver(append(sampleFields(),
		atlassian.Field{ID: "customfield_10014", Name: "Epic Link", Custom: true,
			Schema: atlassian.FieldSchema{Type: "any", Custom: "com.pyxis.greenhopper.jira:gh-epic-link"}}))
	f, ok := epicLinkField(withEpic)
	if !ok || f.ID != "customfield_10014" {
		t.Errorf("epicLinkField = %+v, %v", f, ok)
	}
	if _, ok := epicLinkField(newFieldResolver(sampleFields())); ok {
		t.Errorf("expected no epic link field in sampleFields")
	}
}

// TestLinkToEpicFallsBackToParent verifies that, with no Epic Link field, the
// issue is updated via the parent field.
func TestLinkToEpicFallsBackToParent(t *testing.T) {
	var updateBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/rest/api/3/field":
			b, _ := json.Marshal(sampleFields()) // no epic link field
			_, _ = w.Write(b)
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/"):
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &updateBody)
			w.WriteHeader(204)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()
	s := newToolServer(t, srv)

	res, _, err := s.jiraLinkToEpic(context.Background(), nil, jiraLinkToEpicInput{IssueKey: "KAN-3", EpicKey: "KAN-9"})
	if err != nil {
		t.Fatalf("jiraLinkToEpic: %v", err)
	}
	fields, _ := updateBody["fields"].(map[string]any)
	parent, _ := fields["parent"].(map[string]any)
	if parent["key"] != "KAN-9" {
		t.Errorf("parent = %v, want key KAN-9", fields["parent"])
	}
	m := resultText(t, res)
	if m["via"] != "parent" {
		t.Errorf("via = %v, want parent", m["via"])
	}
}

func TestCreateVersionResolvesProjectID(t *testing.T) {
	var versionBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/rest/api/3/project/KAN":
			_, _ = io.WriteString(w, `{"id":"10000","key":"KAN","name":"Kanban"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/version":
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &versionBody)
			_, _ = io.WriteString(w, `{"id":"55","name":"v2.0","projectId":10000}`)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()
	s := newToolServer(t, srv)

	_, _, err := s.jiraCreateVersion(context.Background(), nil, jiraCreateVersionInput{
		ProjectKey: "KAN", Name: "v2.0", ReleaseDate: "2026-07-01",
	})
	if err != nil {
		t.Fatalf("jiraCreateVersion: %v", err)
	}
	if versionBody["projectId"].(float64) != 10000 {
		t.Errorf("projectId = %v, want 10000 (resolved from key)", versionBody["projectId"])
	}
	if versionBody["name"] != "v2.0" || versionBody["releaseDate"] != "2026-07-01" {
		t.Errorf("body = %v", versionBody)
	}
}
