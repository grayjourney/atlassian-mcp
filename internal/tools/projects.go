package tools

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/grayjourney/atlassian-mcp/internal/atlassian"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- jira_list_projects ---

type jiraListProjectsInput struct {
	Query string `json:"query,omitempty" jsonschema:"filter by project key or name"`
	Limit int    `json:"limit,omitempty" jsonschema:"max projects (1-100), default 50"`
}

func (s *Server) jiraListProjects(ctx context.Context, _ *mcp.CallToolRequest, in jiraListProjectsInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	ps, err := jc.GetProjects(ctx, in.Query, clampLimit(in.Limit, 50, 100))
	if err != nil {
		return nil, nil, err
	}
	out := make([]map[string]any, 0, len(ps))
	for _, p := range ps {
		out = append(out, map[string]any{"id": p.ID, "key": p.Key, "name": p.Name, "type": p.ProjectTypeKey})
	}
	return textResult(map[string]any{"projects": out, "count": len(out)})
}

// --- jira_get_project_versions ---

type jiraProjectInput struct {
	ProjectKey string `json:"project_key" jsonschema:"the project key, e.g. KAN"`
}

func (s *Server) jiraGetProjectVersions(ctx context.Context, _ *mcp.CallToolRequest, in jiraProjectInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	vs, err := jc.GetProjectVersions(ctx, in.ProjectKey)
	if err != nil {
		return nil, nil, err
	}
	out := make([]map[string]any, 0, len(vs))
	for _, v := range vs {
		m := map[string]any{"id": v.ID, "name": v.Name, "released": v.Released, "archived": v.Archived}
		if v.ReleaseDate != "" {
			m["release_date"] = v.ReleaseDate
		}
		out = append(out, m)
	}
	return textResult(map[string]any{"project": in.ProjectKey, "versions": out, "count": len(out)})
}

// --- jira_get_project_components ---

func (s *Server) jiraGetProjectComponents(ctx context.Context, _ *mcp.CallToolRequest, in jiraProjectInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	cs, err := jc.GetProjectComponents(ctx, in.ProjectKey)
	if err != nil {
		return nil, nil, err
	}
	out := make([]map[string]any, 0, len(cs))
	for _, c := range cs {
		m := map[string]any{"id": c.ID, "name": c.Name}
		if c.Description != "" {
			m["description"] = c.Description
		}
		if c.Lead.DisplayName != "" {
			m["lead"] = c.Lead.DisplayName
		}
		out = append(out, m)
	}
	return textResult(map[string]any{"project": in.ProjectKey, "components": out, "count": len(out)})
}

// --- jira_create_version ---

type jiraCreateVersionInput struct {
	ProjectKey  string `json:"project_key" jsonschema:"the project key, e.g. KAN"`
	Name        string `json:"name" jsonschema:"version name, e.g. v2.0"`
	Description string `json:"description,omitempty" jsonschema:"version description"`
	ReleaseDate string `json:"release_date,omitempty" jsonschema:"release date as YYYY-MM-DD"`
	StartDate   string `json:"start_date,omitempty" jsonschema:"start date as YYYY-MM-DD"`
	Released    bool   `json:"released,omitempty" jsonschema:"mark the version as already released"`
}

func (s *Server) jiraCreateVersion(ctx context.Context, _ *mcp.CallToolRequest, in jiraCreateVersionInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	// The create-version API wants a numeric project id, so resolve the key.
	proj, err := jc.GetProject(ctx, in.ProjectKey)
	if err != nil {
		return nil, nil, err
	}
	pid, err := strconv.Atoi(proj.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("unexpected project id %q: %w", proj.ID, err)
	}
	body := map[string]any{"name": in.Name, "projectId": pid}
	if in.Description != "" {
		body["description"] = in.Description
	}
	if in.ReleaseDate != "" {
		body["releaseDate"] = in.ReleaseDate
	}
	if in.StartDate != "" {
		body["startDate"] = in.StartDate
	}
	if in.Released {
		body["released"] = true
	}
	v, err := jc.CreateVersion(ctx, body)
	if err != nil {
		return nil, nil, err
	}
	return textResult(map[string]any{"id": v.ID, "name": v.Name, "project": in.ProjectKey})
}

// --- jira_list_link_types ---

type jiraNoArgsInput struct{}

