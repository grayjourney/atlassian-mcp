# Manual test guide

## 1. Automated suite

```bash
cd atlassian-mcp
go test ./...      # all packages green
go vet ./...       # clean
go build ./...     # compiles
```

Expected: every `internal/*` package reports `ok`; vet and build print nothing.

## 2. CLI flags

```bash
go build -o /tmp/atlassian-mcp ./cmd/atlassian-mcp

/tmp/atlassian-mcp --version
# -> 0.1.0

ATLASSIAN_MCP_CONFIG=/tmp/none.json /tmp/atlassian-mcp --check-config; echo "exit=$?"
# -> not configured
# -> exit=1   (the SessionStart hook uses this to decide whether to open the dashboard)
```

## 3. MCP handshake + tools/list (no credentials needed)

```bash
cat > /tmp/reqs.jsonl <<'EOF'
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"smoke","version":"0"}}}
{"jsonrpc":"2.0","method":"notifications/initialized"}
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
EOF
(cat /tmp/reqs.jsonl; sleep 2) | ATLASSIAN_MCP_CONFIG=/tmp/none.json \
  /tmp/atlassian-mcp --dashboard-port 24290 2>/dev/null \
  | grep -o '"name":"[a-z_]*"' | sort -u
```

Expected: the 10 tool names (`confluence_add_comment` … `jira_update_issue`).

### Not-configured guard

Replace the `tools/list` line with a call and confirm the guidance message:

```bash
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"jira_search","arguments":{"jql":"project = X"}}}
```

Expected result text: *"Jira is not configured. Open the setup dashboard at
http://127.0.0.1:24290 …"* with `"isError":true`.

## 4. Setup dashboard

```bash
ATLASSIAN_MCP_CONFIG=/tmp/dash.json /tmp/atlassian-mcp --dashboard-port 24285 &
sleep 1
curl -s http://127.0.0.1:24285/health        # {"configured":false,...}
open http://127.0.0.1:24285                   # fill the form, Save
curl -s http://127.0.0.1:24285/health         # {"configured":true,...} after saving
cat /tmp/dash.json                            # persisted credentials (file mode 600)
```

The dashboard prefills from `/config` and writes back on Save; `/health` flips to
`configured:true` without restarting the server.

## 5. Live Jira/Confluence (real credentials)

Create a token at <https://id.atlassian.com/manage-profile/security/api-tokens>,
enter it in the dashboard, then drive the tools from Claude Code (or any MCP
client). Quick checks:

- **jira_search** `{"jql":"assignee = currentUser() ORDER BY updated DESC","limit":5}`
  → compact list of your recent issues.
- **jira_get_issue** `{"issue_key":"PROJ-1"}` → summary/status/assignee + description as text.
- **jira_create_issue** `{"project_key":"PROJ","summary":"MCP smoke","issue_type":"Task","description":"From **atlassian-mcp**"}`
  → `{key, id, url}`; open the URL to confirm.
- **jira_transition_issue** `{"issue_key":"PROJ-1","transition":"In Progress"}`
  → `transitioned_to`; an unknown status lists the available options.
- **confluence_create_page** `{"space_key":"DOCS","title":"MCP test","content":"# Hi\n\n- a\n- b"}`
  → `{id, title, url}`.
- **confluence_get_page** `{"page_id":"<id>"}` → body rendered to plain text.
- **confluence_add_comment** `{"page_id":"<id>","content":"Looks good"}` → `{id, page_id}`.

## 6. Plugin hook (manual)

```bash
ATLASSIAN_MCP_CONFIG=/tmp/none.json CLAUDE_PLUGIN_ROOT=$PWD/plugin \
  bash plugin/hooks/open-dashboard.sh
```

With no config it waits for `/health` then opens the browser; with a configured
file it exits silently and immediately.
