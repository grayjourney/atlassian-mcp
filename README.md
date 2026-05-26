# atlassian-mcp

A small [MCP](https://modelcontextprotocol.io) server, written in Go, that gives
Claude Code (or any MCP client) tools to work with **Jira** and **Confluence** on
**Atlassian Cloud**. Inspired by the Python
[`mcp-atlassian`](https://github.com/sooperset/mcp-atlassian) project, scoped down
to a focused MVP.

Credentials are entered through a tiny **local setup dashboard** (à la Serena) —
no hand-editing of config files required.

## Tools

| Jira | Confluence |
| --- | --- |
| `jira_search` — search with JQL | `confluence_search` — search with CQL |
| `jira_get_issue` — issue details | `confluence_get_page` — page content |
| `jira_create_issue` — create issues | `confluence_create_page` — create pages |
| `jira_update_issue` — update issues | `confluence_update_page` — update pages |
| `jira_transition_issue` — change status | `confluence_add_comment` — add comments |

Page/issue bodies accept **Markdown** on input (converted to Jira ADF / Confluence
storage XHTML) and are returned as plain text on read.

## Install

```bash
go install github.com/grayjourney/atlassian-mcp/cmd/atlassian-mcp@latest
```

This puts the `atlassian-mcp` binary on your `$GOPATH/bin` (make sure that's on
your `PATH`).

## Use as a Claude Code plugin

The `plugin/` directory is a ready-to-use Claude Code plugin. It registers the
MCP server and adds a `SessionStart` hook that opens the setup dashboard in your
browser the first time you start Claude Code without credentials configured.

1. Install the binary (above).
2. Add the plugin (point Claude Code at this repo's `plugin/` directory, or
   publish it to a plugin marketplace).
3. Start Claude Code. If you're not configured yet, your browser opens at
   `http://127.0.0.1:24285`.

## Setup dashboard

However you run the server, it hosts a loopback-only dashboard at
`http://127.0.0.1:24285` (override with `ATLASSIAN_MCP_DASHBOARD_PORT`).

1. Go to <https://id.atlassian.com/manage-profile/security/api-tokens> and create
   an API token. The same token works for both Jira and Confluence.
2. Open the dashboard and fill in your URL, email, and token for the service(s)
   you want. Confluence URL is usually your Jira URL + `/wiki`.
3. Save. The values are written to `~/.atlassian-mcp/config.json` (`0600`) and
   picked up immediately — no restart needed.

### Alternative: environment variables

If you'd rather not use the dashboard, set these in the MCP server's `env` block
(they override the config file):

```json
{
  "mcpServers": {
    "atlassian-mcp": {
      "command": "atlassian-mcp",
      "env": {
        "JIRA_URL": "https://your-company.atlassian.net",
        "JIRA_USERNAME": "your.email@company.com",
        "JIRA_API_TOKEN": "your_api_token",
        "CONFLUENCE_URL": "https://your-company.atlassian.net/wiki",
        "CONFLUENCE_USERNAME": "your.email@company.com",
        "CONFLUENCE_API_TOKEN": "your_api_token"
      }
    }
  }
}
```

## Layout

```
cmd/atlassian-mcp   entrypoint (stdio MCP + dashboard goroutine)
internal/config     config file + env-override loading
internal/atlassian  REST client (Jira v3, Confluence v1)
internal/content    Markdown ↔ ADF / storage XHTML conversion
internal/tools      the 10 MCP tools
internal/dashboard  loopback setup UI
plugin/             Claude Code plugin (manifest, .mcp.json, SessionStart hook)
docs/               implementation plan
tickets-details/    ticket write-up
```

## Scope & future work

MVP is **Atlassian Cloud + API-token (basic auth)** only. Deliberately not yet
implemented (PRs welcome):

- Server / Data Center, Personal Access Tokens, and OAuth 2.0
- `READ_ONLY_MODE` to block write tools
- Attachments, worklogs, and the rest of the Python project's ~50 tools
- Confluence Cloud v2 content API
- Higher-fidelity Markdown (inline bold/italic/links, tables, code blocks)
- Resolving Jira assignees by email/name (currently expects an account ID)

## Development

```bash
go test ./...
go vet ./...
go build ./...
```
