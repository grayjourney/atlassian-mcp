# P4 — Agile: boards & sprints

## Goal

Cover the user's "check the current sprint" need and the agile surface around it:
list boards, read board/sprint issues, and create/update/populate sprints — via
the separate `/rest/agile/1.0` API.

## Changes summary

| File | Status | What |
| --- | --- | --- |
| `internal/atlassian/agile.go` | new | `Board`, `Sprint` types; `GetBoards`, `GetSprints`, `GetBoardIssues`, `GetSprintIssues`, `CreateSprint`, `UpdateSprint`, `MoveIssuesToSprint`. All on `/rest/agile/1.0`. |
| `internal/atlassian/agile_test.go` | new | httptest coverage for each (paths, query, bodies). |
| `internal/tools/agile.go` | new | 8 tool handlers + `boardMeta`/`sprintMeta`/`issueList`/`clampLimit` helpers. |
| `internal/tools/agile_test.go` | new | board-list shaping + create-sprint body-building tests. |
| `internal/tools/register.go` | modified | Registered the 8 agile tools. |
| `README.md` | modified | Added the agile tool table. |

## Tools added

| Tool | Notes |
| --- | --- |
| `jira_list_boards` | optional `project` filter |
| `jira_get_board_issues` | optional JQL; compact issue projection |
| `jira_list_sprints` | optional `state` (active/future/closed) |
| `jira_get_active_sprint` | convenience over `state=active` — the "current sprint" |
| `jira_get_sprint_issues` | optional JQL |
| `jira_create_sprint` / `jira_update_sprint` | create; partial update of name/goal/state/dates (state drives start/close) |
| `jira_move_issues_to_sprint` | move issues by key |

## Solution & why

The agile API lives at `/rest/agile/1.0` (not `/rest/api/3`) but on the same
host with the same basic auth, so the existing `restClient` handles it unchanged
— the new methods just pass the agile paths. Board/sprint ids are integers, so
the inputs use `int` (the SDK validates the JSON schema). `jira_get_active_sprint`
is deliberately a thin wrapper over `GetSprints(state=active)` because "what's the
current sprint" is the single most common ask and shouldn't require the model to
know the state vocabulary. Sprint `update` uses the agile API's partial `POST`
(setting `state` to `active`/`closed` is how you start/complete a sprint).

**Alternative considered:** fold board issues into the existing `jira_search`.
Rejected — board/sprint issue lists honor the board's own filter/ranking and live
under different endpoints; a dedicated tool matches how users think ("issues on
my board / in this sprint").

## Tests

`go test ./...`, `go vet ./...`, `go build ./...` all green. New: client tests for
all seven methods, plus tool-layer board-list and create-sprint tests.

## Live verification (KAN board, real MCP stdio)

KAN is a **team-managed Kanban** board (no sprints), which exercises both the
working board path and the Scrum-only error path:

| Tool | Result |
| --- | --- |
| `jira_list_boards {project:"KAN"}` | board id **1**, "KAN board", type `simple` |
| `jira_get_board_issues {board_id:1}` | KAN-1 + KAN-2 with full detail |
| `jira_list_sprints {board_id:1}` | clean error: *"The board does not support sprints"* (Kanban) |
| `jira_get_active_sprint {board_id:1}` | same clean error |

The sprint create/update/move and sprint-issue paths can't be exercised on a
Kanban board; they're covered by unit tests against the mock API and will work on
any Scrum board. To live-test them, point at a Scrum board on a project with the
Sprints feature enabled.

## Known follow-ups

- Backlog endpoints (move issues to backlog, rank) — not in scope; candidate for
  a later enhancement.
- Sprint create/update against a real Scrum board still wants one live pass when
  such a board exists on the instance.
