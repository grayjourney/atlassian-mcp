# Live Jira verification + Claude Code install

Verify the existing `atlassian-mcp` MVP against a **real** Atlassian Cloud Jira
(`https://grayjourney.atlassian.net`, project `KAN`), install it into Claude
Code, and write a usage guide. No production code changed — this is a
verification + install + docs task on top of
[`mvp-jira-confluence-mcp`](../mvp-jira-confluence-mcp/README.md).

## Goal

Prove the five Jira tools work end-to-end over a real MCP stdio handshake against
a live instance, then make them usable from Claude Code on this machine.

## Changes summary

| File | Status | What |
| --- | --- | --- |
| `docs/using-with-claude-code.md` | new | Step-by-step guide: install binary, supply credentials (dashboard / config file / env), register with Claude Code (`claude mcp add` / project `.mcp.json` / plugin), example prompts mapped to `KAN`, troubleshooting, security. |
| `tickets-details/live-jira-verification/README.md` | new | This write-up. |
| `~/.atlassian-mcp/config.json` | new (machine state, not repo) | Jira credentials, mode `0600`. Written outside the repo. |
| Claude Code user config (`~/.claude.json`) | modified (machine state) | `atlassian-mcp` MCP server added at user scope. |

No `.go` files were touched. The automated suite is unchanged and re-confirmed
green (below).

## Pre-flight: automated suite

```
go test ./...   # all internal/* packages ok
go vet ./...    # clean
go build ./...  # compiles
```

All green on Go 1.26.1 / darwin-arm64.

## Live verification (real MCP stdio handshake)

The server was driven exactly as a client does: `initialize` →
`notifications/initialized` → `tools/call`, with credentials supplied via
`JIRA_URL` / `JIRA_USERNAME` / `JIRA_API_TOKEN` env and an isolated
`ATLASSIAN_MCP_CONFIG=/tmp/none.json` so the real config file wasn't involved.

| # | Tool | Input | Result |
| --- | --- | --- | --- |
| 1 | `tools/list` | — | all 10 tool names returned |
| 2 | `jira_search` | `project = KAN ORDER BY created DESC` | `count: 0` (empty board) — auth OK |
| 3 | `jira_create_issue` | `KAN` / "MCP smoke test…" / Task / Markdown body | `KAN-1`, id `10000`, browse URL |
| 4 | `jira_get_issue` | `KAN-1` | status `To Do`, type `Task`, priority `Medium`, description rendered to text |
| 5 | `jira_transition_issue` | `KAN-1` → `In Progress` + comment | `transitioned_to: In Progress`, id `21` |
| 6 | `jira_update_issue` | `KAN-1` summary + `{"labels":["mcp-test"]}` | `updated: true` |
| 7 | `jira_get_issue` (re-read) | `KAN-1` | status `In Progress`, summary updated — confirms 5 & 6 |
| 8 | `jira_search` (re-read) | `project = KAN` | `count: 1`, returns `KAN-1` |
| 9 | `jira_transition_issue` (error path) | `KAN-1` → "Bogus Status" | `isError`, lists valid options: `"To Do" (id 11), "In Progress" (id 21), "Done" (id 31)` |
| 10 | not-configured guard | `jira_search` with no creds | `isError`, "Jira is not configured. Open the setup dashboard at …" |

**Outcome:** all five Jira tools work against the live instance; the happy path,
the helpful error path (unknown transition lists options), and the
not-configured guard all behave as designed. `KAN-1` was left on the board as the
verification artifact (currently `In Progress`, label `mcp-test`).

## Install performed

1. `go install ./cmd/atlassian-mcp` → `~/go/bin/atlassian-mcp` (on `PATH`), `--version` = `0.1.0`.
2. Wrote `~/.atlassian-mcp/config.json` (Jira only, Confluence blank) at mode `0600`; `atlassian-mcp --check-config` → `configured` (exit 0).
3. `claude mcp add --scope user atlassian-mcp atlassian-mcp`.
4. `claude mcp list` → `atlassian-mcp - ✓ Connected`.

## How to reproduce

The full command sequence — handshake driver, each `tools/call`, and the install
steps — is in [`docs/using-with-claude-code.md`](../../docs/using-with-claude-code.md)
(§4 for usage, §2–3 for install). To re-run the raw smoke test, set the three
`JIRA_*` env vars and pipe the JSON-RPC `initialize`/`tools/call` lines into
`atlassian-mcp --dashboard-port <free-port>` as shown in
[`../mvp-jira-confluence-mcp/03-manual-testing.md`](../mvp-jira-confluence-mcp/03-manual-testing.md).

## Suggested branch

`live-jira-verification` (docs + verification only; no source change). Not
committed — that's yours to trigger.

## Cleanup

`KAN-1` is a throwaway test issue. Delete it from the board when you no longer
need the evidence, or keep it as a marker that the MCP round-trips.
