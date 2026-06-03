package atlassian

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// Project is a Jira project.
type Project struct {
	ID             string `json:"id"`
	Key            string `json:"key"`
	Name           string `json:"name"`
	ProjectTypeKey string `json:"projectTypeKey"`
}

// Version is a project version (a "release" / milestone).
type Version struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Released    bool   `json:"released"`
	Archived    bool   `json:"archived"`
	ReleaseDate string `json:"releaseDate"`
	StartDate   string `json:"startDate"`
	ProjectID   int    `json:"projectId"`
}

// Component is a project component.
type Component struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Lead        struct {
		DisplayName string `json:"displayName"`
	} `json:"lead"`
}

// IssueLinkType is a kind of link between issues (e.g. Blocks).
type IssueLinkType struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Inward  string `json:"inward"`
	Outward string `json:"outward"`
}

// GetProjects searches projects (query matches key or name).
func (j *JiraClient) GetProjects(ctx context.Context, query string, maxResults int) ([]Project, error) {
	q := url.Values{}
	if query != "" {
		q.Set("query", query)
	}
	if maxResults > 0 {
		q.Set("maxResults", strconv.Itoa(maxResults))
	}
	var out struct {
		Values []Project `json:"values"`
	}
	if err := j.rc.do(ctx, http.MethodGet, "/rest/api/3/project/search", q, nil, &out); err != nil {
		return nil, err
	}
	return out.Values, nil
}

// GetProject fetches a single project by key or id.
func (j *JiraClient) GetProject(ctx context.Context, keyOrID string) (*Project, error) {
	var out Project
	if err := j.rc.do(ctx, http.MethodGet, "/rest/api/3/project/"+keyOrID, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetProjectVersions lists a project's versions (releases / milestones).
func (j *JiraClient) GetProjectVersions(ctx context.Context, keyOrID string) ([]Version, error) {
	var out []Version
	if err := j.rc.do(ctx, http.MethodGet, "/rest/api/3/project/"+keyOrID+"/versions", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateVersion creates a version from a pre-built body (name, projectId, ...).
func (j *JiraClient) CreateVersion(ctx context.Context, body map[string]any) (*Version, error) {
	var out Version
	if err := j.rc.do(ctx, http.MethodPost, "/rest/api/3/version", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetProjectComponents lists a project's components.
func (j *JiraClient) GetProjectComponents(ctx context.Context, keyOrID string) ([]Component, error) {
	var out []Component
	if err := j.rc.do(ctx, http.MethodGet, "/rest/api/3/project/"+keyOrID+"/components", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetIssueLinkTypes lists the available issue link types.
func (j *JiraClient) GetIssueLinkTypes(ctx context.Context) ([]IssueLinkType, error) {
	var out struct {
		IssueLinkTypes []IssueLinkType `json:"issueLinkTypes"`
	}
	if err := j.rc.do(ctx, http.MethodGet, "/rest/api/3/issueLinkType", nil, nil, &out); err != nil {
		return nil, err
	}
	return out.IssueLinkTypes, nil
}

// CreateIssueLink links two issues with the named link type.
func (j *JiraClient) CreateIssueLink(ctx context.Context, linkType, inwardKey, outwardKey string) error {
	body := map[string]any{
		"type":         map[string]any{"name": linkType},
		"inwardIssue":  map[string]any{"key": inwardKey},
		"outwardIssue": map[string]any{"key": outwardKey},
	}
	return j.rc.do(ctx, http.MethodPost, "/rest/api/3/issueLink", nil, body, nil)
}

// DeleteIssueLink removes an issue link by id.
func (j *JiraClient) DeleteIssueLink(ctx context.Context, linkID string) error {
	return j.rc.do(ctx, http.MethodDelete, "/rest/api/3/issueLink/"+linkID, nil, nil, nil)
}

// CreateRemoteLink attaches a web link (URL + title) to an issue.
func (j *JiraClient) CreateRemoteLink(ctx context.Context, key, link, title string) error {
	body := map[string]any{"object": map[string]any{"url": link, "title": title}}
	return j.rc.do(ctx, http.MethodPost, "/rest/api/3/issue/"+key+"/remotelink", nil, body, nil)
}
