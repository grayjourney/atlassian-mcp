package atlassian

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// ServiceDesk is a Jira Service Management service desk (one per JSM project).
type ServiceDesk struct {
	ID          string `json:"id"`
	ProjectID   string `json:"projectId"`
	ProjectKey  string `json:"projectKey"`
	ProjectName string `json:"projectName"`
}

// Queue is a request queue within a service desk.
type Queue struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	IssueCount int    `json:"issueCount"`
}

// GetServiceDesks lists the service desks (JSM projects) the user can see.
func (j *JiraClient) GetServiceDesks(ctx context.Context) ([]ServiceDesk, error) {
	var out struct {
		Values []ServiceDesk `json:"values"`
	}
	if err := j.rc.do(ctx, http.MethodGet, "/rest/servicedeskapi/servicedesk", nil, nil, &out); err != nil {
		return nil, err
	}
	return out.Values, nil
}

// GetQueues lists a service desk's queues.
func (j *JiraClient) GetQueues(ctx context.Context, serviceDeskID string) ([]Queue, error) {
	var out struct {
		Values []Queue `json:"values"`
	}
	path := "/rest/servicedeskapi/servicedesk/" + serviceDeskID + "/queue"
	if err := j.rc.do(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return out.Values, nil
}

// GetQueueIssues lists the issues currently in a queue.
func (j *JiraClient) GetQueueIssues(ctx context.Context, serviceDeskID, queueID string, maxResults int) ([]Issue, error) {
	q := url.Values{}
	if maxResults > 0 {
		q.Set("limit", strconv.Itoa(maxResults))
	}
	var out struct {
		Values []Issue `json:"values"`
	}
	path := "/rest/servicedeskapi/servicedesk/" + serviceDeskID + "/queue/" + queueID + "/issue"
	if err := j.rc.do(ctx, http.MethodGet, path, q, nil, &out); err != nil {
		return nil, err
	}
	return out.Values, nil
}

// GetDevelopmentSummary returns the development panel summary (counts of
// branches, commits, pull requests from connected VCS tools) for an issue by its
// numeric id, via the internal dev-status API.
func (j *JiraClient) GetDevelopmentSummary(ctx context.Context, issueID string) (map[string]any, error) {
	q := url.Values{}
	q.Set("issueId", issueID)
	var out struct {
		Summary map[string]any `json:"summary"`
	}
	if err := j.rc.do(ctx, http.MethodGet, "/rest/dev-status/latest/issue/summary", q, nil, &out); err != nil {
		return nil, err
	}
	return out.Summary, nil
}
