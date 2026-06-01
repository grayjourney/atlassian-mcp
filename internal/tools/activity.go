package tools

import (
	"context"

	"github.com/grayjourney/atlassian-mcp/internal/content"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- jira_add_comment ---

type jiraAddCommentInput struct {
	IssueKey string `json:"issue_key" jsonschema:"the issue key, e.g. PROJ-123"`
	Comment  string `json:"comment" jsonschema:"comment body in Markdown"`
}

func (s *Server) jiraAddComment(ctx context.Context, _ *mcp.CallToolRequest, in jiraAddCommentInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	c, err := jc.AddComment(ctx, in.IssueKey, content.MarkdownToADF(in.Comment))
	if err != nil {
		return nil, nil, err
	}
	return textResult(map[string]any{"key": in.IssueKey, "comment_id": c.ID})
}

// --- jira_list_comments ---

type jiraListCommentsInput struct {
	IssueKey string `json:"issue_key" jsonschema:"the issue key, e.g. PROJ-123"`
	Limit    int    `json:"limit,omitempty" jsonschema:"max comments (1-100), default 50"`
}

func (s *Server) jiraListComments(ctx context.Context, _ *mcp.CallToolRequest, in jiraListCommentsInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	limit := in.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	cs, err := jc.GetComments(ctx, in.IssueKey, limit)
	if err != nil {
		return nil, nil, err
	}
	out := make([]map[string]any, 0, len(cs))
	for _, c := range cs {
		out = append(out, map[string]any{
			"id": c.ID, "author": c.Author.DisplayName,
			"created": c.Created, "updated": c.Updated,
			"body": content.ADFToText(c.Body),
		})
	}
	return textResult(map[string]any{"key": in.IssueKey, "comments": out, "count": len(out)})
}

// --- jira_edit_comment ---

type jiraEditCommentInput struct {
	IssueKey  string `json:"issue_key" jsonschema:"the issue key, e.g. PROJ-123"`
	CommentID string `json:"comment_id" jsonschema:"id of the comment to edit"`
	Comment   string `json:"comment" jsonschema:"new comment body in Markdown"`
}

func (s *Server) jiraEditComment(ctx context.Context, _ *mcp.CallToolRequest, in jiraEditCommentInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	if _, err := jc.EditComment(ctx, in.IssueKey, in.CommentID, content.MarkdownToADF(in.Comment)); err != nil {
		return nil, nil, err
	}
	return textResult(map[string]any{"key": in.IssueKey, "comment_id": in.CommentID, "updated": true})
}

// --- jira_add_worklog ---

type jiraAddWorklogInput struct {
	IssueKey  string `json:"issue_key" jsonschema:"the issue key, e.g. PROJ-123"`
	TimeSpent string `json:"time_spent" jsonschema:"time spent, Jira format e.g. \"2h 30m\", \"1d\""`
	Comment   string `json:"comment,omitempty" jsonschema:"optional worklog comment in Markdown"`
	Started   string `json:"started,omitempty" jsonschema:"optional start time, ISO8601 e.g. 2026-06-01T09:00:00.000+0000"`
}

func (s *Server) jiraAddWorklog(ctx context.Context, _ *mcp.CallToolRequest, in jiraAddWorklogInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	body := map[string]any{"timeSpent": in.TimeSpent}
	if in.Comment != "" {
		body["comment"] = content.MarkdownToADF(in.Comment)
	}
	if in.Started != "" {
		body["started"] = in.Started
	}
	w, err := jc.AddWorklog(ctx, in.IssueKey, body)
	if err != nil {
		return nil, nil, err
	}
	return textResult(map[string]any{"key": in.IssueKey, "worklog_id": w.ID, "time_spent": w.TimeSpent})
}

// --- jira_get_worklog ---

type jiraGetWorklogInput struct {
	IssueKey string `json:"issue_key" jsonschema:"the issue key, e.g. PROJ-123"`
}

func (s *Server) jiraGetWorklog(ctx context.Context, _ *mcp.CallToolRequest, in jiraGetWorklogInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	ws, err := jc.GetWorklogs(ctx, in.IssueKey)
	if err != nil {
		return nil, nil, err
	}
	out := make([]map[string]any, 0, len(ws))
	total := 0
	for _, w := range ws {
		total += w.TimeSpentSeconds
		out = append(out, map[string]any{
			"id": w.ID, "author": w.Author.DisplayName,
			"time_spent": w.TimeSpent, "started": w.Started,
			"comment": content.ADFToText(w.Comment),
		})
	}
	return textResult(map[string]any{
		"key": in.IssueKey, "worklogs": out, "count": len(out),
		"total_seconds": total,
	})
}

// --- jira_get_issue_dates ---

type jiraGetIssueDatesInput struct {
	IssueKey string `json:"issue_key" jsonschema:"the issue key, e.g. PROJ-123"`
}

func (s *Server) jiraGetIssueDates(ctx context.Context, _ *mcp.CallToolRequest, in jiraGetIssueDatesInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	r, err := s.jiraFields(ctx, jc)
	if err != nil {
		return nil, nil, err
	}
	names := map[string]string{
		"created": "Created", "updated": "Updated",
		"duedate": "Due date", "resolutiondate": "Resolved",
	}
	ids := []string{"created", "updated", "duedate", "resolutiondate"}
	for _, f := range r.all {
		if (f.Schema.Type == "date" || f.Schema.Type == "datetime") && names[f.ID] == "" {
			names[f.ID] = f.Name
			ids = append(ids, f.ID)
		}
	}
	iss, err := jc.GetIssue(ctx, in.IssueKey, ids, "")
	if err != nil {
		return nil, nil, err
	}
	dates := map[string]any{}
	for id, nm := range names {
		if v, ok := iss.Fields[id]; ok && v != nil && v != "" {
			dates[nm] = v
		}
	}
	return textResult(map[string]any{"key": in.IssueKey, "dates": dates})
}

// --- jira_list_watchers ---

type jiraListWatchersInput struct {
	IssueKey string `json:"issue_key" jsonschema:"the issue key, e.g. PROJ-123"`
}

func (s *Server) jiraListWatchers(ctx context.Context, _ *mcp.CallToolRequest, in jiraListWatchersInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	w, err := jc.GetWatchers(ctx, in.IssueKey)
	if err != nil {
		return nil, nil, err
	}
	names := make([]string, 0, len(w.Watchers))
	for _, u := range w.Watchers {
		names = append(names, u.DisplayName)
	}
	return textResult(map[string]any{
		"key": in.IssueKey, "count": w.WatchCount,
		"is_watching": w.IsWatching, "watchers": names,
	})
}

// --- jira_add_watcher / jira_remove_watcher ---

type jiraWatcherInput struct {
	IssueKey string `json:"issue_key" jsonschema:"the issue key, e.g. PROJ-123"`
	User     string `json:"user" jsonschema:"watcher's email, display name, or account id"`
}

func (s *Server) jiraAddWatcher(ctx context.Context, _ *mcp.CallToolRequest, in jiraWatcherInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	id, err := resolveAccountID(ctx, jc, in.User)
	if err != nil {
		return nil, nil, err
	}
	if err := jc.AddWatcher(ctx, in.IssueKey, id); err != nil {
		return nil, nil, err
	}
	return textResult(map[string]any{"key": in.IssueKey, "account_id": id, "watching": true})
}

func (s *Server) jiraRemoveWatcher(ctx context.Context, _ *mcp.CallToolRequest, in jiraWatcherInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	id, err := resolveAccountID(ctx, jc, in.User)
	if err != nil {
		return nil, nil, err
	}
	if err := jc.RemoveWatcher(ctx, in.IssueKey, id); err != nil {
		return nil, nil, err
	}
	return textResult(map[string]any{"key": in.IssueKey, "account_id": id, "watching": false})
}

// --- jira_get_user ---

type jiraGetUserInput struct {
	User string `json:"user" jsonschema:"email, display name, or account id to look up"`
}

func (s *Server) jiraGetUser(ctx context.Context, _ *mcp.CallToolRequest, in jiraGetUserInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	users, err := jc.SearchUsers(ctx, in.User, 10)
	if err != nil {
		return nil, nil, err
	}
	out := make([]map[string]any, 0, len(users))
	for _, u := range users {
		out = append(out, map[string]any{
			"account_id": u.AccountID, "display_name": u.DisplayName,
			"email": u.Email, "active": u.Active,
		})
	}
	return textResult(map[string]any{"query": in.User, "users": out, "count": len(out)})
}
