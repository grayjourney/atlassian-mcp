# MVP — Jira & Confluence MCP server (Go)

Build a Go MCP server exposing 10 Jira/Confluence tools for Atlassian Cloud, with
API-token auth entered through a local setup dashboard, packaged as a Claude Code
plugin. Based on the analysis in
[`docs/implementation-plan/mvp-plan.md`](../../docs/implementation-plan/mvp-plan.md).

| Doc | Contents |
| --- | --- |
| [01-changes-summary.md](01-changes-summary.md) | Every file added and what it does |
| [02-solution-design.md](02-solution-design.md) | The design, key APIs, and the alternatives weighed |
| [03-manual-testing.md](03-manual-testing.md) | How to verify it — automated + end-to-end |

**Outcome:** all 10 tools register and respond over a real MCP stdio handshake;
`go test ./...`, `go vet ./...`, and `go build ./...` are clean. Auth is gated
with a helpful "open the dashboard" message until credentials are set.
