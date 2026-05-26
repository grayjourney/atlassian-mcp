package atlassian

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// capture records the last request a test server received.
type capture struct {
	method string
	path   string
	query  string
	auth   string
	body   map[string]any
}

func newServer(t *testing.T, status int, respBody string, cap *capture) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.method = r.Method
		cap.path = r.URL.Path
		cap.query = r.URL.RawQuery
		cap.auth = r.Header.Get("Authorization")
		if raw, _ := io.ReadAll(r.Body); len(raw) > 0 {
			_ = json.Unmarshal(raw, &cap.body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, respBody)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func wantBasicAuth(t *testing.T, got, user, token string) {
	t.Helper()
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+token))
	if got != want {
		t.Errorf("auth header = %q, want %q", got, want)
	}
}

func TestJiraSearch(t *testing.T) {
	cap := &capture{}
	resp := `{"issues":[{"id":"1","key":"PROJ-1","fields":{"summary":"hello"}}],"nextPageToken":"tok","isLast":false}`
	srv := newServer(t, 200, resp, cap)
	jc := NewJiraClient(srv.URL, "a@acme.com", "tok-jira", srv.Client())

	res, err := jc.Search(context.Background(), SearchRequest{
		JQL: "project = PROJ", Fields: []string{"summary", "status"}, MaxResults: 25,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if cap.method != http.MethodPost || cap.path != "/rest/api/3/search/jql" {
		t.Errorf("got %s %s, want POST /rest/api/3/search/jql", cap.method, cap.path)
	}
	wantBasicAuth(t, cap.auth, "a@acme.com", "tok-jira")
	if cap.body["jql"] != "project = PROJ" {
		t.Errorf("jql in body = %v", cap.body["jql"])
	}
	if cap.body["maxResults"].(float64) != 25 {
		t.Errorf("maxResults = %v", cap.body["maxResults"])
	}
	if len(res.Issues) != 1 || res.Issues[0].Key != "PROJ-1" {
		t.Errorf("issues = %+v", res.Issues)
	}
	if res.NextPageToken != "tok" {
		t.Errorf("nextPageToken = %q", res.NextPageToken)
	}
}

func TestJiraGetIssue(t *testing.T) {
	cap := &capture{}
	srv := newServer(t, 200, `{"id":"1","key":"PROJ-1","fields":{"summary":"hi"}}`, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	issue, err := jc.GetIssue(context.Background(), "PROJ-1", []string{"summary"}, "renderedFields")
	if err != nil {
		t.Fatalf("GetIssue: %v", err)
	}
	if cap.method != http.MethodGet || cap.path != "/rest/api/3/issue/PROJ-1" {
		t.Errorf("got %s %s", cap.method, cap.path)
	}
	if cap.query != "expand=renderedFields&fields=summary" {
		t.Errorf("query = %q", cap.query)
	}
	if issue.Key != "PROJ-1" {
		t.Errorf("key = %q", issue.Key)
	}
}

func TestJiraCreateIssue(t *testing.T) {
	cap := &capture{}
	srv := newServer(t, 201, `{"id":"100","key":"PROJ-7","self":"http://x/100"}`, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	created, err := jc.CreateIssue(context.Background(), map[string]any{
		"project":   map[string]any{"key": "PROJ"},
		"summary":   "New",
		"issuetype": map[string]any{"name": "Task"},
	})
	if err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	if cap.method != http.MethodPost || cap.path != "/rest/api/3/issue" {
		t.Errorf("got %s %s", cap.method, cap.path)
	}
	fields, ok := cap.body["fields"].(map[string]any)
	if !ok || fields["summary"] != "New" {
		t.Errorf("body.fields = %v", cap.body["fields"])
	}
	if created.Key != "PROJ-7" {
		t.Errorf("created.Key = %q", created.Key)
	}
}

func TestJiraUpdateIssue(t *testing.T) {
	cap := &capture{}
	srv := newServer(t, 204, ``, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	err := jc.UpdateIssue(context.Background(), "PROJ-1", map[string]any{"summary": "edited"})
	if err != nil {
		t.Fatalf("UpdateIssue: %v", err)
	}
	if cap.method != http.MethodPut || cap.path != "/rest/api/3/issue/PROJ-1" {
		t.Errorf("got %s %s", cap.method, cap.path)
	}
	fields := cap.body["fields"].(map[string]any)
	if fields["summary"] != "edited" {
		t.Errorf("fields = %v", fields)
	}
}

func TestJiraTransitions(t *testing.T) {
	cap := &capture{}
	srv := newServer(t, 200, `{"transitions":[{"id":"31","name":"Done","to":{"name":"Done"}}]}`, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	trs, err := jc.GetTransitions(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatalf("GetTransitions: %v", err)
	}
	if cap.path != "/rest/api/3/issue/PROJ-1/transitions" {
		t.Errorf("path = %q", cap.path)
	}
	if len(trs) != 1 || trs[0].Name != "Done" || trs[0].ID != "31" {
		t.Errorf("transitions = %+v", trs)
	}
}

func TestJiraTransitionIssue(t *testing.T) {
	cap := &capture{}
	srv := newServer(t, 204, ``, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	err := jc.TransitionIssue(context.Background(), "PROJ-1", "31", nil, nil)
	if err != nil {
		t.Fatalf("TransitionIssue: %v", err)
	}
	if cap.method != http.MethodPost || cap.path != "/rest/api/3/issue/PROJ-1/transitions" {
		t.Errorf("got %s %s", cap.method, cap.path)
	}
	tr := cap.body["transition"].(map[string]any)
	if tr["id"] != "31" {
		t.Errorf("transition.id = %v", tr["id"])
	}
}

func TestAPIErrorMapping(t *testing.T) {
	cap := &capture{}
	srv := newServer(t, 401, `{"errorMessages":["bad creds"]}`, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	_, err := jc.GetIssue(context.Background(), "PROJ-1", nil, "")
	if err == nil {
		t.Fatal("expected error for 401")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("StatusCode = %d", apiErr.StatusCode)
	}
}

func TestConfluenceGetPage(t *testing.T) {
	cap := &capture{}
	resp := `{"id":"123","type":"page","title":"Doc","space":{"key":"DS","name":"Docs"},"version":{"number":3},"body":{"storage":{"value":"<p>hi</p>"}}}`
	srv := newServer(t, 200, resp, cap)
	cc := NewConfluenceClient(srv.URL, "u", "t", srv.Client())

	page, err := cc.GetPage(context.Background(), "123", "body.storage,version,space")
	if err != nil {
		t.Fatalf("GetPage: %v", err)
	}
	if cap.path != "/rest/api/content/123" {
		t.Errorf("path = %q", cap.path)
	}
	if cap.query != "expand=body.storage%2Cversion%2Cspace" {
		t.Errorf("query = %q", cap.query)
	}
	if page.Title != "Doc" || page.Version.Number != 3 || page.Body.Storage.Value != "<p>hi</p>" {
		t.Errorf("page = %+v", page)
	}
}

func TestConfluenceCreatePage(t *testing.T) {
	cap := &capture{}
	srv := newServer(t, 200, `{"id":"500","type":"page","title":"New","_links":{"webui":"/x"}}`, cap)
	cc := NewConfluenceClient(srv.URL, "u", "t", srv.Client())

	page, err := cc.CreatePage(context.Background(), "DS", "New", "<p>body</p>", "999")
	if err != nil {
		t.Fatalf("CreatePage: %v", err)
	}
	if cap.method != http.MethodPost || cap.path != "/rest/api/content" {
		t.Errorf("got %s %s", cap.method, cap.path)
	}
	if cap.body["type"] != "page" || cap.body["title"] != "New" {
		t.Errorf("body = %v", cap.body)
	}
	space := cap.body["space"].(map[string]any)
	if space["key"] != "DS" {
		t.Errorf("space.key = %v", space["key"])
	}
	ancestors := cap.body["ancestors"].([]any)
	if len(ancestors) != 1 || ancestors[0].(map[string]any)["id"] != "999" {
		t.Errorf("ancestors = %v", cap.body["ancestors"])
	}
	if page.ID != "500" {
		t.Errorf("page.ID = %q", page.ID)
	}
}

func TestConfluenceUpdatePageBumpsVersion(t *testing.T) {
	cap := &capture{}
	srv := newServer(t, 200, `{"id":"123","type":"page","title":"Doc","version":{"number":4}}`, cap)
	cc := NewConfluenceClient(srv.URL, "u", "t", srv.Client())

	_, err := cc.UpdatePage(context.Background(), "123", "Doc", "<p>new</p>", 3)
	if err != nil {
		t.Fatalf("UpdatePage: %v", err)
	}
	if cap.method != http.MethodPut || cap.path != "/rest/api/content/123" {
		t.Errorf("got %s %s", cap.method, cap.path)
	}
	version := cap.body["version"].(map[string]any)
	if version["number"].(float64) != 4 {
		t.Errorf("version.number = %v, want 4 (prev+1)", version["number"])
	}
}

func TestConfluenceAddComment(t *testing.T) {
	cap := &capture{}
	srv := newServer(t, 200, `{"id":"c1","type":"comment"}`, cap)
	cc := NewConfluenceClient(srv.URL, "u", "t", srv.Client())

	_, err := cc.AddComment(context.Background(), "123", "<p>nice</p>")
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if cap.body["type"] != "comment" {
		t.Errorf("type = %v", cap.body["type"])
	}
	container := cap.body["container"].(map[string]any)
	if container["id"] != "123" || container["type"] != "page" {
		t.Errorf("container = %v", container)
	}
}

func TestConfluenceSearch(t *testing.T) {
	cap := &capture{}
	resp := `{"results":[{"content":{"id":"1","type":"page","title":"A"},"title":"A","excerpt":"e","url":"/x"}],"size":1}`
	srv := newServer(t, 200, resp, cap)
	cc := NewConfluenceClient(srv.URL, "u", "t", srv.Client())

	res, err := cc.Search(context.Background(), "type=page", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if cap.path != "/rest/api/search" {
		t.Errorf("path = %q", cap.path)
	}
	if cap.query != "cql=type%3Dpage&limit=10" {
		t.Errorf("query = %q", cap.query)
	}
	if len(res.Results) != 1 || res.Results[0].Content.ID != "1" {
		t.Errorf("results = %+v", res.Results)
	}
}
