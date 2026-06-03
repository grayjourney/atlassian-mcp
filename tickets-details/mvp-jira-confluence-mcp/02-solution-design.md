# Solution design

## Goal

Reproduce the *useful shape* of the Python `mcp-atlassian` — compact JSON tool
outputs, `{service}_{action}` tool names, Markdown ↔ rich-text conversion, lazy
auth — in Go, scoped to 10 tools and Atlassian Cloud basic auth.

## Layered architecture

```
cmd/atlassian-mcp ── main: stdio MCP server + dashboard goroutine
        │
        ├── internal/dashboard ── owns the live *config.Config (atomic), serves the setup UI
        │
        └── internal/tools ── 10 MCP tool handlers (go-sdk)
                 │              reads config via a provider func; builds clients per call
                 ├── internal/atlassian ── thin REST client (Jira v3, Confluence v1)
                 └── internal/content ── Markdown ↔ ADF / storage XHTML
```

Each layer is independently testable: the client against `httptest`, content
conversion as pure functions, the dashboard via its `http.Handler`, and the tool
helpers as pure functions.

## Key decisions

### MCP SDK — official `modelcontextprotocol/go-sdk`

Handlers use the typed form `func(ctx, *mcp.CallToolRequest, In) (*CallToolResult,
Out, error)`. Input structs carry `jsonschema:"..."` description tags; the SDK
infers the input schema and validates calls. **`Out` is typed `any`**, which makes
the SDK skip output-schema inference — so each handler returns its compact result
as a single `TextContent` JSON blob (mirroring the Python project's JSON-string
returns) without committing to a rigid output schema per tool.

### Auth config — file written by the dashboard, env overrides

`config.Load` reads `~/.atlassian-mcp/config.json` then overlays any of the
`JIRA_*` / `CONFLUENCE_*` env vars. The dashboard is the source of truth for the
no-env install; the env block in `.mcp.json` still works for those who prefer it.
The file is `0600` since it holds API tokens.

### Hot reload via an atomic pointer

The `Dashboard` holds `atomic.Pointer[config.Config]`. `POST /config` writes the
file and swaps the pointer; tools read config through `dashboard.Config`, so a
saved change takes effect on the next tool call with **no server restart**. This
is what lets the SessionStart-hook flow work: the server can boot unconfigured,
and the user fills in credentials live.

### The not-configured guard

Every handler calls `s.jira()` / `s.confluence()`, which return an error naming
the dashboard URL when credentials are missing. The SDK turns that into an
`isError` tool result, so the model sees *"open the dashboard at
http://127.0.0.1:24285"* instead of a raw 401 — turning a failure into a setup
nudge.

### Markdown ↔ rich text

Jira Cloud v3 wants **ADF** (a JSON doc) for descriptions/comments; Confluence
wants **storage XHTML**. A single `parseBlocks` Markdown parser (headings,
paragraphs, bullet lists) feeds both renderers. On read, ADF and storage are
flattened back to plain text. It's a deliberate block-level subset — inline
marks pass through as literal text — which keeps the converter small and
predictable. Higher fidelity is listed as future work.

## Alternatives considered

| Decision | Chosen | Rejected | Why |
| --- | --- | --- | --- |
| MCP SDK | official `go-sdk` | `mark3labs/mcp-go` | Spec alignment + long-term maintenance; typed-struct schema inference removes boilerplate. |
| Tool output | `Out = any` + text JSON | typed `Out` per tool with output schema | 10 heterogeneous compact shapes aren't worth 10 output schemas; text JSON matches the Python project and stays flexible. |
| Auth storage | dashboard-written file + env override | env-only in `.mcp.json` | The brief asked for the Serena-style dashboard; the file lets the hook configure live. Env override retained for power users. |
| Config refresh | atomic pointer, per-call read | restart on change / fsnotify watcher | Simplest correct option; no restart, no extra dependency, no file-watch races. |
| Markdown | block-level subset | full CommonMark lib | MVP only needs headings/paragraphs/lists; a full parser is scope the brief explicitly deferred. |
| Confluence API | v1 `/wiki/rest/api` | v2 `/api/v2` | v1 still serves all five operations and is simpler; v2 migration is future work. |

## Known MVP limitations (intentional)

- Cloud + API-token only (no Server/DC, PAT, OAuth).
- `jira_create_issue` assignee expects an **account ID**, not email/name.
- No `READ_ONLY_MODE`; writes are flagged non-destructive via tool annotations.
- Markdown inline formatting (bold/italic/links/tables) is not converted.
