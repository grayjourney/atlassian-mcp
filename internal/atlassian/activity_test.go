package atlassian

import (
	"context"
	"net/http"
	"testing"
)

func TestJiraGetComments(t *testing.T) {
	cap := &capture{}
	resp := `{"comments":[{"id":"1","author":{"displayName":"Ada"},"created":"2026-01-01","updated":"2026-01-02",
		"body":{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"hi"}]}]}}],"total":1}`
	srv := newServer(t, 200, resp, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	cs, err := jc.GetComments(context.Background(), "PROJ-1", 20)
	if err != nil {
		t.Fatalf("GetComments: %v", err)
	}
	if cap.method != http.MethodGet || cap.path != "/rest/api/3/issue/PROJ-1/comment" {
		t.Errorf("got %s %s", cap.method, cap.path)
	}
	if cap.query != "maxResults=20" {
		t.Errorf("query = %q", cap.query)
	}
	if len(cs) != 1 || cs[0].ID != "1" || cs[0].Author.DisplayName != "Ada" {
		t.Errorf("comments = %+v", cs)
	}
}

func TestJiraAddComment(t *testing.T) {
	cap := &capture{}
	srv := newServer(t, 201, `{"id":"55","author":{"displayName":"Me"},"created":"2026-01-01"}`, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	c, err := jc.AddComment(context.Background(), "PROJ-1", map[string]any{"type": "doc"})
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if cap.method != http.MethodPost || cap.path != "/rest/api/3/issue/PROJ-1/comment" {
		t.Errorf("got %s %s", cap.method, cap.path)
	}
	if cap.body["body"] == nil {
		t.Errorf("expected body.body (ADF), got %v", cap.body)
	}
	if c.ID != "55" {
		t.Errorf("id = %q", c.ID)
	}
}

func TestJiraEditComment(t *testing.T) {
	cap := &capture{}
	srv := newServer(t, 200, `{"id":"55","updated":"2026-02-02"}`, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	_, err := jc.EditComment(context.Background(), "PROJ-1", "55", map[string]any{"type": "doc"})
	if err != nil {
		t.Fatalf("EditComment: %v", err)
	}
	if cap.method != http.MethodPut || cap.path != "/rest/api/3/issue/PROJ-1/comment/55" {
		t.Errorf("got %s %s", cap.method, cap.path)
	}
}

func TestJiraGetWorklogs(t *testing.T) {
	cap := &capture{}
	resp := `{"worklogs":[{"id":"9","author":{"displayName":"Ada"},"timeSpent":"2h","timeSpentSeconds":7200,"started":"2026-01-01T09:00:00.000+0000"}],"total":1}`
	srv := newServer(t, 200, resp, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	ws, err := jc.GetWorklogs(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatalf("GetWorklogs: %v", err)
	}
	if cap.path != "/rest/api/3/issue/PROJ-1/worklog" {
		t.Errorf("path = %q", cap.path)
	}
	if len(ws) != 1 || ws[0].TimeSpent != "2h" || ws[0].TimeSpentSeconds != 7200 {
		t.Errorf("worklogs = %+v", ws)
	}
}

func TestJiraAddWorklog(t *testing.T) {
	cap := &capture{}
	srv := newServer(t, 201, `{"id":"9","timeSpent":"2h"}`, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	_, err := jc.AddWorklog(context.Background(), "PROJ-1", map[string]any{"timeSpent": "2h"})
	if err != nil {
		t.Fatalf("AddWorklog: %v", err)
	}
	if cap.method != http.MethodPost || cap.path != "/rest/api/3/issue/PROJ-1/worklog" {
		t.Errorf("got %s %s", cap.method, cap.path)
	}
	if cap.body["timeSpent"] != "2h" {
		t.Errorf("timeSpent = %v", cap.body["timeSpent"])
	}
}

func TestJiraWatchers(t *testing.T) {
	cap := &capture{}
	resp := `{"watchCount":2,"isWatching":true,"watchers":[{"accountId":"a1","displayName":"Ada"},{"accountId":"a2","displayName":"Bob"}]}`
	srv := newServer(t, 200, resp, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	w, err := jc.GetWatchers(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatalf("GetWatchers: %v", err)
	}
	if cap.path != "/rest/api/3/issue/PROJ-1/watchers" {
		t.Errorf("path = %q", cap.path)
	}
	if w.WatchCount != 2 || !w.IsWatching || len(w.Watchers) != 2 {
		t.Errorf("watchers = %+v", w)
	}
}

func TestJiraAddWatcher(t *testing.T) {
	cap := &capture{}
	srv := newServer(t, 204, ``, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	if err := jc.AddWatcher(context.Background(), "PROJ-1", "acc-1"); err != nil {
		t.Fatalf("AddWatcher: %v", err)
	}
	if cap.method != http.MethodPost || cap.path != "/rest/api/3/issue/PROJ-1/watchers" {
		t.Errorf("got %s %s", cap.method, cap.path)
	}
}

func TestJiraRemoveWatcher(t *testing.T) {
	cap := &capture{}
	srv := newServer(t, 204, ``, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	if err := jc.RemoveWatcher(context.Background(), "PROJ-1", "acc-1"); err != nil {
		t.Fatalf("RemoveWatcher: %v", err)
	}
	if cap.method != http.MethodDelete || cap.path != "/rest/api/3/issue/PROJ-1/watchers" {
		t.Errorf("got %s %s", cap.method, cap.path)
	}
	if cap.query != "accountId=acc-1" {
		t.Errorf("query = %q", cap.query)
	}
}

func TestJiraGetUser(t *testing.T) {
	cap := &capture{}
	srv := newServer(t, 200, `{"accountId":"acc-1","displayName":"Ada","emailAddress":"ada@acme.com","active":true}`, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	u, err := jc.GetUser(context.Background(), "acc-1")
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if cap.path != "/rest/api/3/user" || cap.query != "accountId=acc-1" {
		t.Errorf("got %s?%s", cap.path, cap.query)
	}
	if u.AccountID != "acc-1" || u.Email != "ada@acme.com" {
		t.Errorf("user = %+v", u)
	}
}
