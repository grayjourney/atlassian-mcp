package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListBoardsShaping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"isLast":true,"values":[{"id":1,"name":"KAN board","type":"kanban"}]}`)
	}))
	defer srv.Close()
	s := newToolServer(t, srv)

	res, _, err := s.jiraListBoards(context.Background(), nil, jiraListBoardsInput{Project: "KAN"})
	if err != nil {
		t.Fatalf("jiraListBoards: %v", err)
	}
	m := resultText(t, res)
	if m["count"].(float64) != 1 {
		t.Fatalf("count = %v", m["count"])
	}
	boards := m["boards"].([]any)
	b0 := boards[0].(map[string]any)
	if b0["name"] != "KAN board" || b0["type"] != "kanban" {
		t.Errorf("board = %v", b0)
	}
}

func TestCreateSprintBuildsBody(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":99,"name":"Sprint X","state":"future"}`)
	}))
	defer srv.Close()
	s := newToolServer(t, srv)

	res, _, err := s.jiraCreateSprint(context.Background(), nil, jiraCreateSprintInput{
		BoardID: 2, Name: "Sprint X", Goal: "ship",
	})
	if err != nil {
		t.Fatalf("jiraCreateSprint: %v", err)
	}
	if body["name"] != "Sprint X" || body["goal"] != "ship" || body["originBoardId"].(float64) != 2 {
		t.Errorf("body = %v", body)
	}
	m := resultText(t, res)
	if m["id"].(float64) != 99 {
		t.Errorf("id = %v", m["id"])
	}
}
