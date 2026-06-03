# P2 — Comments, worklog, dates, watchers, users

## Goal

Let Claude hold a conversation on an issue (comments), log and read time
(worklog), see every date on an issue, manage watchers, and resolve people to
account ids — all the per-issue "activity" the MVP lacked.

## Changes summary

| File | Status | What |
| --- | --- | --- |
| `internal/atlassian/activity.go` | new | Client methods: `GetComments`/`AddComment`/`EditComment`, `GetWorklogs`/`AddWorklog`, `GetWatchers`/`AddWatcher`/`RemoveWatcher`, `GetUser`. Types `IssueComment`, `Worklog`, `Watchers`. |
| `internal/atlassian/activity_test.go` | new | httptest coverage for each (paths, methods, bodies, query params). |
| `internal/tools/activity.go` | new | 10 tool handlers + input structs. |
| `internal/tools/activity_test.go` | new | ADF-shaping test for comments and a date-collection test for `jira_get_issue_dates`; shared `newToolServer`/`resultText` helpers. |
| `internal/tools/register.go` | modified | Registered the 10 P2 tools (reads `readOnly`, writes `write`). |
| `README.md` | modified | Added the "comments, time, people" tool table. |

## Tools added

| Tool | Notes |
| --- | --- |
| `jira_add_comment` / `jira_list_comments` / `jira_edit_comment` | Markdown ↔ ADF; reads render to text |
| `jira_add_worklog` / `jira_get_worklog` | `time_spent` like "2h 30m"; optional comment + ISO `started`; read reports `total_seconds` |
| `jira_get_issue_dates` | discovers all date/datetime fields (incl. custom) via the P1 field resolver and returns them by name |
| `jira_list_watchers` / `jira_add_watcher` / `jira_remove_watcher` | add/remove accept email/name/id (resolved via `/user/search`) |
| `jira_get_user` | search users by email/name → account id |

## Solution & why

These all reuse the P1 plumbing: comments/worklog comments go through
`content.MarkdownToADF`, watcher add/remove and the assignee-style inputs go
through `resolveAccountID`, and `jira_get_issue_dates` reuses `jiraFields` to find
custom date fields and request exactly those. `AddWatcher` sends the bare account
id as the JSON body (a quoted string) — the one Jira endpoint that doesn't take
an object — which the shared `restClient.do` handles by marshalling the string.

The Jira `Comment` type was named **`IssueComment`** to avoid colliding with the
existing Confluence `Comment`.

**Alternative considered:** a single `jira_manage_watcher` with an add/remove
flag — rejected; two named tools read more clearly to the model and match the
read/write annotation split.

## Tests

`go test ./...`, `go vet ./...`, `go build ./...` all green. New: client-method
tests for every endpoint, a comment ADF-shaping test, and a date-collection test
asserting `Due date`/`Created` surface by name.

## Live verification (KAN-2, real MCP stdio)

| Tool | Result |
| --- | --- |
| `jira_get_user {user: <email>}` | resolved to account id `64036ffa314f50881382aace` |
| `jira_add_comment` | created comment `10000` |
| `jira_list_comments` (fresh session) | returns "P2 test comment via MCP" |
| `jira_add_worklog 1h 30m` | created worklog `10000` |
| `jira_get_worklog` (fresh session) | 1 entry, `total_seconds: 5400`, comment "investigating" |
| `jira_get_issue_dates` | Created, Updated, **Due date**, **Start date** (custom), Status Category Changed |
| `jira_list_watchers` | 1 watcher (Gray Tran), `is_watching: true` |

> Note: when several tool calls are pipelined in one MCP session the server runs
> them **concurrently**, so a read issued in the same batch as its write can race
> ahead and miss it. Reads in a subsequent session returned the written data
> correctly — a test-harness artifact, not a tool bug.

Artifacts on KAN-2: one comment + one worklog. Remove when no longer needed.
