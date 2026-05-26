# Implementation Plan: `atlassian-mcp` — a Go MCP server for Jira & Confluence

> Module path: `github.com/grayjourney/atlassian-mcp`
> Status: MVP scope. Based on analysis of the Python reference `mcp-atlassian`.

## 1. What the reference repo does (analysis)

The Python `mcp-atlassian` is structured as:

- **`servers/jira.py` + `servers/confluence.py`** — FastMCP tool definitions. Each tool is an async
  function with `Annotated[..., Field(description=...)]` params (these descriptions are what the LLM
  sees) and returns a JSON string.
- **`jira/` + `confluence/` mixins** — the actual REST client logic (built on `atlassian-python-api`
  + `requests`). Tool naming convention: `{service}_{action}_{target}`.
- **`*/config.py`** — `from_env()` factories. For Cloud the relevant path is **basic auth =
  `JIRA_USERNAME` + `JIRA_API_TOKEN`** (and Confluence equivalents). Also supports PAT/OAuth and
  Server/DC, plus `READ_ONLY_MODE` to block writes, and `is_cloud` branching.
- **`models/` (Pydantic)** — `to_simplified_dict()` flattens fat API responses into compact JSON for
  the LLM.
- **`preprocessing/`** — converts Jira ADF / Confluence storage-XHTML ↔ Markdown.

Confirmed from source: Jira Cloud search uses the **new** `POST /rest/api/3/search/jql` endpoint (the
old `/search` is deprecated); writes are guarded by a `@check_write_access` decorator.

We replicate the *shape* (compact JSON tool outputs, `{service}_{action}` naming, markdown
conversion, lazy auth) in Go, scoped to **10 MVP tools, Cloud + API-token only**.

## 2. Decisions locked

- **SDK:** official `github.com/modelcontextprotocol/go-sdk/mcp` (typed structs → auto JSON schema via
  `jsonschema` tags; `StdioTransport`).
- **Auth:** Cloud basic auth only (`email:api_token` → `Authorization: Basic`). Server/DC + PAT/OAuth
  noted as future work in code comments + README.
- **Config:** file at `~/.atlassian-mcp/config.json`, written by the local dashboard; env vars
  override. `IsConfigured()` gate.
- **Content:** Markdown in; convert to ADF (Jira) / storage-XHTML (Confluence). On read, return both
  `raw` and a `text`/markdown rendering. Pragmatic subset, not full fidelity.

## 3. Repo layout

```
atlassian-mcp/
├── go.mod                          # module github.com/grayjourney/atlassian-mcp
├── README.md
├── docs/implementation-plan/mvp-plan.md
├── cmd/atlassian-mcp/main.go       # flags: --check-config, --dashboard-port; starts dashboard + stdio MCP
├── internal/
│   ├── config/config.go            # Config struct, Load (file+env), Save, Path, IsConfigured
│   ├── atlassian/
│   │   ├── client.go               # HTTP client, basic auth, do(), error mapping (401/403/404)
│   │   ├── jira.go                 # Search, GetIssue, CreateIssue, UpdateIssue, GetTransitions, Transition
│   │   └── confluence.go           # Search, GetPage, CreatePage, UpdatePage, AddComment
│   ├── content/
│   │   ├── adf.go                  # markdown→ADF (subset) + ADF→text
│   │   └── storage.go              # markdown→storage-XHTML + storage→text
│   ├── tools/
│   │   ├── common.go               # ensureConfigured guard, toJSON, READ_ONLY guard
│   │   ├── jira_tools.go           # 5 jira_* tool registrations + handlers
│   │   └── confluence_tools.go     # 5 confluence_* tool registrations + handlers
│   └── dashboard/
│       ├── server.go               # 127.0.0.1:<port> HTTP server (goroutine)
│       ├── handlers.go             # GET / , POST /config , GET /health
│       └── index.html              # embedded (go:embed) guide + auth form
└── plugin/                         # Claude Code plugin
    ├── .claude-plugin/plugin.json
    ├── .mcp.json                   # registers the built binary as an MCP server
    └── hooks/
        ├── hooks.json              # SessionStart → open-dashboard.sh
        └── open-dashboard.sh       # opens browser only if not yet configured
```

