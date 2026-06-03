package atlassian

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newMultiServer serves a handler that knows which sequential request it's on
// (n starts at 0), for flows that make several calls.
func newMultiServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request, n int)) *httptest.Server {
	t.Helper()
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		handler(w, r, n)
		n++
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestJiraGetFields(t *testing.T) {
	cap := &capture{}
	resp := `[
		{"id":"summary","name":"Summary","custom":false,"schema":{"type":"string","system":"summary"}},
		{"id":"customfield_10016","name":"Story Points","custom":true,"schema":{"type":"number","custom":"com.pyxis.greenhopper.jira:gh-story-points"}}
	]`
	srv := newServer(t, 200, resp, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	fields, err := jc.GetFields(context.Background())
	if err != nil {
		t.Fatalf("GetFields: %v", err)
	}
	if cap.method != http.MethodGet || cap.path != "/rest/api/3/field" {
		t.Errorf("got %s %s, want GET /rest/api/3/field", cap.method, cap.path)
	}
	if len(fields) != 2 {
		t.Fatalf("len(fields) = %d, want 2", len(fields))
	}
	sp := fields[1]
	if sp.ID != "customfield_10016" || sp.Name != "Story Points" || !sp.Custom || sp.Schema.Type != "number" {
		t.Errorf("story points field = %+v", sp)
	}
}

func TestJiraGetChangelog(t *testing.T) {
	cap := &capture{}
	resp := `{"values":[{"id":"1","author":{"displayName":"Ada"},"created":"2026-01-01T00:00:00.000+0000",
		"items":[{"field":"status","fromString":"To Do","toString":"In Progress"}]}],"total":1,"isLast":true,"startAt":0,"maxResults":100}`
	srv := newServer(t, 200, resp, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	page, err := jc.GetChangelog(context.Background(), "PROJ-1", 0, 50)
	if err != nil {
		t.Fatalf("GetChangelog: %v", err)
	}
	if cap.method != http.MethodGet || cap.path != "/rest/api/3/issue/PROJ-1/changelog" {
		t.Errorf("got %s %s", cap.method, cap.path)
	}
	if cap.query != "maxResults=50&startAt=0" {
		t.Errorf("query = %q", cap.query)
	}
	if len(page.Values) != 1 || page.Values[0].Author.DisplayName != "Ada" {
		t.Fatalf("values = %+v", page.Values)
	}
	if it := page.Values[0].Items; len(it) != 1 || it[0].Field != "status" || it[0].ToString != "In Progress" {
		t.Errorf("items = %+v", page.Values[0].Items)
	}
}

func TestJiraDeleteIssue(t *testing.T) {
	cap := &capture{}
	srv := newServer(t, 204, ``, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	if err := jc.DeleteIssue(context.Background(), "PROJ-1", true); err != nil {
		t.Fatalf("DeleteIssue: %v", err)
	}
	if cap.method != http.MethodDelete || cap.path != "/rest/api/3/issue/PROJ-1" {
		t.Errorf("got %s %s", cap.method, cap.path)
	}
	if cap.query != "deleteSubtasks=true" {
		t.Errorf("query = %q", cap.query)
	}
}

func TestJiraSearchUsers(t *testing.T) {
	cap := &capture{}
	resp := `[{"accountId":"abc-123","displayName":"Ada Lovelace","emailAddress":"ada@acme.com","active":true}]`
	srv := newServer(t, 200, resp, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	users, err := jc.SearchUsers(context.Background(), "ada@acme.com", 5)
	if err != nil {
		t.Fatalf("SearchUsers: %v", err)
	}
	if cap.path != "/rest/api/3/user/search" {
		t.Errorf("path = %q", cap.path)
	}
	if cap.query != "maxResults=5&query=ada%40acme.com" {
		t.Errorf("query = %q", cap.query)
	}
	if len(users) != 1 || users[0].AccountID != "abc-123" || users[0].Email != "ada@acme.com" {
		t.Errorf("users = %+v", users)
	}
}

func TestJiraGetFieldOptions(t *testing.T) {
	// Two-step: GET contexts, then GET options for the first context.
	var paths []string
	srv := newMultiServer(t, func(w http.ResponseWriter, r *http.Request, n int) {
		paths = append(paths, r.URL.Path)
		switch n {
		case 0:
			_, _ = w.Write([]byte(`{"values":[{"id":"10100"}]}`))
		default:
			_, _ = w.Write([]byte(`{"values":[{"id":"1","value":"High"},{"id":"2","value":"Low"}]}`))
		}
	})
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	opts, err := jc.GetFieldOptions(context.Background(), "customfield_10050")
	if err != nil {
		t.Fatalf("GetFieldOptions: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 calls, got %d: %v", len(paths), paths)
	}
	if paths[0] != "/rest/api/3/field/customfield_10050/context" {
		t.Errorf("contexts path = %q", paths[0])
	}
	if paths[1] != "/rest/api/3/field/customfield_10050/context/10100/option" {
		t.Errorf("options path = %q", paths[1])
	}
	if len(opts) != 2 || opts[0].Value != "High" || opts[1].Value != "Low" {
		t.Errorf("opts = %+v", opts)
	}
}
