package tools

import (
	"testing"

	"github.com/grayjourney/atlassian-mcp/internal/atlassian"
)

func TestFlattenIssue(t *testing.T) {
	iss := atlassian.Issue{
		Key: "PROJ-1",
		Fields: map[string]any{
			"summary":   "Fix the thing",
			"status":    map[string]any{"name": "In Progress"},
			"issuetype": map[string]any{"name": "Bug"},
			"assignee":  map[string]any{"displayName": "Ada L."},
			"priority":  map[string]any{"name": "High"},
			"description": map[string]any{
				"type":    "doc",
				"version": 1,
				"content": []any{map[string]any{"type": "paragraph",
					"content": []any{map[string]any{"type": "text", "text": "details here"}}}},
			},
		},
	}
	got := flattenIssue(iss, true)

	if got["key"] != "PROJ-1" || got["summary"] != "Fix the thing" {
		t.Errorf("key/summary = %v / %v", got["key"], got["summary"])
	}
	if got["status"] != "In Progress" || got["type"] != "Bug" {
		t.Errorf("status/type = %v / %v", got["status"], got["type"])
	}
	if got["assignee"] != "Ada L." || got["priority"] != "High" {
		t.Errorf("assignee/priority = %v / %v", got["assignee"], got["priority"])
	}
	if got["description"] != "details here" {
		t.Errorf("description = %q", got["description"])
	}
}

func TestFlattenIssueOmitsDescriptionWhenAsked(t *testing.T) {
	iss := atlassian.Issue{Key: "P-1", Fields: map[string]any{"summary": "s"}}
	got := flattenIssue(iss, false)
	if _, ok := got["description"]; ok {
		t.Errorf("description should be omitted")
	}
	// Missing fields should simply be absent, not nil entries.
	if _, ok := got["assignee"]; ok {
		t.Errorf("absent assignee should not appear")
	}
}

func TestResolveTransition(t *testing.T) {
	trs := []atlassian.Transition{
		{ID: "11", Name: "To Do"},
		{ID: "21", Name: "In Progress"},
		{ID: "31", Name: "Done"},
	}
	tests := []struct {
		target string
		wantID string
		wantOK bool
	}{
		{"Done", "31", true},
		{"done", "31", true}, // case-insensitive name
		{"21", "21", true},   // by id
		{"In Progress", "21", true},
		{"Nonexistent", "", false},
	}
	for _, tt := range tests {
		got, ok := resolveTransition(trs, tt.target)
		if ok != tt.wantOK {
			t.Errorf("resolveTransition(%q) ok = %v, want %v", tt.target, ok, tt.wantOK)
			continue
		}
		if ok && got.ID != tt.wantID {
			t.Errorf("resolveTransition(%q) id = %q, want %q", tt.target, got.ID, tt.wantID)
		}
	}
}
