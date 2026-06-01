# P1 — Issue power-ups + field resolution

## Goal

Make `jira_create_issue` / `jira_update_issue` able to set the fields users
actually ask for — **due date, story points, assignee (by email), labels,
components, fix versions, parent/epic, priority, and any custom field by name** —
and make `jira_get_issue` show them back. Add field discovery so "set the
Severity field to High" works without anyone knowing `customfield_10050`.

## Changes summary

| File | Status | What |
| --- | --- | --- |
| `internal/atlassian/fields.go` | new | Client methods: `GetFields` (`GET /rest/api/3/field`), `GetChangelog`, `DeleteIssue`, `SearchUsers` (`/user/search`), `GetFieldOptions` (context → options). New types `Field`, `FieldSchema`, `ChangelogPage`, `User`, `FieldOption`. |
| `internal/atlassian/fields_test.go` | new | httptest coverage for each, incl. a `newMultiServer` helper for the two-step options call. |
| `internal/tools/fields.go` | new | `fieldResolver` (name↔id, Story Points / Sprint special-casing), `formatFieldValue` (shape by schema type), `nameObjects`, `resolveAccountID`, `applyIssueFields`, and the 4 new tool handlers. |
| `internal/tools/fields_test.go` | new | Resolver/format unit tests + an end-to-end `jiraCreateIssue` shaping test against a fake Jira. |
| `internal/tools/jira_tools.go` | modified | Create/update inputs gained the typed fields + `fields` (by-name JSON); `jira_get_issue` now fetches a rich default field set and surfaces story points / sprints. |
| `internal/tools/common.go` | modified | `flattenIssue` surfaces created, due date, reporter, resolution, parent, labels, components, fix versions; added `objectNames`. |
| `internal/tools/register.go` | modified | Registered `jira_delete_issue` (destructive), `jira_get_changelog`, `jira_list_fields`, `jira_get_field_options`; added a `destructive` annotation and `ptr` helper. |
| `README.md` | modified | Tool tables updated for the expanded Jira surface. |

## Solution & why

**Field resolution is the keystone.** Custom field ids differ per Jira instance,
so the only robust way to "set Story Points / Severity / any custom field" is to
fetch `GET /rest/api/3/field`, build a name→id map, and special-case the
greenfield plugin keys (`gh-story-points`, `gh-sprint`). `applyIssueFields` only
pays for that fetch when story points or by-name fields are actually used; the
plain typed fields (due date, labels, components, fix versions, parent, priority)
are shaped directly with no extra call. Assignee is resolved through
`/user/search` so users can pass an **email** instead of an opaque account id.

`formatFieldValue` shapes a raw value to Jira's expected JSON by the field's
schema type (`option`→`{value}`, `user`→`{accountId}`, arrays by element type,
numbers/dates pass through), mirroring the Python project's
`_format_field_value_for_write` dispatch but trimmed to the Cloud cases we hit.

**Alternatives considered:** (1) require raw `customfield_xxxxx` ids from the
caller — rejected, it pushes instance-specific knowledge onto the LLM/user and is
exactly what the Python project avoids; (2) cache the field map on the server —
deferred (noted below) to keep P1 correct and simple before optimizing.

## Tests

`go test ./...`, `go vet ./...`, `go build ./...` all green. New tests:
field resolver (name/id/miss), Story Points discovery, `formatFieldValue` matrix,
`nameObjects`, the client methods (incl. the two-step options call), and a
full `jiraCreateIssue` shaping test asserting the outgoing request body.

## Live verification (KAN board, real MCP stdio)

The `KAN` project is a basic team-managed Kanban — **no** Story Points or Sprint
field — which conveniently exercises both paths:

| Check | Result |
| --- | --- |
| `jira_list_fields {custom_only:true}` | 10 custom fields listed (Start date, Team, …); no Story Points/Sprint |
| `jira_create_issue` with `due_date`, `labels`, and `fields:{"Start date":"2026-06-10"}` | created **KAN-2**; "Start date" resolved by name and accepted |
| `jira_get_issue KAN-2` | `due_date: 2026-06-20`, `labels: [mcp, p1-test]`, reporter/created/status surfaced |
| `jira_get_changelog KAN-1` | real history (status + summary/labels changes) |
| `jira_create_issue` with `story_points` | clean error: *"could not find a Story Points field on this Jira instance"* |

Artifacts left on the board: **KAN-2** (P1 test issue). Delete when no longer
needed.

## Known follow-ups

- **Field-map cache** on the `Server` (per Jira URL) to avoid repeated
  `GET /field` on create/update/get_issue; add a refresh path via `jira_list_fields`.
- Epic linking on classic (company-managed) projects may need the resolved *Epic
  Link* custom field rather than `parent`; revisit in P5 (links).
- Higher-fidelity Markdown↔ADF (inline bold/links/code) — tracked for a later phase.
