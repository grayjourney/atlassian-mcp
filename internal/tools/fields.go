package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grayjourney/atlassian-mcp/internal/atlassian"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// fieldResolver maps human field names to Jira field definitions so the tools
// can let users refer to fields (including custom ones) by name. Built once per
// call from atlassian.JiraClient.GetFields.
type fieldResolver struct {
	byID   map[string]atlassian.Field
	byName map[string]atlassian.Field // lowercased name -> field
	all    []atlassian.Field
}

func newFieldResolver(fields []atlassian.Field) *fieldResolver {
	r := &fieldResolver{
		byID:   make(map[string]atlassian.Field, len(fields)),
		byName: make(map[string]atlassian.Field, len(fields)),
		all:    fields,
	}
	for _, f := range fields {
		r.byID[f.ID] = f
		r.byName[strings.ToLower(f.Name)] = f
	}
	return r
}

// resolve finds a field by exact ID or case-insensitive name.
func (r *fieldResolver) resolve(nameOrID string) (atlassian.Field, bool) {
	if f, ok := r.byID[nameOrID]; ok {
		return f, true
	}
	if f, ok := r.byName[strings.ToLower(strings.TrimSpace(nameOrID))]; ok {
		return f, true
	}
	return atlassian.Field{}, false
}

// storyPointsField locates the Story Points field, which has a different custom
// id on every Jira instance.
func (r *fieldResolver) storyPointsField() (atlassian.Field, bool) {
	for _, f := range r.all {
		if strings.Contains(f.Schema.Custom, "gh-story-points") {
			return f, true
		}
		n := strings.ToLower(f.Name)
		if n == "story points" || n == "story point estimate" {
			return f, true
		}
	}
	return atlassian.Field{}, false
}

// sprintField locates the Sprint custom field.
func (r *fieldResolver) sprintField() (atlassian.Field, bool) {
	for _, f := range r.all {
		if strings.Contains(f.Schema.Custom, "gh-sprint") || strings.ToLower(f.Name) == "sprint" {
			return f, true
		}
	}
	return atlassian.Field{}, false
}

// formatFieldValue shapes a raw value into the structure the Jira API expects
// for the given field, based on its schema type. Generic fields-by-name and
// story points flow through here; typed conveniences (labels, components, ...)
// are shaped directly by the caller.
func formatFieldValue(f atlassian.Field, value any) any {
	switch f.Schema.Type {
	case "option":
		return map[string]any{"value": toString(value)}
	case "user":
		return map[string]any{"accountId": toString(value)}
	case "array":
		return formatArray(f.Schema.Items, value)
	default:
		// number, date, datetime, string, and anything else: pass through.
		return value
	}
}

func formatArray(items string, value any) any {
	elems := toSlice(value)
	switch items {
	case "option":
		out := make([]any, 0, len(elems))
		for _, e := range elems {
			out = append(out, map[string]any{"value": toString(e)})
		}
		return out
	case "user":
		out := make([]any, 0, len(elems))
		for _, e := range elems {
			out = append(out, map[string]any{"accountId": toString(e)})
		}
		return out
	default:
		return value // array of strings (labels, etc.) passes through
	}
}

