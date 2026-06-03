# Changes summary

This is a greenfield repo; every file below is **new**.

## Go source

| File | What it does |
| --- | --- |
| `go.mod` | Module `github.com/grayjourney/atlassian-mcp`, Go 1.26; depends on `modelcontextprotocol/go-sdk`. |
| `cmd/atlassian-mcp/main.go` | Entrypoint. Flags `--check-config`, `--version`, `--dashboard-port`. Loads config, starts the dashboard goroutine, registers tools, runs MCP over stdio. Logs to **stderr** (stdout is the MCP transport). |
| `internal/config/config.go` | `Config` struct + `Load`/`LoadFrom`/`Save`/`SaveTo`, env-var overrides, `JiraConfigured`/`ConfluenceConfigured`/`IsConfigured`. File at `~/.atlassian-mcp/config.json` (0600). |
| `internal/atlassian/client.go` | Shared `restClient`: basic-auth header, JSON `do()`, `*APIError` for non-2xx. |
| `internal/atlassian/jira.go` | `JiraClient`: `Search` (POST `/rest/api/3/search/jql`), `GetIssue`, `CreateIssue`, `UpdateIssue`, `GetTransitions`, `TransitionIssue`. |
| `internal/atlassian/confluence.go` | `ConfluenceClient`: `Search` (CQL), `GetPage`, `CreatePage`, `UpdatePage` (auto version bump), `AddComment`. v1 `/wiki/rest/api`. |
| `internal/content/markdown.go` | `parseBlocks` — shared Markdown block parser (headings, paragraphs, bullets). |
| `internal/content/adf.go` | `MarkdownToADF` + `ADFToText` (Jira rich text). |
| `internal/content/storage.go` | `MarkdownToStorage` + `StorageToText` (Confluence XHTML). |
| `internal/tools/common.go` | Tool `Server`, lazy client builders, not-configured guard, compact-output helpers (`flattenIssue`, `resolveTransition`, `textResult`). |
| `internal/tools/jira_tools.go` | The 5 Jira tool input structs + handlers. |
| `internal/tools/confluence_tools.go` | The 5 Confluence tool input structs + handlers. |
| `internal/tools/register.go` | `Register` — adds all 10 tools with read/write annotations. |
| `internal/dashboard/server.go` | `Dashboard`: atomic hot-reloadable config, `/health`, `/config` (GET prefill / POST save), `/` UI; graceful shutdown. |
| `internal/dashboard/assets.go` | `go:embed` of the dashboard HTML. |
| `internal/dashboard/index.html` | Setup page: API-token guide + credential form (vanilla JS, fetch). |

## Tests (table-driven where it pays off)

| File | Covers |
| --- | --- |
| `internal/config/config_test.go` | Missing file, file read, env override, `IsConfigured` matrix, save round-trip + 0600 perms. |
| `internal/atlassian/atlassian_test.go` | Every client method against `httptest`: request method/path/query/auth/body + response parsing + `APIError` mapping. |
| `internal/content/content_test.go` | Markdown→ADF structure, ADF→text, Markdown→storage (incl. HTML escaping), storage→text, round-trips. |
| `internal/tools/tools_test.go` | `flattenIssue` projection + omission, `resolveTransition` (by id / case-insensitive name / miss). |
| `internal/tools/register_test.go` | `Register` doesn't panic — i.e. every input struct yields a valid JSON schema. |
| `internal/dashboard/dashboard_test.go` | `/health` reflects config, `POST /config` persists + hot-reloads, `/` serves the guide. |

## Packaging & docs

| File | What it does |
| --- | --- |
| `plugin/.claude-plugin/plugin.json` | Plugin manifest. |
| `plugin/.mcp.json` | Registers the `atlassian-mcp` binary as an MCP server. |
| `plugin/hooks/hooks.json` | `SessionStart` → `open-dashboard.sh`. |
| `plugin/hooks/open-dashboard.sh` | Opens the dashboard in a browser only when unconfigured; silent otherwise. |
| `README.md` | Install, plugin use, setup, env-var alternative, scope/future work. |
| `docs/implementation-plan/mvp-plan.md` | The plan this work followed. |
