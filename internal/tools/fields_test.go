package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/grayjourney/atlassian-mcp/internal/atlassian"
	"github.com/grayjourney/atlassian-mcp/internal/config"
)

func sampleFields() []atlassian.Field {
	return []atlassian.Field{
		{ID: "summary", Name: "Summary", Schema: atlassian.FieldSchema{Type: "string", System: "summary"}},
		{ID: "duedate", Name: "Due date", Schema: atlassian.FieldSchema{Type: "date", System: "duedate"}},
		{ID: "labels", Name: "Labels", Schema: atlassian.FieldSchema{Type: "array", Items: "string", System: "labels"}},
		{ID: "customfield_10016", Name: "Story Points", Custom: true, Schema: atlassian.FieldSchema{Type: "number", Custom: "com.pyxis.greenhopper.jira:gh-story-points"}},
		{ID: "customfield_10020", Name: "Sprint", Custom: true, Schema: atlassian.FieldSchema{Type: "array", Items: "json", Custom: "com.pyxis.greenhopper.jira:gh-sprint"}},
		{ID: "customfield_10050", Name: "Severity", Custom: true, Schema: atlassian.FieldSchema{Type: "option", Custom: "x:select"}},
		{ID: "customfield_10051", Name: "Team Lead", Custom: true, Schema: atlassian.FieldSchema{Type: "user", Custom: "x:user"}},
		{ID: "customfield_10052", Name: "Squads", Custom: true, Schema: atlassian.FieldSchema{Type: "array", Items: "option", Custom: "x:multiselect"}},
	}
}

func TestFieldResolverResolve(t *testing.T) {
	r := newFieldResolver(sampleFields())
	tests := []struct {
		in     string
		wantID string
		wantOK bool
	}{
		{"Story Points", "customfield_10016", true},
		{"story points", "customfield_10016", true},      // case-insensitive
		{"customfield_10016", "customfield_10016", true}, // by id
		{"Severity", "customfield_10050", true},
		{"Nonexistent Field", "", false},
	}
	for _, tt := range tests {
		f, ok := r.resolve(tt.in)
		if ok != tt.wantOK {
			t.Errorf("resolve(%q) ok=%v want %v", tt.in, ok, tt.wantOK)
			continue
		}
		if ok && f.ID != tt.wantID {
			t.Errorf("resolve(%q) id=%q want %q", tt.in, f.ID, tt.wantID)
		}
	}
}

func TestFieldResolverStoryPoints(t *testing.T) {
	r := newFieldResolver(sampleFields())
	f, ok := r.storyPointsField()
	if !ok || f.ID != "customfield_10016" {
		t.Errorf("storyPointsField() = %+v, %v", f, ok)
	}
}

func TestFormatFieldValue(t *testing.T) {
	r := newFieldResolver(sampleFields())
	get := func(name string) atlassian.Field { f, _ := r.resolve(name); return f }

	tests := []struct {
		name  string
		field atlassian.Field
		value any
		want  any
	}{
		{"number", get("Story Points"), float64(5), float64(5)},
		{"date", get("Due date"), "2026-06-01", "2026-06-01"},
		{"option", get("Severity"), "High", map[string]any{"value": "High"}},
		{"user", get("Team Lead"), "acc-1", map[string]any{"accountId": "acc-1"}},
		{"array-of-string", get("Labels"), []any{"a", "b"}, []any{"a", "b"}},
		{"array-of-option", get("Squads"), []any{"X", "Y"}, []any{map[string]any{"value": "X"}, map[string]any{"value": "Y"}}},
	}
	for _, tt := range tests {
		got := formatFieldValue(tt.field, tt.value)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: formatFieldValue = %#v, want %#v", tt.name, got, tt.want)
		}
	}
}

func TestNameObjects(t *testing.T) {
	got := nameObjects([]string{"Backend", "API"})
	want := []map[string]any{{"name": "Backend"}, {"name": "API"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("nameObjects = %#v, want %#v", got, want)
	}
}

// TestCreateIssueShapesFields drives jiraCreateIssue end-to-end against a fake
// Jira, asserting the typed args + story points + generic fields are shaped and
// the assignee email is resolved to an account id.
func TestCreateIssueShapesFields(t *testing.T) {
	var createBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/rest/api/3/field":
			b, _ := json.Marshal(sampleFields())
			_, _ = w.Write(b)
		case r.URL.Path == "/rest/api/3/user/search":
			_, _ = io.WriteString(w, `[{"accountId":"acc-99","displayName":"Ada","emailAddress":"ada@acme.com","active":true}]`)
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue":
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &createBody)
			_, _ = io.WriteString(w, `{"id":"100","key":"KAN-9","self":"x"}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	cfg := &config.Config{JiraURL: srv.URL, JiraUsername: "u", JiraAPIToken: "t"}
	s := NewServer(func() *config.Config { return cfg }, "http://x")
	s.httpClient = srv.Client()

	sp := 5.0
	_, _, err := s.jiraCreateIssue(context.Background(), nil, jiraCreateIssueInput{
		ProjectKey: "KAN", Summary: "Test", IssueType: "Task",
		DueDate: "2026-06-01", StoryPoints: &sp, Priority: "High",
		Assignee: "ada@acme.com", Labels: []string{"backend"},
		Components: []string{"API"}, Parent: "KAN-1",
		Fields: `{"Severity":"High"}`,
	})
	if err != nil {
		t.Fatalf("jiraCreateIssue: %v", err)
	}

	fields, ok := createBody["fields"].(map[string]any)
	if !ok {
		t.Fatalf("no fields in create body: %v", createBody)
	}
	if fields["duedate"] != "2026-06-01" {
		t.Errorf("duedate = %v", fields["duedate"])
	}
	if fields["customfield_10016"] != 5.0 {
		t.Errorf("story points (customfield_10016) = %v, want 5", fields["customfield_10016"])
	}
	if p, _ := fields["priority"].(map[string]any); p["name"] != "High" {
		t.Errorf("priority = %v", fields["priority"])
	}
	if a, _ := fields["assignee"].(map[string]any); a["accountId"] != "acc-99" {
		t.Errorf("assignee = %v, want accountId acc-99", fields["assignee"])
	}
	if parent, _ := fields["parent"].(map[string]any); parent["key"] != "KAN-1" {
		t.Errorf("parent = %v", fields["parent"])
	}
	if sev, _ := fields["customfield_10050"].(map[string]any); sev["value"] != "High" {
		t.Errorf("Severity (customfield_10050) = %v, want {value:High}", fields["customfield_10050"])
	}
	if labels, _ := fields["labels"].([]any); len(labels) != 1 || labels[0] != "backend" {
		t.Errorf("labels = %v", fields["labels"])
	}
}