func (s *Server) jiraListLinkTypes(ctx context.Context, _ *mcp.CallToolRequest, _ jiraNoArgsInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	ts, err := jc.GetIssueLinkTypes(ctx)
	if err != nil {
		return nil, nil, err
	}
	out := make([]map[string]any, 0, len(ts))
	for _, lt := range ts {
		out = append(out, map[string]any{"name": lt.Name, "inward": lt.Inward, "outward": lt.Outward})
	}
	return textResult(map[string]any{"link_types": out, "count": len(out)})
}

// --- jira_create_issue_link ---

type jiraCreateIssueLinkInput struct {
	Type         string `json:"type" jsonschema:"link type name, e.g. Blocks, Relates (see jira_list_link_types)"`
	InwardIssue  string `json:"inward_issue" jsonschema:"the inward issue key (e.g. the blocked issue)"`
	OutwardIssue string `json:"outward_issue" jsonschema:"the outward issue key (e.g. the blocking issue)"`
}

func (s *Server) jiraCreateIssueLink(ctx context.Context, _ *mcp.CallToolRequest, in jiraCreateIssueLinkInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	if err := jc.CreateIssueLink(ctx, in.Type, in.InwardIssue, in.OutwardIssue); err != nil {
		return nil, nil, err
	}
	return textResult(map[string]any{
		"type": in.Type, "inward_issue": in.InwardIssue,
		"outward_issue": in.OutwardIssue, "linked": true,
	})
}

// --- jira_remove_issue_link ---

type jiraRemoveIssueLinkInput struct {
	LinkID string `json:"link_id" jsonschema:"the issue link id to remove"`
}

func (s *Server) jiraRemoveIssueLink(ctx context.Context, _ *mcp.CallToolRequest, in jiraRemoveIssueLinkInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	if err := jc.DeleteIssueLink(ctx, in.LinkID); err != nil {
		return nil, nil, err
	}
	return textResult(map[string]any{"link_id": in.LinkID, "removed": true})
}

// --- jira_link_to_epic ---

type jiraLinkToEpicInput struct {
	IssueKey string `json:"issue_key" jsonschema:"the issue to put under the epic"`
	EpicKey  string `json:"epic_key" jsonschema:"the epic's issue key"`
}

func (s *Server) jiraLinkToEpic(ctx context.Context, _ *mcp.CallToolRequest, in jiraLinkToEpicInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	// Team-managed projects use the standard "parent" field; company-managed
	// classic projects use an Epic Link custom field. Prefer the Epic Link field
	// when the instance has one, else fall back to parent.
	fields := map[string]any{}
	via := "parent"
	if r, err := s.jiraFields(ctx, jc); err == nil {
		if f, ok := epicLinkField(r); ok {
			fields[f.ID] = in.EpicKey
			via = f.Name
		}
	}
	if len(fields) == 0 {
		fields["parent"] = map[string]any{"key": in.EpicKey}
	}
	if err := jc.UpdateIssue(ctx, in.IssueKey, fields); err != nil {
		return nil, nil, err
	}
	return textResult(map[string]any{"issue": in.IssueKey, "epic": in.EpicKey, "via": via})
}

// epicLinkField finds the classic Epic Link custom field if present.
func epicLinkField(r *fieldResolver) (atlassian.Field, bool) {
	for _, f := range r.all {
		if strings.Contains(f.Schema.Custom, "gh-epic-link") || strings.EqualFold(f.Name, "epic link") {
			return f, true
		}
	}
	return atlassian.Field{}, false
}

// --- jira_create_remote_link ---

type jiraCreateRemoteLinkInput struct {
	IssueKey string `json:"issue_key" jsonschema:"the issue key, e.g. PROJ-123"`
	URL      string `json:"url" jsonschema:"the web URL to link"`
	Title    string `json:"title" jsonschema:"link title/label"`
}

func (s *Server) jiraCreateRemoteLink(ctx context.Context, _ *mcp.CallToolRequest, in jiraCreateRemoteLinkInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	if err := jc.CreateRemoteLink(ctx, in.IssueKey, in.URL, in.Title); err != nil {
		return nil, nil, err
	}
	return textResult(map[string]any{"key": in.IssueKey, "url": in.URL, "title": in.Title, "linked": true})
}
