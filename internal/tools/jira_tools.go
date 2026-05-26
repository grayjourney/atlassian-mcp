package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/grayjourney/atlassian-mcp/internal/atlassian"
	"github.com/grayjourney/atlassian-mcp/internal/content"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const defaultJiraFields = "summary,status,assignee,priority,issuetype,updated"

// --- jira_search ---

type jiraSearchInput struct {
	JQL           string `json:"jql" jsonschema:"JQL query, e.g. \"project = PROJ AND status = 'In Progress'\" or \"assignee = currentUser()\""`
	Fields        string `json:"fields,omitempty" jsonschema:"comma-separated fields to return; '*all' for everything"`
	Limit         int    `json:"limit,omitempty" jsonschema:"max results (1-50), default 10"`
	NextPageToken string `json:"next_page_token,omitempty" jsonschema:"pagination token from a previous search"`
}

func (s *Server) jiraSearch(ctx context.Context, _ *mcp.CallToolRequest, in jiraSearchInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	limit := in.Limit
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	fieldsStr := in.Fields
	if fieldsStr == "" {
		fieldsStr = defaultJiraFields
	}
	var fields []string
	if fieldsStr != "*all" {
		fields = splitCSV(fieldsStr)
	}

	res, err := jc.Search(ctx, atlassian.SearchRequest{
		JQL: in.JQL, Fields: fields, MaxResults: limit, NextPageToken: in.NextPageToken,
	})
	if err != nil {
		return nil, nil, err
	}
	issues := make([]map[string]any, 0, len(res.Issues))
	for _, iss := range res.Issues {
		issues = append(issues, flattenIssue(iss, false))
	}
	return textResult(map[string]any{
		"issues":          issues,
		"count":           len(issues),
		"next_page_token": res.NextPageToken,
	})
}

// --- jira_get_issue ---

type jiraGetIssueInput struct {
	IssueKey string `json:"issue_key" jsonschema:"the issue key, e.g. PROJ-123"`
	Fields   string `json:"fields,omitempty" jsonschema:"comma-separated fields to return; '*all' for everything"`
	Expand   string `json:"expand,omitempty" jsonschema:"comma-separated expansions, e.g. renderedFields,changelog"`
}

func (s *Server) jiraGetIssue(ctx context.Context, _ *mcp.CallToolRequest, in jiraGetIssueInput) (*mcp.CallToolResult, any, error) {
	jc, cfg, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	var fields []string
	if in.Fields != "" && in.Fields != "*all" {
		fields = splitCSV(in.Fields)
	}
	iss, err := jc.GetIssue(ctx, in.IssueKey, fields, in.Expand)
	if err != nil {
		return nil, nil, err
	}
	out := flattenIssue(*iss, true)
	out["url"] = browseURL(cfg.JiraURL, iss.Key)
	return textResult(out)
}

// --- jira_create_issue ---

type jiraCreateIssueInput struct {
	ProjectKey  string `json:"project_key" jsonschema:"project key prefix, e.g. PROJ. Ask the user if unknown"`
	Summary     string `json:"summary" jsonschema:"issue title"`
	IssueType   string `json:"issue_type" jsonschema:"issue type name, e.g. Task, Bug, Story, Epic"`
	Description string `json:"description,omitempty" jsonschema:"issue description in Markdown"`
	AssigneeID  string `json:"assignee_account_id,omitempty" jsonschema:"assignee Atlassian account ID (not email) for Cloud"`
}

func (s *Server) jiraCreateIssue(ctx context.Context, _ *mcp.CallToolRequest, in jiraCreateIssueInput) (*mcp.CallToolResult, any, error) {
	jc, cfg, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	fields := map[string]any{
		"project":   map[string]any{"key": in.ProjectKey},
		"summary":   in.Summary,
		"issuetype": map[string]any{"name": in.IssueType},
	}
	if in.Description != "" {
		fields["description"] = content.MarkdownToADF(in.Description)
	}
	if in.AssigneeID != "" {
		fields["assignee"] = map[string]any{"accountId": in.AssigneeID}
	}
	created, err := jc.CreateIssue(ctx, fields)
	if err != nil {
		return nil, nil, err
	}
	return textResult(map[string]any{
		"key": created.Key,
		"id":  created.ID,
		"url": browseURL(cfg.JiraURL, created.Key),
	})
}

// --- jira_update_issue ---

type jiraUpdateIssueInput struct {
	IssueKey         string `json:"issue_key" jsonschema:"the issue key, e.g. PROJ-123"`
	Summary          string `json:"summary,omitempty" jsonschema:"new summary/title"`
	Description      string `json:"description,omitempty" jsonschema:"new description in Markdown"`
	AdditionalFields string `json:"additional_fields,omitempty" jsonschema:"raw JSON object of extra Jira fields to set, e.g. {\"labels\":[\"x\"]}"`
}

func (s *Server) jiraUpdateIssue(ctx context.Context, _ *mcp.CallToolRequest, in jiraUpdateIssueInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	fields := map[string]any{}
	if in.Summary != "" {
		fields["summary"] = in.Summary
	}
	if in.Description != "" {
		fields["description"] = content.MarkdownToADF(in.Description)
	}
	if in.AdditionalFields != "" {
		extra := map[string]any{}
		if err := json.Unmarshal([]byte(in.AdditionalFields), &extra); err != nil {
			return nil, nil, fmt.Errorf("additional_fields must be a JSON object: %w", err)
		}
		maps.Copy(fields, extra)
	}
	if len(fields) == 0 {
		return nil, nil, fmt.Errorf("nothing to update: provide summary, description or additional_fields")
	}
	if err := jc.UpdateIssue(ctx, in.IssueKey, fields); err != nil {
		return nil, nil, err
	}
	return textResult(map[string]any{"key": in.IssueKey, "updated": true})
}

// --- jira_transition_issue ---

type jiraTransitionInput struct {
	IssueKey   string `json:"issue_key" jsonschema:"the issue key, e.g. PROJ-123"`
	Transition string `json:"transition" jsonschema:"target status name (e.g. 'In Progress', 'Done') or transition id"`
	Comment    string `json:"comment,omitempty" jsonschema:"optional comment to add during the transition (Markdown)"`
}

func (s *Server) jiraTransitionIssue(ctx context.Context, _ *mcp.CallToolRequest, in jiraTransitionInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	trs, err := jc.GetTransitions(ctx, in.IssueKey)
	if err != nil {
		return nil, nil, err
	}
	tr, ok := resolveTransition(trs, in.Transition)
	if !ok {
		return nil, nil, fmt.Errorf("transition %q not available for %s; options: %s",
			in.Transition, in.IssueKey, availableTransitions(trs))
	}
	var comment map[string]any
	if in.Comment != "" {
		comment = content.MarkdownToADF(in.Comment)
	}
	if err := jc.TransitionIssue(ctx, in.IssueKey, tr.ID, nil, comment); err != nil {
		return nil, nil, err
	}
	return textResult(map[string]any{
		"key":             in.IssueKey,
		"transitioned_to": tr.Name,
		"transition_id":   tr.ID,
	})
}

func availableTransitions(trs []atlassian.Transition) string {
	names := make([]string, 0, len(trs))
	for _, tr := range trs {
		names = append(names, fmt.Sprintf("%q (id %s)", tr.Name, tr.ID))
	}
	return strings.Join(names, ", ")
}

func browseURL(base, key string) string {
	return strings.TrimRight(base, "/") + "/browse/" + key
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
