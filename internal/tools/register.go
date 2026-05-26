package tools

import "github.com/modelcontextprotocol/go-sdk/mcp"

// Register adds all MVP tools to the MCP server. Read tools carry a readOnly
// hint; writes are flagged non-destructive so clients can prompt for confirmation.
func (s *Server) Register(srv *mcp.Server) {
	readOnly := &mcp.ToolAnnotations{ReadOnlyHint: true}
	write := &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: new(false)}

	// Jira
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_search",
		Description: "Search Jira issues using JQL. Returns a compact list of matching issues.",
		Annotations: readOnly,
	}, s.jiraSearch)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_get_issue",
		Description: "Get the details of a single Jira issue by key (summary, status, assignee, description, ...).",
		Annotations: readOnly,
	}, s.jiraGetIssue)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_create_issue",
		Description: "Create a new Jira issue in a project. Description accepts Markdown.",
		Annotations: write,
	}, s.jiraCreateIssue)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_update_issue",
		Description: "Update fields of an existing Jira issue (summary, description, or arbitrary fields).",
		Annotations: write,
	}, s.jiraUpdateIssue)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_transition_issue",
		Description: "Move a Jira issue to a new status by transition name or id, optionally with a comment.",
		Annotations: write,
	}, s.jiraTransitionIssue)

	// Confluence
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "confluence_search",
		Description: "Search Confluence content using CQL. Returns a compact list of matching pages.",
		Annotations: readOnly,
	}, s.confluenceSearch)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "confluence_get_page",
		Description: "Get a Confluence page by id, with its body rendered to plain text.",
		Annotations: readOnly,
	}, s.confluenceGetPage)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "confluence_create_page",
		Description: "Create a new Confluence page in a space. Body accepts Markdown.",
		Annotations: write,
	}, s.confluenceCreatePage)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "confluence_update_page",
		Description: "Replace the body (and optionally title) of an existing Confluence page. Body accepts Markdown.",
		Annotations: write,
	}, s.confluenceUpdatePage)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "confluence_add_comment",
		Description: "Add a comment to a Confluence page. Body accepts Markdown.",
		Annotations: write,
	}, s.confluenceAddComment)
}
