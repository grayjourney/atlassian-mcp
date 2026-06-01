package atlassian

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// Board is a Jira agile board (Scrum or Kanban).
type Board struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// Sprint is a Jira agile sprint.
type Sprint struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	State         string `json:"state"`
	StartDate     string `json:"startDate"`
	EndDate       string `json:"endDate"`
	CompleteDate  string `json:"completeDate"`
	Goal          string `json:"goal"`
	OriginBoardID int    `json:"originBoardId"`
}

// GetBoards lists agile boards, optionally filtered to a project.
func (j *JiraClient) GetBoards(ctx context.Context, projectKeyOrID string, maxResults int) ([]Board, error) {
	q := url.Values{}
	if projectKeyOrID != "" {
		q.Set("projectKeyOrId", projectKeyOrID)
	}
	if maxResults > 0 {
		q.Set("maxResults", strconv.Itoa(maxResults))
	}
	var out struct {
		Values []Board `json:"values"`
	}
	if err := j.rc.do(ctx, http.MethodGet, "/rest/agile/1.0/board", q, nil, &out); err != nil {
		return nil, err
	}
	return out.Values, nil
}

// GetSprints lists a board's sprints, optionally filtered by state
// (active, future, closed, or a comma list). Kanban boards return an error.
func (j *JiraClient) GetSprints(ctx context.Context, boardID int, state string, maxResults int) ([]Sprint, error) {
	q := url.Values{}
	if state != "" {
		q.Set("state", state)
	}
	if maxResults > 0 {
		q.Set("maxResults", strconv.Itoa(maxResults))
	}
	var out struct {
		Values []Sprint `json:"values"`
	}
	path := "/rest/agile/1.0/board/" + strconv.Itoa(boardID) + "/sprint"
	if err := j.rc.do(ctx, http.MethodGet, path, q, nil, &out); err != nil {
		return nil, err
	}
	return out.Values, nil
}

// GetBoardIssues lists issues on a board, optionally narrowed by JQL.
func (j *JiraClient) GetBoardIssues(ctx context.Context, boardID int, jql string, maxResults int) ([]Issue, error) {
	path := "/rest/agile/1.0/board/" + strconv.Itoa(boardID) + "/issue"
	return j.agileIssues(ctx, path, jql, maxResults)
}

// GetSprintIssues lists the issues in a sprint, optionally narrowed by JQL.
func (j *JiraClient) GetSprintIssues(ctx context.Context, sprintID int, jql string, maxResults int) ([]Issue, error) {
	path := "/rest/agile/1.0/sprint/" + strconv.Itoa(sprintID) + "/issue"
	return j.agileIssues(ctx, path, jql, maxResults)
}

func (j *JiraClient) agileIssues(ctx context.Context, path, jql string, maxResults int) ([]Issue, error) {
	q := url.Values{}
	if jql != "" {
		q.Set("jql", jql)
	}
	if maxResults > 0 {
		q.Set("maxResults", strconv.Itoa(maxResults))
	}
	var out struct {
		Issues []Issue `json:"issues"`
	}
	if err := j.rc.do(ctx, http.MethodGet, path, q, nil, &out); err != nil {
		return nil, err
	}
	return out.Issues, nil
}

// CreateSprint creates a sprint from a pre-built body (name, originBoardId,
// optional startDate/endDate/goal).
func (j *JiraClient) CreateSprint(ctx context.Context, body map[string]any) (*Sprint, error) {
	var out Sprint
	if err := j.rc.do(ctx, http.MethodPost, "/rest/agile/1.0/sprint", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateSprint partially updates a sprint (POST applies a partial change:
// name, goal, state, dates).
func (j *JiraClient) UpdateSprint(ctx context.Context, sprintID int, body map[string]any) (*Sprint, error) {
	var out Sprint
	path := "/rest/agile/1.0/sprint/" + strconv.Itoa(sprintID)
	if err := j.rc.do(ctx, http.MethodPost, path, nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// MoveIssuesToSprint moves issues into a sprint by key.
func (j *JiraClient) MoveIssuesToSprint(ctx context.Context, sprintID int, keys []string) error {
	path := "/rest/agile/1.0/sprint/" + strconv.Itoa(sprintID) + "/issue"
	return j.rc.do(ctx, http.MethodPost, path, nil, map[string]any{"issues": keys}, nil)
}
