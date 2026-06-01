package tools

import (
	"context"
	"fmt"

	"github.com/grayjourney/atlassian-mcp/internal/atlassian"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func boardMeta(b atlassian.Board) map[string]any {
	return map[string]any{"id": b.ID, "name": b.Name, "type": b.Type}
}

func sprintMeta(sp atlassian.Sprint) map[string]any {
	m := map[string]any{"id": sp.ID, "name": sp.Name, "state": sp.State}
	if sp.Goal != "" {
		m["goal"] = sp.Goal
	}
	if sp.StartDate != "" {
		m["start_date"] = sp.StartDate
	}
	if sp.EndDate != "" {
		m["end_date"] = sp.EndDate
	}
	return m
}

func issueList(issues []atlassian.Issue) []map[string]any {
	out := make([]map[string]any, 0, len(issues))
	for _, iss := range issues {
		out = append(out, flattenIssue(iss, false))
	}
	return out
}

func clampLimit(n, def, max int) int {
	if n <= 0 || n > max {
		return def
	}
	return n
}

// --- jira_list_boards ---

type jiraListBoardsInput struct {
	Project string `json:"project,omitempty" jsonschema:"filter to a project key or id"`
	Limit   int    `json:"limit,omitempty" jsonschema:"max boards (1-100), default 50"`
}

func (s *Server) jiraListBoards(ctx context.Context, _ *mcp.CallToolRequest, in jiraListBoardsInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	boards, err := jc.GetBoards(ctx, in.Project, clampLimit(in.Limit, 50, 100))
	if err != nil {
		return nil, nil, err
	}
	out := make([]map[string]any, 0, len(boards))
	for _, b := range boards {
		out = append(out, boardMeta(b))
	}
	return textResult(map[string]any{"boards": out, "count": len(out)})
}

// --- jira_get_board_issues ---

type jiraBoardIssuesInput struct {
	BoardID int    `json:"board_id" jsonschema:"the board id"`
	JQL     string `json:"jql,omitempty" jsonschema:"optional JQL to narrow the board's issues"`
	Limit   int    `json:"limit,omitempty" jsonschema:"max issues (1-50), default 25"`
}

func (s *Server) jiraGetBoardIssues(ctx context.Context, _ *mcp.CallToolRequest, in jiraBoardIssuesInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	issues, err := jc.GetBoardIssues(ctx, in.BoardID, in.JQL, clampLimit(in.Limit, 25, 50))
	if err != nil {
		return nil, nil, err
	}
	out := issueList(issues)
	return textResult(map[string]any{"board_id": in.BoardID, "issues": out, "count": len(out)})
}

// --- jira_list_sprints ---

type jiraListSprintsInput struct {
	BoardID int    `json:"board_id" jsonschema:"the board id (must be a Scrum board)"`
	State   string `json:"state,omitempty" jsonschema:"filter: active, future, closed, or a comma list"`
	Limit   int    `json:"limit,omitempty" jsonschema:"max sprints (1-100), default 50"`
}

func (s *Server) jiraListSprints(ctx context.Context, _ *mcp.CallToolRequest, in jiraListSprintsInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	sprints, err := jc.GetSprints(ctx, in.BoardID, in.State, clampLimit(in.Limit, 50, 100))
	if err != nil {
		return nil, nil, err
	}
	out := make([]map[string]any, 0, len(sprints))
	for _, sp := range sprints {
		out = append(out, sprintMeta(sp))
	}
	return textResult(map[string]any{"board_id": in.BoardID, "sprints": out, "count": len(out)})
}

// --- jira_get_active_sprint ---

type jiraActiveSprintInput struct {
	BoardID int `json:"board_id" jsonschema:"the board id (must be a Scrum board)"`
}

func (s *Server) jiraGetActiveSprint(ctx context.Context, _ *mcp.CallToolRequest, in jiraActiveSprintInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	sprints, err := jc.GetSprints(ctx, in.BoardID, "active", 10)
	if err != nil {
		return nil, nil, err
	}
	out := make([]map[string]any, 0, len(sprints))
	for _, sp := range sprints {
		out = append(out, sprintMeta(sp))
	}
	return textResult(map[string]any{"board_id": in.BoardID, "active_sprints": out, "count": len(out)})
}

// --- jira_get_sprint_issues ---

type jiraSprintIssuesInput struct {
	SprintID int    `json:"sprint_id" jsonschema:"the sprint id"`
	JQL      string `json:"jql,omitempty" jsonschema:"optional JQL to narrow the sprint's issues"`
	Limit    int    `json:"limit,omitempty" jsonschema:"max issues (1-50), default 25"`
}

func (s *Server) jiraGetSprintIssues(ctx context.Context, _ *mcp.CallToolRequest, in jiraSprintIssuesInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	issues, err := jc.GetSprintIssues(ctx, in.SprintID, in.JQL, clampLimit(in.Limit, 25, 50))
	if err != nil {
		return nil, nil, err
	}
	out := issueList(issues)
	return textResult(map[string]any{"sprint_id": in.SprintID, "issues": out, "count": len(out)})
}

// --- jira_create_sprint ---

type jiraCreateSprintInput struct {
	BoardID   int    `json:"board_id" jsonschema:"origin board id (Scrum board)"`
	Name      string `json:"name" jsonschema:"sprint name"`
	Goal      string `json:"goal,omitempty" jsonschema:"sprint goal"`
	StartDate string `json:"start_date,omitempty" jsonschema:"ISO8601 start, e.g. 2026-06-01T00:00:00.000+0000"`
	EndDate   string `json:"end_date,omitempty" jsonschema:"ISO8601 end"`
}

func (s *Server) jiraCreateSprint(ctx context.Context, _ *mcp.CallToolRequest, in jiraCreateSprintInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	body := map[string]any{"name": in.Name, "originBoardId": in.BoardID}
	if in.Goal != "" {
		body["goal"] = in.Goal
	}
	if in.StartDate != "" {
		body["startDate"] = in.StartDate
	}
	if in.EndDate != "" {
		body["endDate"] = in.EndDate
	}
	sp, err := jc.CreateSprint(ctx, body)
	if err != nil {
		return nil, nil, err
	}
	return textResult(sprintMeta(*sp))
}

// --- jira_update_sprint ---

type jiraUpdateSprintInput struct {
	SprintID  int    `json:"sprint_id" jsonschema:"the sprint id"`
	Name      string `json:"name,omitempty" jsonschema:"new name"`
	Goal      string `json:"goal,omitempty" jsonschema:"new goal"`
	State     string `json:"state,omitempty" jsonschema:"new state: future, active, or closed"`
	StartDate string `json:"start_date,omitempty" jsonschema:"ISO8601 start"`
	EndDate   string `json:"end_date,omitempty" jsonschema:"ISO8601 end"`
}

func (s *Server) jiraUpdateSprint(ctx context.Context, _ *mcp.CallToolRequest, in jiraUpdateSprintInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	body := map[string]any{}
	if in.Name != "" {
		body["name"] = in.Name
	}
	if in.Goal != "" {
		body["goal"] = in.Goal
	}
	if in.State != "" {
		body["state"] = in.State
	}
	if in.StartDate != "" {
		body["startDate"] = in.StartDate
	}
	if in.EndDate != "" {
		body["endDate"] = in.EndDate
	}
	if len(body) == 0 {
		return nil, nil, fmt.Errorf("nothing to update: provide a field to change")
	}
	sp, err := jc.UpdateSprint(ctx, in.SprintID, body)
	if err != nil {
		return nil, nil, err
	}
	return textResult(sprintMeta(*sp))
}

// --- jira_move_issues_to_sprint ---

type jiraMoveIssuesInput struct {
	SprintID  int      `json:"sprint_id" jsonschema:"the target sprint id"`
	IssueKeys []string `json:"issue_keys" jsonschema:"issue keys to move, e.g. [\"KAN-1\",\"KAN-2\"]"`
}

func (s *Server) jiraMoveIssuesToSprint(ctx context.Context, _ *mcp.CallToolRequest, in jiraMoveIssuesInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	if len(in.IssueKeys) == 0 {
		return nil, nil, fmt.Errorf("issue_keys must not be empty")
	}
	if err := jc.MoveIssuesToSprint(ctx, in.SprintID, in.IssueKeys); err != nil {
		return nil, nil, err
	}
	return textResult(map[string]any{"sprint_id": in.SprintID, "moved": in.IssueKeys, "count": len(in.IssueKeys)})
}
