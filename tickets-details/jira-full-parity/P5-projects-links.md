# P5 — Projects, versions (milestones), components, links

## Goal

Give Claude the project-level context and relationships: list projects, read &
create versions (releases / milestones) and components, and link issues (issue
links, epic, remote web links).

## Changes summary

| File | Status | What |
| --- | --- | --- |
| `internal/atlassian/projects.go` | new | `Project`, `Version`, `Component`, `IssueLinkType` types; `GetProjects`, `GetProject`, `GetProjectVersions`, `CreateVersion`, `GetProjectComponents`, `GetIssueLinkTypes`, `CreateIssueLink`, `DeleteIssueLink`, `CreateRemoteLink`. |
| `internal/atlassian/projects_test.go` | new | httptest coverage for all nine methods. |
| `internal/tools/projects.go` | new | 9 tool handlers + `epicLinkField` helper. |
| `internal/tools/projects_test.go` | new | epic-link fallback, link-to-epic via parent, and create-version project-id resolution tests. |
| `internal/tools/register.go` | modified | Registered the 9 P5 tools (`jira_remove_issue_link` is destructive). |
| `README.md` | modified | Added the projects/versions/components/links table. |

## Tools added

| Tool | Notes |
| --- | --- |
| `jira_list_projects` | optional `query` |
| `jira_get_project_versions` / `jira_create_version` | versions = releases / milestones; create resolves project **key → numeric id** (the API requires `projectId`) |
| `jira_get_project_components` | id, name, description, lead |
| `jira_list_link_types` | name + inward/outward phrasing |
| `jira_create_issue_link` / `jira_remove_issue_link` | by type name + two keys / by link id |
| `jira_link_to_epic` | uses the Epic Link custom field if the instance has one, else `parent` |
| `jira_create_remote_link` | URL + title |

## Solution & why

Two wrinkles drove the design:

1. **`jira_create_version` takes a project key, not an id.** The REST create API
   wants a numeric `projectId`, so the handler does a `GetProject(key)` first and
   resolves it. Users (and the model) think in keys like `KAN`, not `10000`.
2. **Epic linking differs by project style.** Team-managed projects use the
   standard `parent` field; company-managed classic projects use an *Epic Link*
   custom field. `jira_link_to_epic` reuses the P1 field resolver to detect an
   Epic Link field (`gh-epic-link`) and uses it when present, otherwise falls
   back to `parent`. The result reports which path (`via`) it took.

**Alternative considered:** expose raw `projectId` on `create_version` to skip the
extra lookup. Rejected — one extra GET is cheap and keeping the interface in keys
is far friendlier; the model rarely knows numeric project ids.

## Tests

`go test ./...`, `go vet ./...`, `go build ./...` all green. New: client tests for
all nine methods; tool tests for `epicLinkField`, the link-to-epic→parent
fallback (asserting the PUT body), and create-version project-id resolution.

## Live verification (KAN, real MCP stdio)

| Tool | Result |
| --- | --- |
| `jira_list_projects` | "My Kanban Space" (KAN, software) |
| `jira_list_link_types` | Blocks, Cloners, Duplicate, Relates |
| `jira_create_version {KAN, v0.1-mcp, release_date 2026-07-15}` | created id 10000 |
| `jira_get_project_versions` (fresh session) | shows **v0.1-mcp**, release 2026-07-15 |
| `jira_create_remote_link KAN-2 → modelcontextprotocol.io` | linked |
| `jira_create_issue_link Blocks (KAN-1 ↔ KAN-2)` | linked |

(As in P2/P5, a version read pipelined in the same batch as its create raced and
showed empty; the fresh-session read returned it — harness artifact.)

Artifacts left on KAN: version `v0.1-mcp`, a remote link and a Blocks link on
KAN-1/KAN-2. Remove when no longer needed.

## Known follow-ups

- `link_to_epic` for classic projects wants one live pass on a company-managed
  project (KAN is team-managed, so the `parent` path was exercised live; the Epic
  Link path is unit-tested).
- Version update/delete and component create — not in scope; candidates for later.
