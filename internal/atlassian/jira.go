package atlassian

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

// JiraClient wraps the Jira Cloud REST v3 API.
type JiraClient struct {
	rc *restClient
}

// NewJiraClient builds a Jira client for baseURL (e.g. https://acme.atlassian.net).
func NewJiraClient(baseURL, username, token string, hc *http.Client) *JiraClient {
	return &JiraClient{rc: newRESTClient(baseURL, username, token, hc)}
}

// Issue is a Jira issue with its raw field map left intact for the tool layer.
type Issue struct {
	ID     string         `json:"id"`
	Key    string         `json:"key"`
	Self   string         `json:"self"`
	Fields map[string]any `json:"fields"`
}

// SearchRequest holds parameters for a JQL search.
type SearchRequest struct {
	JQL           string
	Fields        []string
	MaxResults    int
	NextPageToken string
}

// SearchResult is the response of POST /rest/api/3/search/jql.
type SearchResult struct {
	Issues        []Issue `json:"issues"`
	NextPageToken string  `json:"nextPageToken"`
	IsLast        bool    `json:"isLast"`
}

// Search runs a JQL query using the current Cloud search endpoint.
func (j *JiraClient) Search(ctx context.Context, req SearchRequest) (*SearchResult, error) {
	body := map[string]any{"jql": req.JQL}
	if len(req.Fields) > 0 {
		body["fields"] = req.Fields
	}
	if req.MaxResults > 0 {
		body["maxResults"] = req.MaxResults
	}
	if req.NextPageToken != "" {
		body["nextPageToken"] = req.NextPageToken
	}
	var out SearchResult
	if err := j.rc.do(ctx, http.MethodPost, "/rest/api/3/search/jql", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetIssue fetches a single issue. fields restricts returned fields (empty = all)
// and expand controls expansions (e.g. "renderedFields").
func (j *JiraClient) GetIssue(ctx context.Context, key string, fields []string, expand string) (*Issue, error) {
	q := url.Values{}
	if len(fields) > 0 {
		q.Set("fields", strings.Join(fields, ","))
	}
	if expand != "" {
		q.Set("expand", expand)
	}
	var out Issue
	if err := j.rc.do(ctx, http.MethodGet, "/rest/api/3/issue/"+key, q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreatedIssue is the response of issue creation.
type CreatedIssue struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Self string `json:"self"`
}

// CreateIssue creates an issue from a pre-built fields map (project, summary,
// issuetype, description-as-ADF, ...). The caller owns field construction.
func (j *JiraClient) CreateIssue(ctx context.Context, fields map[string]any) (*CreatedIssue, error) {
	var out CreatedIssue
	body := map[string]any{"fields": fields}
	if err := j.rc.do(ctx, http.MethodPost, "/rest/api/3/issue", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateIssue applies a partial fields update (HTTP 204, no body).
func (j *JiraClient) UpdateIssue(ctx context.Context, key string, fields map[string]any) error {
	body := map[string]any{"fields": fields}
	return j.rc.do(ctx, http.MethodPut, "/rest/api/3/issue/"+key, nil, body, nil)
}

// Transition is an available workflow transition for an issue.
type Transition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	To   struct {
		Name string `json:"name"`
	} `json:"to"`
}

// GetTransitions lists the transitions currently available for an issue.
func (j *JiraClient) GetTransitions(ctx context.Context, key string) ([]Transition, error) {
	var out struct {
		Transitions []Transition `json:"transitions"`
	}
	if err := j.rc.do(ctx, http.MethodGet, "/rest/api/3/issue/"+key+"/transitions", nil, nil, &out); err != nil {
		return nil, err
	}
	return out.Transitions, nil
}

// TransitionIssue moves an issue through transitionID, optionally setting fields
// and adding an ADF comment (commentADF may be nil).
func (j *JiraClient) TransitionIssue(ctx context.Context, key, transitionID string, fields, commentADF map[string]any) error {
	body := map[string]any{"transition": map[string]any{"id": transitionID}}
	if len(fields) > 0 {
		body["fields"] = fields
	}
	if commentADF != nil {
		body["update"] = map[string]any{
			"comment": []any{map[string]any{"add": map[string]any{"body": commentADF}}},
		}
	}
	return j.rc.do(ctx, http.MethodPost, "/rest/api/3/issue/"+key+"/transitions", nil, body, nil)
}
