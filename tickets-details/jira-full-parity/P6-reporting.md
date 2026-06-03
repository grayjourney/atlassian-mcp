# P6 — Reporting over sprint / board / milestone

## Goal

Let Claude answer "how's the sprint/board/release going?" — summarize a set of
issues into status/assignee/type breakdowns, done-vs-remaining counts, and a
story-point total/completed/remaining split.

## Changes summary

| File | Status | What |
| --- | --- | --- |
| `internal/tools/reports.go` | new | `aggregateIssues` (the pure roll-up), `toFloat`, `storyPointsID`, and the 3 report handlers. |
| `internal/tools/reports_test.go` | new | Table tests for the aggregation (with and without a story-point field). |
| `internal/tools/register.go` | modified | Registered `jira_board_report`, `jira_sprint_report`, `jira_version_report` (all `readOnly`). |
| `README.md` | modified | Added the reporting tool table. |

## Tools added

| Tool | Source of issues |
| --- | --- |
| `jira_board_report` | `GetBoardIssues(board_id)` |
| `jira_sprint_report` | `GetSprintIssues(sprint_id)` |
| `jira_version_report` | `Search(project = X AND fixVersion = "name")` |

Each returns: `total`, `done` / `in_progress` / `to_do` (by **status category**,
so it's workflow-agnostic), `by_status`, `by_assignee` (with `Unassigned`),
`by_type`, and — when the instance has a Story Points field — `story_points`
{`total`, `completed`, `remaining`}.

## Solution & why

P6 is composition, not new endpoints: a single pure `aggregateIssues([]Issue,
spID)` does the work, and the three tools just feed it from P4 (board/sprint) and
P1+search (version). Keeping the roll-up pure made it trivial to unit-test the
arithmetic exhaustively without touching the network. Done-ness keys off
`status.statusCategory.key` (`done` / `indeterminate` / `new`) rather than status
*names*, so it works on any workflow. Story points are summed only when the
instance actually has the field (`storyPointsID` returns "" otherwise and the
section is omitted) — the same graceful degradation as P1.

**Alternative considered:** call Jira's built-in Greenhopper sprint-report
endpoint. Rejected — it's undocumented/unstable, Scrum-only, and wouldn't cover
board or version reports; aggregating issues ourselves is uniform across all
three and reuses code we already trust.

**Scope note:** reports aggregate up to `reportMax` (100) issues in one pass; very
large boards would need pagination (tracked follow-up).

## Tests

`go test ./...`, `go vet ./...`, `go build ./...` all green. New: `aggregateIssues`
table tests asserting every breakdown and the story-point split, plus the
no-story-points-field case.

## Live verification (KAN, real MCP stdio)

| Tool | Result |
| --- | --- |
| `jira_board_report {board_id:1}` | total 2; In Progress 1 / To Do 1; all Task; Unassigned 2; (no story_points — instance has no SP field) |
| `jira_version_report {KAN, v0.1-mcp}` (before tagging) | total 0 — correct, nothing tagged yet |
| `jira_update_issue {KAN-2, fix_versions:[v0.1-mcp]}` | updated (also a live re-proof of P1's `fix_versions`) |
| `jira_version_report {KAN, v0.1-mcp}` (after) | total 1; To Do 1; Task 1 |

Sprint report couldn't be exercised (KAN is Kanban, no sprints); it shares the
exact aggregation path and is unit-tested.

Artifact: KAN-2 now carries fix version `v0.1-mcp`.
