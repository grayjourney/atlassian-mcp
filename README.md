# atlassian-mcp

A small [MCP](https://modelcontextprotocol.io) server, written in Go, that gives
Claude Code (or any MCP client) tools to work with **Jira** and **Confluence** on
**Atlassian Cloud**. Inspired by the Python
[`mcp-atlassian`](https://github.com/sooperset/mcp-atlassian) project, scoped down
to a focused MVP.

Credentials are entered through a tiny **local setup dashboard** (à la Serena) —
no hand-editing of config files required.

## Tools

**Jira** — issues & fields:

| Tool | What it does |
| --- | --- |
| `jira_search` | search issues with JQL |
| `jira_get_issue` | rich issue details (status, dates, labels, components, story points, sprints, …) |
| `jira_create_issue` | create issues with due date, story points, assignee (by email), labels, components, fix versions, parent/epic, priority, and any custom field by name |
| `jira_update_issue` | update any of the above on an existing issue |
| `jira_transition_issue` | change status by name/id, optionally with a comment |
| `jira_delete_issue` | delete an issue (and optionally its subtasks) |
| `jira_get_changelog` | issue change history |
| `jira_list_fields` | discover field names/ids (system + custom) |
| `jira_get_field_options` | allowed values of a select/multi-select field |

**Jira** — comments, time, people:

| Tool | What it does |
| --- | --- |
| `jira_add_comment` / `jira_list_comments` / `jira_edit_comment` | add, read, and edit issue comments (Markdown) |
| `jira_add_worklog` / `jira_get_worklog` | log and read time spent (e.g. "2h 30m") |
| `jira_get_issue_dates` | all date fields (created, updated, due, resolved, custom dates) |
| `jira_list_watchers` / `jira_add_watcher` / `jira_remove_watcher` | manage watchers (by email/name/id) |
| `jira_get_user` | resolve users by email/name to an account id |

**Jira** — attachments:

| Tool | What it does |
| --- | --- |
| `jira_list_attachments` | list an issue's attachments (filename, size, type) |
| `jira_read_attachment` | read a text attachment's content inline (by id or filename) |
| `jira_download_attachment` | download any attachment to a local file, return its path |

**Jira** — agile (boards & sprints):

| Tool | What it does |
| --- | --- |
| `jira_list_boards` | list boards (Scrum/Kanban), optionally by project |
| `jira_get_board_issues` | issues on a board (optional JQL) |
| `jira_list_sprints` / `jira_get_active_sprint` | a Scrum board's sprints / its current sprint |
| `jira_get_sprint_issues` | issues in a sprint (optional JQL) |
| `jira_create_sprint` / `jira_update_sprint` | create a sprint; rename/start/close/reschedule |
| `jira_move_issues_to_sprint` | move issues into a sprint |

**Jira** — projects, versions, components, links:

| Tool | What it does |
| --- | --- |
| `jira_list_projects` | list/search projects |
| `jira_get_project_versions` / `jira_create_version` | list & create versions (releases / milestones) |
| `jira_get_project_components` | list a project's components |
| `jira_list_link_types` | available link types (Blocks, Relates, …) |
| `jira_create_issue_link` / `jira_remove_issue_link` | link / unlink two issues |
| `jira_link_to_epic` | put an issue under an epic (Epic Link or parent) |
| `jira_create_remote_link` | attach a web link to an issue |

**Jira** — reporting:

| Tool | What it does |
| --- | --- |
| `jira_board_report` | board summary: counts by status/assignee/type, done vs remaining, story points |
| `jira_sprint_report` | same, for a sprint |
| `jira_version_report` | same, for a version / milestone (by fix version) |

**Jira** — service management & development:

| Tool | What it does |
| --- | --- |
| `jira_list_service_desks` | list JSM service desks you can access |
| `jira_list_queues` / `jira_get_queue_issues` | a service desk's queues / a queue's issues |
| `jira_get_development_info` | an issue's branches, commits & pull requests (from connected Bitbucket/GitHub/GitLab) |

**Confluence:**

| Tool | What it does |
| --- | --- |
| `confluence_search` | search with CQL |
| `confluence_get_page` | page content |
| `confluence_create_page` | create pages |
| `confluence_update_page` | update pages |
| `confluence_add_comment` | add comments |

Page/issue bodies accept **Markdown** on input (converted to Jira ADF / Confluence
storage XHTML) and are returned as plain text on read. Custom fields, story
points, due dates, and assignees can be set by **human name / email** — the
server resolves them to Jira's internal ids.

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

## Using with Claude Code

## 1. Install the binary

The MCP server is a single Go binary. Install it onto your `PATH`:

```bash
# From a published module:
go install github.com/grayjourney/atlassian-mcp/cmd/atlassian-mcp@latest

# …or from this local checkout:
cd /Users/tran.quang.huyf/Works/go/src/github/atlassian-mcp
go install ./cmd/atlassian-mcp
```

This drops `atlassian-mcp` in `$(go env GOPATH)/bin` (here:
`~/go/bin/atlassian-mcp`). Confirm it's reachable:

```bash
command -v atlassian-mcp   # -> ~/go/bin/atlassian-mcp
atlassian-mcp --version    # -> 0.1.0
```

If `command -v` finds nothing, add `~/go/bin` to your `PATH` in `~/.zshrc`.

## 2. Provide credentials

You need an Atlassian API token (the same token works for Jira and Confluence):
create one at <https://id.atlassian.com/manage-profile/security/api-tokens>.

There are three ways to feed credentials to the server — pick **one**.

### Option A — the setup dashboard (recommended, no file editing)

The server hosts a loopback-only web form. Start it once, fill in the form, save:

```bash
atlassian-mcp --dashboard-port 24285 &   # or just start Claude Code (see §3)
open http://127.0.0.1:24285
```

Fill in **Jira URL** (`https://grayjourney.atlassian.net`), **email**, and
**token**, then Save. It writes `~/.atlassian-mcp/config.json` at `0600` and is
picked up immediately — no restart. Confluence URL is usually your Jira URL +
`/wiki`; leave it blank if you only want Jira.

### Option B — the config file directly

Write `~/.atlassian-mcp/config.json` yourself (this is what the dashboard
produces). Keep it owner-only since it holds a token:

```bash
mkdir -p ~/.atlassian-mcp && chmod 700 ~/.atlassian-mcp
umask 077
cat > ~/.atlassian-mcp/config.json <<'JSON'
{
  "jira_url": "https://grayjourney.atlassian.net",
  "jira_username": "gray.tran201@gmail.com",
  "jira_api_token": "YOUR_API_TOKEN",
  "confluence_url": "",
  "confluence_username": "",
  "confluence_api_token": ""
}
JSON
chmod 600 ~/.atlassian-mcp/config.json
atlassian-mcp --check-config   # -> "configured", exit 0
```

### Option C — environment variables in the MCP registration

If you'd rather not have a config file, put the credentials in the server's
`env` block (these override the config file). See §3.

## 3. Register the server with Claude Code

Three ways, in order of simplicity.

### 3a. `claude mcp add` (user scope — what's set up here)

Makes the tools available in **every** project on this machine:

```bash
claude mcp add --scope user atlassian-mcp atlassian-mcp
claude mcp list
# atlassian-mcp: atlassian-mcp  - ✓ Connected
```

This relies on the credentials from §2 (config file or dashboard). To inline the
credentials instead (Option C), pass them as env:

```bash
claude mcp add --scope user atlassian-mcp \
  --env JIRA_URL=https://grayjourney.atlassian.net \
  --env JIRA_USERNAME=gray.tran201@gmail.com \
  --env JIRA_API_TOKEN=YOUR_API_TOKEN \
  -- atlassian-mcp
```

To remove it later: `claude mcp remove --scope user atlassian-mcp`.

### 3b. Project-scoped `.mcp.json`

To share the server with a single repo's collaborators, drop a `.mcp.json` at the
project root:

```json
{
  "mcpServers": {
    "atlassian-mcp": { "command": "atlassian-mcp", "args": [], "env": {} }
  }
}
```

Claude Code will ask you to approve the project MCP server the first time.

### 3c. As a Claude Code plugin (the `plugin/` directory)

The repo ships a ready-made plugin in `plugin/` that registers the server **and**
adds a `SessionStart` hook: the first time you start Claude Code without
credentials, it opens the dashboard in your browser automatically. Install it by
pointing Claude Code at a marketplace that serves this `plugin/` directory. The
plugin's `.mcp.json` calls `atlassian-mcp` from your `PATH`, so step 1 is still a
prerequisite.

> The user-scope registration (3a) and the plugin (3c) do the same job. Use one,
> not both, to avoid a duplicate server entry.

## 4. Using it from Claude Code

You don't call the tools by name — you ask Claude in plain language and it picks
the right tool. Restart Claude Code (or start a new session) after registering so
it discovers the tools. The five Jira tools and example prompts:

| You say | Tool used | What happens |
| --- | --- | --- |
| "Search Jira for open issues in KAN" | `jira_search` | runs JQL `project = KAN AND statusCategory != Done` |
| "Show me KAN-1" | `jira_get_issue` | returns summary, status, assignee, description |
| "Create a Task in KAN titled 'Fix login', describe it as …" | `jira_create_issue` | creates the issue, returns its key + URL |
| "Rename KAN-1 to … and add label backend" | `jira_update_issue` | patches summary + `labels` |
| "Move KAN-1 to In Progress with a comment" | `jira_transition_issue` | transitions by status name + adds comment |

Concrete things to try against your Kanban board (`KAN`):

- *"List the most recently created issues in project KAN."*
- *"Create a Task in KAN called 'Wire up auth' with a markdown description that
  has a bullet list of two steps."*
- *"What's the status and description of KAN-1?"*
- *"Move KAN-1 to Done and leave the comment 'verified via MCP'."*
- *"Add the label `mcp-test` to KAN-1 and change its title to 'Auth (done)'."*

Notes that match how the tools behave:

- **Descriptions/comments accept Markdown** on input (converted to Jira's ADF) and
  come back as plain text on read. Markdown fidelity is intentionally basic
  (paragraphs + bullet lists); inline bold/links render as literal text for now.
- **Transitions** are by status *name* (`In Progress`, `Done`) or numeric id. Ask
  for a status that doesn't exist and the tool replies with the valid options,
  e.g. `"To Do" (id 11), "In Progress" (id 21), "Done" (id 31)`.
- **Assignee** can be given by email, display name, or account id — the server
  resolves it via user search.
- **Search** defaults to 10 results (max 50) and returns a compact projection
  (key, summary, status, type, assignee, priority, updated).
- Write tools are flagged non-destructive; Claude Code may ask you to confirm
  before a create/update/transition depending on your permission settings.

## 5. Verify the connection

```bash
claude mcp list
# atlassian-mcp: atlassian-mcp  - ✓ Connected
```

## Layout

```
cmd/atlassian-mcp   entrypoint (stdio MCP + dashboard goroutine)
internal/config     config file + env-override loading
internal/atlassian  REST client (Jira v3, Confluence v1)
internal/content    Markdown ↔ ADF / storage XHTML conversion
internal/tools      the MCP tools (42 Jira + 5 Confluence)
internal/dashboard  loopback setup UI
plugin/             Claude Code plugin (manifest, .mcp.json, SessionStart hook)
docs/               implementation plan
tickets-details/    ticket write-up
```

## Scope & future work

Auth is **Atlassian Cloud + API-token (basic auth)** only. The Jira tool surface
now spans issues/fields, comments/worklog/watchers, attachments, agile
(boards/sprints), projects/versions/components/links, reporting, and service
management + development info.

Still not implemented (PRs welcome):

- Server / Data Center, Personal Access Tokens, and OAuth 2.0
- `READ_ONLY_MODE` to block write tools
- **ProForma forms** — the Forms API lives on `api.atlassian.com` and requires
  OAuth/PAT bearer auth, so it can't work under the basic-auth model; it's gated
  behind the OAuth item above.
- Confluence Cloud v2 content API and the broader Confluence tool set
- Higher-fidelity Markdown (inline bold/italic/links, tables, code blocks)
- Field-map caching, and pagination for large reports/searches

## Development

```bash
go test ./...
go vet ./...
go build ./...
```
