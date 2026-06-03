package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- jira_list_service_desks ---

func (s *Server) jiraListServiceDesks(ctx context.Context, _ *mcp.CallToolRequest, _ jiraNoArgsInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	ds, err := jc.GetServiceDesks(ctx)
	if err != nil {
		return nil, nil, err
	}
	out := make([]map[string]any, 0, len(ds))
	for _, d := range ds {
		out = append(out, map[string]any{
			"id": d.ID, "project_key": d.ProjectKey, "project_name": d.ProjectName,
		})
	}
	return textResult(map[string]any{"service_desks": out, "count": len(out)})
}

// --- jira_list_queues ---

type jiraListQueuesInput struct {
	ServiceDeskID string `json:"service_desk_id" jsonschema:"the service desk id (see jira_list_service_desks)"`
}

func (s *Server) jiraListQueues(ctx context.Context, _ *mcp.CallToolRequest, in jiraListQueuesInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	qs, err := jc.GetQueues(ctx, in.ServiceDeskID)
	if err != nil {
		return nil, nil, err
	}
	out := make([]map[string]any, 0, len(qs))
	for _, q := range qs {
		out = append(out, map[string]any{"id": q.ID, "name": q.Name, "issue_count": q.IssueCount})
	}
	return textResult(map[string]any{"service_desk_id": in.ServiceDeskID, "queues": out, "count": len(out)})
}

// --- jira_get_queue_issues ---

type jiraQueueIssuesInput struct {
	ServiceDeskID string `json:"service_desk_id" jsonschema:"the service desk id"`
	QueueID       string `json:"queue_id" jsonschema:"the queue id (see jira_list_queues)"`
	Limit         int    `json:"limit,omitempty" jsonschema:"max issues (1-50), default 25"`
}

func (s *Server) jiraGetQueueIssues(ctx context.Context, _ *mcp.CallToolRequest, in jiraQueueIssuesInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	issues, err := jc.GetQueueIssues(ctx, in.ServiceDeskID, in.QueueID, clampLimit(in.Limit, 25, 50))
	if err != nil {
		return nil, nil, err
	}
	return textResult(map[string]any{
		"service_desk_id": in.ServiceDeskID, "queue_id": in.QueueID,
		"issues": issueList(issues), "count": len(issues),
	})
}

// --- jira_get_development_info ---

type jiraDevInfoInput struct {
	IssueKey string `json:"issue_key" jsonschema:"the issue key, e.g. PROJ-123"`
}

func (s *Server) jiraGetDevelopmentInfo(ctx context.Context, _ *mcp.CallToolRequest, in jiraDevInfoInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	// The dev-status API keys off the numeric issue id, not the key.
	iss, err := jc.GetIssue(ctx, in.IssueKey, []string{"summary"}, "")
	if err != nil {
		return nil, nil, err
	}
	summary, err := jc.GetDevelopmentSummary(ctx, iss.ID)
	if err != nil {
		return nil, nil, err
	}
	return textResult(map[string]any{"key": in.IssueKey, "development": summary})
}
