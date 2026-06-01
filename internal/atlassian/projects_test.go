package atlassian

import (
	"context"
	"net/http"
	"testing"
)

func TestJiraGetProjects(t *testing.T) {
	cap := &capture{}
	resp := `{"values":[{"id":"10000","key":"KAN","name":"Kanban","projectTypeKey":"software"}]}`
	srv := newServer(t, 200, resp, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	ps, err := jc.GetProjects(context.Background(), "kan", 50)
	if err != nil {
		t.Fatalf("GetProjects: %v", err)
	}
	if cap.path != "/rest/api/3/project/search" {
		t.Errorf("path = %q", cap.path)
	}
	if cap.query != "maxResults=50&query=kan" {
		t.Errorf("query = %q", cap.query)
	}
	if len(ps) != 1 || ps[0].Key != "KAN" || ps[0].Name != "Kanban" {
		t.Errorf("projects = %+v", ps)
	}
}

func TestJiraGetProject(t *testing.T) {
	cap := &capture{}
	srv := newServer(t, 200, `{"id":"10000","key":"KAN","name":"Kanban"}`, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	p, err := jc.GetProject(context.Background(), "KAN")
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if cap.path != "/rest/api/3/project/KAN" {
		t.Errorf("path = %q", cap.path)
	}
	if p.ID != "10000" {
		t.Errorf("id = %q", p.ID)
	}
}

func TestJiraGetProjectVersions(t *testing.T) {
	cap := &capture{}
	resp := `[{"id":"1","name":"v1.0","released":true,"archived":false,"releaseDate":"2026-01-01"}]`
	srv := newServer(t, 200, resp, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	vs, err := jc.GetProjectVersions(context.Background(), "KAN")
	if err != nil {
		t.Fatalf("GetProjectVersions: %v", err)
	}
	if cap.path != "/rest/api/3/project/KAN/versions" {
		t.Errorf("path = %q", cap.path)
	}
	if len(vs) != 1 || vs[0].Name != "v1.0" || !vs[0].Released {
		t.Errorf("versions = %+v", vs)
	}
}

func TestJiraCreateVersion(t *testing.T) {
	cap := &capture{}
	srv := newServer(t, 201, `{"id":"55","name":"v2.0","projectId":10000}`, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	v, err := jc.CreateVersion(context.Background(), map[string]any{"name": "v2.0", "projectId": 10000})
	if err != nil {
		t.Fatalf("CreateVersion: %v", err)
	}
	if cap.method != http.MethodPost || cap.path != "/rest/api/3/version" {
		t.Errorf("got %s %s", cap.method, cap.path)
	}
	if cap.body["name"] != "v2.0" {
		t.Errorf("name = %v", cap.body["name"])
	}
	if v.ID != "55" {
		t.Errorf("id = %q", v.ID)
	}
}

func TestJiraGetProjectComponents(t *testing.T) {
	cap := &capture{}
	resp := `[{"id":"1","name":"API","description":"backend"}]`
	srv := newServer(t, 200, resp, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	cs, err := jc.GetProjectComponents(context.Background(), "KAN")
	if err != nil {
		t.Fatalf("GetProjectComponents: %v", err)
	}
	if cap.path != "/rest/api/3/project/KAN/components" {
		t.Errorf("path = %q", cap.path)
	}
	if len(cs) != 1 || cs[0].Name != "API" {
		t.Errorf("components = %+v", cs)
	}
}

func TestJiraGetIssueLinkTypes(t *testing.T) {
	cap := &capture{}
	resp := `{"issueLinkTypes":[{"id":"1","name":"Blocks","inward":"is blocked by","outward":"blocks"}]}`
	srv := newServer(t, 200, resp, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	ts, err := jc.GetIssueLinkTypes(context.Background())
	if err != nil {
		t.Fatalf("GetIssueLinkTypes: %v", err)
	}
	if cap.path != "/rest/api/3/issueLinkType" {
		t.Errorf("path = %q", cap.path)
	}
	if len(ts) != 1 || ts[0].Name != "Blocks" || ts[0].Outward != "blocks" {
		t.Errorf("types = %+v", ts)
	}
}

func TestJiraCreateIssueLink(t *testing.T) {
	cap := &capture{}
	srv := newServer(t, 201, ``, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	err := jc.CreateIssueLink(context.Background(), "Blocks", "KAN-1", "KAN-2")
	if err != nil {
		t.Fatalf("CreateIssueLink: %v", err)
	}
	if cap.method != http.MethodPost || cap.path != "/rest/api/3/issueLink" {
		t.Errorf("got %s %s", cap.method, cap.path)
	}
	typ, _ := cap.body["type"].(map[string]any)
	if typ["name"] != "Blocks" {
		t.Errorf("type = %v", cap.body["type"])
	}
	inw, _ := cap.body["inwardIssue"].(map[string]any)
	outw, _ := cap.body["outwardIssue"].(map[string]any)
	if inw["key"] != "KAN-1" || outw["key"] != "KAN-2" {
		t.Errorf("links = %v / %v", inw, outw)
	}
}

func TestJiraDeleteIssueLink(t *testing.T) {
	cap := &capture{}
	srv := newServer(t, 204, ``, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	if err := jc.DeleteIssueLink(context.Background(), "9000"); err != nil {
		t.Fatalf("DeleteIssueLink: %v", err)
	}
	if cap.method != http.MethodDelete || cap.path != "/rest/api/3/issueLink/9000" {
		t.Errorf("got %s %s", cap.method, cap.path)
	}
}

func TestJiraCreateRemoteLink(t *testing.T) {
	cap := &capture{}
	srv := newServer(t, 201, `{"id":1,"self":"x"}`, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	err := jc.CreateRemoteLink(context.Background(), "KAN-1", "https://example.com", "Example")
	if err != nil {
		t.Fatalf("CreateRemoteLink: %v", err)
	}
	if cap.method != http.MethodPost || cap.path != "/rest/api/3/issue/KAN-1/remotelink" {
		t.Errorf("got %s %s", cap.method, cap.path)
	}
	obj, _ := cap.body["object"].(map[string]any)
	if obj["url"] != "https://example.com" || obj["title"] != "Example" {
		t.Errorf("object = %v", cap.body["object"])
	}
}
