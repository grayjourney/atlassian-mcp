package tools

import "github.com/modelcontextprotocol/go-sdk/mcp"

func ptr[T any](v T) *T { return &v }

// Register adds all MVP tools to the MCP server. Read tools carry a readOnly
// hint; writes are flagged non-destructive so clients can prompt for confirmation.
func (s *Server) Register(srv *mcp.Server) {
	readOnly := &mcp.ToolAnnotations{ReadOnlyHint: true}
	write := &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: new(false)}
	destructive := &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: ptr(true)}

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
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_delete_issue",
		Description: "Permanently delete a Jira issue (optionally its subtasks).",
		Annotations: destructive,
	}, s.jiraDeleteIssue)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_get_changelog",
		Description: "Get the change history (field transitions over time) of a Jira issue.",
		Annotations: readOnly,
	}, s.jiraGetChangelog)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_list_fields",
		Description: "List Jira fields and their ids (system and custom). Use to find the name/id of a custom field.",
		Annotations: readOnly,
	}, s.jiraListFields)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_get_field_options",
		Description: "List the allowed values of a Jira select/multi-select custom field.",
		Annotations: readOnly,
	}, s.jiraGetFieldOptions)

	// Jira — comments, worklog, dates, watchers, users
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_add_comment",
		Description: "Add a comment to a Jira issue. Body accepts Markdown.",
		Annotations: write,
	}, s.jiraAddComment)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_list_comments",
		Description: "List the comments on a Jira issue (rendered to text).",
		Annotations: readOnly,
	}, s.jiraListComments)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_edit_comment",
		Description: "Edit an existing comment on a Jira issue. Body accepts Markdown.",
		Annotations: write,
	}, s.jiraEditComment)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_add_worklog",
		Description: "Log time spent on a Jira issue (e.g. \"2h 30m\"), optionally with a comment and start time.",
		Annotations: write,
	}, s.jiraAddWorklog)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_get_worklog",
		Description: "List the worklog (time tracking) entries of a Jira issue.",
		Annotations: readOnly,
	}, s.jiraGetWorklog)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_get_issue_dates",
		Description: "Get all date fields of a Jira issue (created, updated, due, resolved, and custom dates).",
		Annotations: readOnly,
	}, s.jiraGetIssueDates)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_list_watchers",
		Description: "List the watchers of a Jira issue.",
		Annotations: readOnly,
	}, s.jiraListWatchers)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_add_watcher",
		Description: "Add a watcher to a Jira issue (by email, name, or account id).",
		Annotations: write,
	}, s.jiraAddWatcher)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_remove_watcher",
		Description: "Remove a watcher from a Jira issue (by email, name, or account id).",
		Annotations: write,
	}, s.jiraRemoveWatcher)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_get_user",
		Description: "Look up Jira users by email, name, or account id (to find an account id).",
		Annotations: readOnly,
	}, s.jiraGetUser)

	// Jira — attachments & content
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_list_attachments",
		Description: "List the attachments on a Jira issue (filename, size, type).",
		Annotations: readOnly,
	}, s.jiraListAttachments)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_download_attachment",
		Description: "Download a Jira issue attachment (by id or filename) to a local file and return its path.",
		Annotations: readOnly,
	}, s.jiraDownloadAttachment)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_read_attachment",
		Description: "Read a text Jira attachment (by id or filename) and return its content inline.",
		Annotations: readOnly,
	}, s.jiraReadAttachment)

	// Jira — agile: boards & sprints
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_list_boards",
		Description: "List Jira agile boards (Scrum/Kanban), optionally filtered to a project.",
		Annotations: readOnly,
	}, s.jiraListBoards)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_get_board_issues",
		Description: "List the issues on an agile board, optionally narrowed by JQL.",
		Annotations: readOnly,
	}, s.jiraGetBoardIssues)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_list_sprints",
		Description: "List a Scrum board's sprints, optionally filtered by state (active/future/closed).",
		Annotations: readOnly,
	}, s.jiraListSprints)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_get_active_sprint",
		Description: "Get the currently active sprint(s) of a Scrum board.",
		Annotations: readOnly,
	}, s.jiraGetActiveSprint)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_get_sprint_issues",
		Description: "List the issues in a sprint, optionally narrowed by JQL.",
		Annotations: readOnly,
	}, s.jiraGetSprintIssues)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_create_sprint",
		Description: "Create a sprint on a Scrum board (name, optional goal and start/end dates).",
		Annotations: write,
	}, s.jiraCreateSprint)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_update_sprint",
		Description: "Update a sprint's name, goal, state (start/close), or dates.",
		Annotations: write,
	}, s.jiraUpdateSprint)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_move_issues_to_sprint",
		Description: "Move issues into a sprint by key.",
		Annotations: write,
	}, s.jiraMoveIssuesToSprint)

	// Jira — projects, versions (milestones), components, links
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_list_projects",
		Description: "List Jira projects (optionally filtered by key or name).",
		Annotations: readOnly,
	}, s.jiraListProjects)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_get_project_versions",
		Description: "List a project's versions (releases / milestones).",
		Annotations: readOnly,
	}, s.jiraGetProjectVersions)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_get_project_components",
		Description: "List a project's components.",
		Annotations: readOnly,
	}, s.jiraGetProjectComponents)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_create_version",
		Description: "Create a version (release / milestone) in a project.",
		Annotations: write,
	}, s.jiraCreateVersion)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_list_link_types",
		Description: "List the available issue link types (Blocks, Relates, ...).",
		Annotations: readOnly,
	}, s.jiraListLinkTypes)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_create_issue_link",
		Description: "Link two issues with a link type (e.g. Blocks).",
		Annotations: write,
	}, s.jiraCreateIssueLink)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_remove_issue_link",
		Description: "Remove an issue link by id.",
		Annotations: destructive,
	}, s.jiraRemoveIssueLink)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_link_to_epic",
		Description: "Put an issue under an epic (uses Epic Link or parent as the project requires).",
		Annotations: write,
	}, s.jiraLinkToEpic)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_create_remote_link",
		Description: "Attach a web link (URL + title) to an issue.",
		Annotations: write,
	}, s.jiraCreateRemoteLink)

	// Jira — reporting
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_board_report",
		Description: "Summarize a board's issues: counts by status/assignee/type, done vs remaining, story points.",
		Annotations: readOnly,
	}, s.jiraBoardReport)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_sprint_report",
		Description: "Summarize a sprint's issues: counts by status/assignee/type, done vs remaining, story points.",
		Annotations: readOnly,
	}, s.jiraSprintReport)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_version_report",
		Description: "Summarize a version/milestone's issues (by fix version): progress, breakdowns, story points.",
		Annotations: readOnly,
	}, s.jiraVersionReport)

	// Jira — service management & development info
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_list_service_desks",
		Description: "List Jira Service Management service desks (JSM projects) you can access.",
		Annotations: readOnly,
	}, s.jiraListServiceDesks)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_list_queues",
		Description: "List the request queues of a service desk.",
		Annotations: readOnly,
	}, s.jiraListQueues)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_get_queue_issues",
		Description: "List the issues currently in a service desk queue.",
		Annotations: readOnly,
	}, s.jiraGetQueueIssues)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "jira_get_development_info",
		Description: "Get an issue's development panel summary: branches, commits, and pull requests from connected VCS tools (Bitbucket/GitHub/GitLab).",
		Annotations: readOnly,
	}, s.jiraGetDevelopmentInfo)

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