## 4. The 10 MVP tools → REST mapping

| Tool | Method / endpoint (Cloud) | Notes |
|---|---|---|
| `jira_search` | `POST /rest/api/3/search/jql` | body `{jql, fields[], maxResults, nextPageToken}` |
| `jira_get_issue` | `GET /rest/api/3/issue/{key}?fields=&expand=` | flatten + ADF→text on description/comments |
| `jira_create_issue` | `POST /rest/api/3/issue` | `fields{project.key,summary,issuetype.name,description(ADF),assignee}` |
| `jira_update_issue` | `PUT /rest/api/3/issue/{key}` | partial `fields{}` |
| `jira_transition_issue` | `GET` then `POST /rest/api/3/issue/{key}/transitions` | list to resolve name→id, then apply `{transition.id}` |
| `confluence_search` | `GET /wiki/rest/api/search?cql=&limit=` | compact results |
| `confluence_get_page` | `GET /wiki/rest/api/content/{id}?expand=body.storage,version,space` | storage→markdown/text |
| `confluence_create_page` | `POST /wiki/rest/api/content` | `{type:page,title,space.key,body.storage,ancestors?}` |
| `confluence_update_page` | `PUT /wiki/rest/api/content/{id}` | fetch current version first, send `version.number+1` |
| `confluence_add_comment` | `POST /wiki/rest/api/content` | `{type:comment,container{id,type:page},body.storage}` |

Confluence uses the v1 `/wiki/rest/api` endpoints for the MVP; v2 migration noted as future work.

## 5. Auth dashboard + plugin flow (serena-style)

- **Server startup** (`main.go`): load config → launch dashboard HTTP server on `127.0.0.1:24285`
  (override via `ATLASSIAN_MCP_DASHBOARD_PORT`) in a goroutine → run MCP over stdio (blocking).
- **Tool guard**: every handler calls `ensureConfigured()`. If not configured, it returns a result
  telling the user to open `http://127.0.0.1:24285` — so the model surfaces the setup link instead of
  a raw 401.
- **Dashboard page** (`index.html`): on load hits `/health`; if unconfigured, shows the guide ("Go to
  https://id.atlassian.com/manage-profile/security/api-tokens, create a token") + a form for
  `jira_url, jira_username, jira_api_token, confluence_url, confluence_username,
  confluence_api_token`. Submitting `POST /config` writes the config file; the server hot-reloads via
  an atomic config pointer.
- **Plugin hook** (`SessionStart`): `open-dashboard.sh` runs `atlassian-mcp --check-config`; if
  already configured it exits silently (no nag), otherwise it polls `/health` for a few seconds then
  `open`s the browser. This avoids the chicken-and-egg of hook vs. MCP-server start order.
- **`.mcp.json`** in the plugin registers the server (`command: atlassian-mcp`, built binary on
  PATH). Since Go has no `uvx`, the README documents `go install ./cmd/atlassian-mcp` (or a shipped
  prebuilt binary) — the env block stays empty because config lives in the dashboard-written file.

## 6. Build order (TDD where it pays off)

1. `go.mod` + skeleton + `config` package (+ tests: file load, env override, `IsConfigured`).
2. `atlassian/client.go` + `jira.go`/`confluence.go` against an `httptest` mock server (tests assert
   request shape & response parsing).
3. `content` conversion (+ table-driven tests: markdown→ADF, ADF→text, storage round-trip).
4. `tools/*` registration & handlers wired to the official SDK; manual smoke via stdio.
5. `dashboard` (embedded HTML, `/health`, `/config`) + hot reload.
6. `plugin/` manifest, hook script, `.mcp.json`.
7. `README.md`: install, token creation, plugin install, and a "future work" section.

## 7. Out of scope for MVP (future work)

Server/DC + PAT + OAuth auth, `READ_ONLY_MODE`, attachments, the other ~40 tools, Confluence v2 API,
multi-tenant header auth, rich ADF fidelity.
