#!/usr/bin/env bash
# SessionStart hook: if the Atlassian MCP server has no credentials yet, wait for
# its loopback setup dashboard to come up and open it in the browser. Stays
# silent (and non-blocking) once configured. All output goes to stderr so it
# never pollutes the session context.
set -euo pipefail

PORT="${ATLASSIAN_MCP_DASHBOARD_PORT:-24285}"
URL="http://127.0.0.1:${PORT}"

BIN="$(command -v atlassian-mcp || true)"
if [ -z "$BIN" ]; then
  echo "atlassian-mcp not on PATH. Install: go install github.com/grayjourney/atlassian-mcp/cmd/atlassian-mcp@latest" >&2
  exit 0
fi

# Already configured → nothing to do.
if "$BIN" --check-config >/dev/null 2>&1; then
  exit 0
fi

# The dashboard is hosted by the MCP server, which Claude Code starts around the
# same time as this hook — poll briefly for it to be reachable.
for _ in $(seq 1 20); do
  if curl -fsS "${URL}/health" >/dev/null 2>&1; then
    break
  fi
  sleep 0.5
done

if command -v open >/dev/null 2>&1; then
  open "$URL" >/dev/null 2>&1 || true
  echo "Atlassian MCP needs setup — opened ${URL} in your browser." >&2
elif command -v xdg-open >/dev/null 2>&1; then
  xdg-open "$URL" >/dev/null 2>&1 || true
  echo "Atlassian MCP needs setup — opened ${URL} in your browser." >&2
else
  echo "Atlassian MCP needs setup — open ${URL} in your browser." >&2
fi
exit 0
