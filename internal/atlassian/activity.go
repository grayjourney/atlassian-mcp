package atlassian

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// IssueComment is a Jira issue comment. Body is raw ADF for the tool layer to render.
type IssueComment struct {
	ID     string `json:"id"`
	Author struct {
		DisplayName string `json:"displayName"`
	} `json:"author"`
	Body    any    `json:"body"`
	Created string `json:"created"`
	Updated string `json:"updated"`
}

// GetComments lists an issue's comments (most recent page).
func (j *JiraClient) GetComments(ctx context.Context, key string, maxResults int) ([]IssueComment, error) {
	q := url.Values{}
	if maxResults > 0 {
		q.Set("maxResults", strconv.Itoa(maxResults))
	}
	var out struct {
		Comments []IssueComment `json:"comments"`
		Total    int            `json:"total"`
	}
	if err := j.rc.do(ctx, http.MethodGet, "/rest/api/3/issue/"+key+"/comment", q, nil, &out); err != nil {
		return nil, err
	}
	return out.Comments, nil
}

// AddComment posts an ADF comment to an issue.
func (j *JiraClient) AddComment(ctx context.Context, key string, bodyADF map[string]any) (*IssueComment, error) {
	var out IssueComment
	body := map[string]any{"body": bodyADF}
	if err := j.rc.do(ctx, http.MethodPost, "/rest/api/3/issue/"+key+"/comment", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// EditComment replaces the body of an existing comment.
func (j *JiraClient) EditComment(ctx context.Context, key, commentID string, bodyADF map[string]any) (*IssueComment, error) {
	var out IssueComment
	body := map[string]any{"body": bodyADF}
	path := "/rest/api/3/issue/" + key + "/comment/" + commentID
	if err := j.rc.do(ctx, http.MethodPut, path, nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Worklog is one time-tracking entry on an issue.
type Worklog struct {
	ID     string `json:"id"`
	Author struct {
		DisplayName string `json:"displayName"`
	} `json:"author"`
	TimeSpent        string `json:"timeSpent"`
	TimeSpentSeconds int    `json:"timeSpentSeconds"`
	Comment          any    `json:"comment"`
	Started          string `json:"started"`
	Created          string `json:"created"`
}

// GetWorklogs lists an issue's worklog entries.
func (j *JiraClient) GetWorklogs(ctx context.Context, key string) ([]Worklog, error) {
	var out struct {
		Worklogs []Worklog `json:"worklogs"`
		Total    int       `json:"total"`
	}
	if err := j.rc.do(ctx, http.MethodGet, "/rest/api/3/issue/"+key+"/worklog", nil, nil, &out); err != nil {
		return nil, err
	}
	return out.Worklogs, nil
}

// AddWorklog logs time against an issue from a pre-built body (timeSpent,
// optional ADF comment, optional started timestamp).
func (j *JiraClient) AddWorklog(ctx context.Context, key string, body map[string]any) (*Worklog, error) {
	var out Worklog
	if err := j.rc.do(ctx, http.MethodPost, "/rest/api/3/issue/"+key+"/worklog", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Watchers is the watcher summary of an issue.
type Watchers struct {
	WatchCount int    `json:"watchCount"`
	IsWatching bool   `json:"isWatching"`
	Watchers   []User `json:"watchers"`
}

// GetWatchers returns an issue's watchers.
func (j *JiraClient) GetWatchers(ctx context.Context, key string) (*Watchers, error) {
	var out Watchers
	if err := j.rc.do(ctx, http.MethodGet, "/rest/api/3/issue/"+key+"/watchers", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AddWatcher adds a watcher by account id. The Jira API expects the bare account
// id as the JSON request body (a quoted string).
func (j *JiraClient) AddWatcher(ctx context.Context, key, accountID string) error {
	return j.rc.do(ctx, http.MethodPost, "/rest/api/3/issue/"+key+"/watchers", nil, accountID, nil)
}

// RemoveWatcher removes a watcher by account id.
func (j *JiraClient) RemoveWatcher(ctx context.Context, key, accountID string) error {
	q := url.Values{}
	q.Set("accountId", accountID)
	return j.rc.do(ctx, http.MethodDelete, "/rest/api/3/issue/"+key+"/watchers", q, nil, nil)
}

// GetUser fetches a single user by account id.
func (j *JiraClient) GetUser(ctx context.Context, accountID string) (*User, error) {
	q := url.Values{}
	q.Set("accountId", accountID)
	var out User
	if err := j.rc.do(ctx, http.MethodGet, "/rest/api/3/user", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