// nameObjects wraps each string as {"name": s}, the shape Jira wants for
// components and fix versions.
func nameObjects(names []string) []map[string]any {
	out := make([]map[string]any, 0, len(names))
	for _, n := range names {
		out = append(out, map[string]any{"name": n})
	}
	return out
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func toSlice(v any) []any {
	switch s := v.(type) {
	case []any:
		return s
	case []string:
		out := make([]any, len(s))
		for i, e := range s {
			out[i] = e
		}
		return out
	default:
		return nil
	}
}

// jiraFields fetches the instance's field definitions and wraps them in a
// resolver. (No caching yet — adding a per-URL cache is a tracked optimization.)
func (s *Server) jiraFields(ctx context.Context, jc *atlassian.JiraClient) (*fieldResolver, error) {
	fields, err := jc.GetFields(ctx)
	if err != nil {
		return nil, err
	}
	return newFieldResolver(fields), nil
}

// resolveAccountID turns an assignee reference (email, display name, or account
// id) into an account id. It searches users and prefers an exact email match,
// falling back to the first result, then to the raw input (assume it's an id).
func resolveAccountID(ctx context.Context, jc *atlassian.JiraClient, assignee string) (string, error) {
	users, err := jc.SearchUsers(ctx, assignee, 5)
	if err != nil {
		return "", err
	}
	for _, u := range users {
		if strings.EqualFold(u.Email, assignee) {
			return u.AccountID, nil
		}
	}
	if len(users) > 0 {
		return users[0].AccountID, nil
	}
	return assignee, nil
}

// issueFieldArgs are the optional, shared issue fields accepted by both create
// and update. Empty/nil members are skipped.
type issueFieldArgs struct {
	DueDate     string
	StoryPoints *float64
	Priority    string
	Assignee    string
	Labels      []string
	Components  []string
	FixVersions []string
	Parent      string
	FieldsJSON  string
}

// applyIssueFields shapes the optional args into the Jira fields map. It only
// calls GetFields when story points or by-name fields are involved.
func (s *Server) applyIssueFields(ctx context.Context, jc *atlassian.JiraClient, fields map[string]any, a issueFieldArgs) error {
	if a.DueDate != "" {
		fields["duedate"] = a.DueDate
	}
	if a.Priority != "" {
		fields["priority"] = map[string]any{"name": a.Priority}
	}
	if len(a.Labels) > 0 {
		fields["labels"] = a.Labels
	}
	if len(a.Components) > 0 {
		fields["components"] = nameObjects(a.Components)
	}
	if len(a.FixVersions) > 0 {
		fields["fixVersions"] = nameObjects(a.FixVersions)
	}
	if a.Parent != "" {
		fields["parent"] = map[string]any{"key": a.Parent}
	}
	if a.Assignee != "" {
		id, err := resolveAccountID(ctx, jc, a.Assignee)
		if err != nil {
			return err
		}
		fields["assignee"] = map[string]any{"accountId": id}
	}
	if a.StoryPoints == nil && a.FieldsJSON == "" {
		return nil
	}

	r, err := s.jiraFields(ctx, jc)
	if err != nil {
		return err
	}
	if a.StoryPoints != nil {
		f, ok := r.storyPointsField()
		if !ok {
			return fmt.Errorf("could not find a Story Points field on this Jira instance")
		}
		fields[f.ID] = *a.StoryPoints
	}
	if a.FieldsJSON != "" {
		extra := map[string]any{}
		if err := json.Unmarshal([]byte(a.FieldsJSON), &extra); err != nil {
			return fmt.Errorf("fields must be a JSON object of name->value: %w", err)
		}
		for name, val := range extra {
			f, ok := r.resolve(name)
			if !ok {
				return fmt.Errorf("unknown field %q (run jira_list_fields to see available names)", name)
			}
			fields[f.ID] = formatFieldValue(f, val)
		}
	}
	return nil
}

// --- jira_list_fields ---

type jiraListFieldsInput struct {
	Query      string `json:"query,omitempty" jsonschema:"filter fields whose name contains this text (case-insensitive)"`
	CustomOnly bool   `json:"custom_only,omitempty" jsonschema:"only return custom fields"`
}

func (s *Server) jiraListFields(ctx context.Context, _ *mcp.CallToolRequest, in jiraListFieldsInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	r, err := s.jiraFields(ctx, jc)
	if err != nil {
		return nil, nil, err
	}
	q := strings.ToLower(in.Query)
	out := make([]map[string]any, 0, len(r.all))
	for _, f := range r.all {
		if in.CustomOnly && !f.Custom {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(f.Name), q) {
			continue
		}
		out = append(out, map[string]any{
			"id":     f.ID,
			"name":   f.Name,
			"custom": f.Custom,
			"type":   f.Schema.Type,
		})
	}
	return textResult(map[string]any{"fields": out, "count": len(out)})
}

// --- jira_get_field_options ---

type jiraGetFieldOptionsInput struct {
	Field string `json:"field" jsonschema:"select/multi-select field name or id"`
}

func (s *Server) jiraGetFieldOptions(ctx context.Context, _ *mcp.CallToolRequest, in jiraGetFieldOptionsInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	r, err := s.jiraFields(ctx, jc)
	if err != nil {
		return nil, nil, err
	}
	f, ok := r.resolve(in.Field)
	if !ok {
		return nil, nil, fmt.Errorf("unknown field %q (run jira_list_fields to see names)", in.Field)
	}
	opts, err := jc.GetFieldOptions(ctx, f.ID)
	if err != nil {
		return nil, nil, err
	}
	values := make([]string, 0, len(opts))
	for _, o := range opts {
		values = append(values, o.Value)
	}
	return textResult(map[string]any{"field": f.Name, "id": f.ID, "options": values})
}

// --- jira_get_changelog ---

type jiraGetChangelogInput struct {
	IssueKey string `json:"issue_key" jsonschema:"the issue key, e.g. PROJ-123"`
	Limit    int    `json:"limit,omitempty" jsonschema:"max history entries (1-100), default 50"`
}

func (s *Server) jiraGetChangelog(ctx context.Context, _ *mcp.CallToolRequest, in jiraGetChangelogInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	limit := in.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	page, err := jc.GetChangelog(ctx, in.IssueKey, 0, limit)
	if err != nil {
		return nil, nil, err
	}
	entries := make([]map[string]any, 0, len(page.Values))
	for _, e := range page.Values {
		changes := make([]map[string]any, 0, len(e.Items))
		for _, it := range e.Items {
			changes = append(changes, map[string]any{
				"field": it.Field, "from": it.FromString, "to": it.ToString,
			})
		}
		entries = append(entries, map[string]any{
			"author": e.Author.DisplayName, "created": e.Created, "changes": changes,
		})
	}
	return textResult(map[string]any{
		"key": in.IssueKey, "history": entries, "count": len(entries), "total": page.Total,
	})
}

// --- jira_delete_issue ---

type jiraDeleteIssueInput struct {
	IssueKey       string `json:"issue_key" jsonschema:"the issue key, e.g. PROJ-123"`
	DeleteSubtasks bool   `json:"delete_subtasks,omitempty" jsonschema:"also delete subtasks (required if the issue has any)"`
}

func (s *Server) jiraDeleteIssue(ctx context.Context, _ *mcp.CallToolRequest, in jiraDeleteIssueInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	if err := jc.DeleteIssue(ctx, in.IssueKey, in.DeleteSubtasks); err != nil {
		return nil, nil, err
	}
	return textResult(map[string]any{"key": in.IssueKey, "deleted": true})
}
