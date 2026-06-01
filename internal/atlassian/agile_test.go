package atlassian

import (
	"context"
	"net/http"
	"testing"
)

func TestJiraGetBoards(t *testing.T) {
	cap := &capture{}
	resp := `{"isLast":true,"values":[{"id":1,"name":"KAN board","type":"kanban"},{"id":2,"name":"Scrum","type":"scrum"}]}`
	srv := newServer(t, 200, resp, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	boards, err := jc.GetBoards(context.Background(), "KAN", 50)
	if err != nil {
		t.Fatalf("GetBoards: %v", err)
	}
	if cap.path != "/rest/agile/1.0/board" {
		t.Errorf("path = %q", cap.path)
	}
	if cap.query != "maxResults=50&projectKeyOrId=KAN" {
		t.Errorf("query = %q", cap.query)
	}
	if len(boards) != 2 || boards[0].ID != 1 || boards[0].Name != "KAN board" || boards[0].Type != "kanban" {
		t.Errorf("boards = %+v", boards)
	}
}

func TestJiraGetSprints(t *testing.T) {
	cap := &capture{}
	resp := `{"isLast":true,"values":[{"id":10,"name":"Sprint 1","state":"active","startDate":"2026-05-01","endDate":"2026-05-15","originBoardId":2,"goal":"ship it"}]}`
	srv := newServer(t, 200, resp, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	sprints, err := jc.GetSprints(context.Background(), 2, "active", 50)
	if err != nil {
		t.Fatalf("GetSprints: %v", err)
	}
	if cap.path != "/rest/agile/1.0/board/2/sprint" {
		t.Errorf("path = %q", cap.path)
	}
	if cap.query != "maxResults=50&state=active" {
		t.Errorf("query = %q", cap.query)
	}
	if len(sprints) != 1 || sprints[0].ID != 10 || sprints[0].State != "active" || sprints[0].Goal != "ship it" {
		t.Errorf("sprints = %+v", sprints)
	}
}

func TestJiraGetBoardIssues(t *testing.T) {
	cap := &capture{}
	resp := `{"total":1,"issues":[{"id":"1","key":"KAN-1","fields":{"summary":"hi"}}]}`
	srv := newServer(t, 200, resp, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	issues, err := jc.GetBoardIssues(context.Background(), 1, "status = Done", 25)
	if err != nil {
		t.Fatalf("GetBoardIssues: %v", err)
	}
	if cap.path != "/rest/agile/1.0/board/1/issue" {
		t.Errorf("path = %q", cap.path)
	}
	if cap.query != "jql=status+%3D+Done&maxResults=25" {
		t.Errorf("query = %q", cap.query)
	}
	if len(issues) != 1 || issues[0].Key != "KAN-1" {
		t.Errorf("issues = %+v", issues)
	}
}

func TestJiraGetSprintIssues(t *testing.T) {
	cap := &capture{}
	resp := `{"total":1,"issues":[{"id":"1","key":"KAN-3","fields":{"summary":"x"}}]}`
	srv := newServer(t, 200, resp, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	issues, err := jc.GetSprintIssues(context.Background(), 10, "", 50)
	if err != nil {
		t.Fatalf("GetSprintIssues: %v", err)
	}
	if cap.path != "/rest/agile/1.0/sprint/10/issue" {
		t.Errorf("path = %q", cap.path)
	}
	if len(issues) != 1 || issues[0].Key != "KAN-3" {
		t.Errorf("issues = %+v", issues)
	}
}

func TestJiraCreateSprint(t *testing.T) {
	cap := &capture{}
	srv := newServer(t, 201, `{"id":99,"name":"Sprint X","state":"future","originBoardId":2}`, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	sp, err := jc.CreateSprint(context.Background(), map[string]any{"name": "Sprint X", "originBoardId": 2})
	if err != nil {
		t.Fatalf("CreateSprint: %v", err)
	}
	if cap.method != http.MethodPost || cap.path != "/rest/agile/1.0/sprint" {
		t.Errorf("got %s %s", cap.method, cap.path)
	}
	if cap.body["name"] != "Sprint X" {
		t.Errorf("name = %v", cap.body["name"])
	}
	if sp.ID != 99 {
		t.Errorf("id = %d", sp.ID)
	}
}

func TestJiraUpdateSprint(t *testing.T) {
	cap := &capture{}
	srv := newServer(t, 200, `{"id":10,"name":"Renamed","state":"active"}`, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	_, err := jc.UpdateSprint(context.Background(), 10, map[string]any{"name": "Renamed"})
	if err != nil {
		t.Fatalf("UpdateSprint: %v", err)
	}
	if cap.method != http.MethodPost || cap.path != "/rest/agile/1.0/sprint/10" {
		t.Errorf("got %s %s", cap.method, cap.path)
	}
}

func TestJiraMoveIssuesToSprint(t *testing.T) {
	cap := &capture{}
	srv := newServer(t, 204, ``, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	err := jc.MoveIssuesToSprint(context.Background(), 10, []string{"KAN-1", "KAN-2"})
	if err != nil {
		t.Fatalf("MoveIssuesToSprint: %v", err)
	}
	if cap.method != http.MethodPost || cap.path != "/rest/agile/1.0/sprint/10/issue" {
		t.Errorf("got %s %s", cap.method, cap.path)
	}
	issues, _ := cap.body["issues"].([]any)
	if len(issues) != 2 || issues[0] != "KAN-1" {
		t.Errorf("issues body = %v", cap.body["issues"])
	}
}
