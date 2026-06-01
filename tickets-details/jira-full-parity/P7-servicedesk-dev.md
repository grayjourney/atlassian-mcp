# P7 — Service Desk, Development info (+ ProForma decision)

## Goal

Close out "full parity": Jira Service Management (service desks, queues, queue
issues), development information (branches/commits/PRs from connected VCS), and
ProForma forms.

## Outcome

Service Desk (3 tools) and Development info (1 tool) shipped and verified.
**ProForma forms were deliberately not shipped** — see the decision below.

## Changes summary

| File | Status | What |
| --- | --- | --- |
| `internal/atlassian/servicedesk.go` | new | `ServiceDesk`, `Queue` types; `GetServiceDesks`, `GetQueues`, `GetQueueIssues` (`/rest/servicedeskapi`), and `GetDevelopmentSummary` (`/rest/dev-status/latest/issue/summary`). |
| `internal/atlassian/servicedesk_test.go` | new | httptest coverage for all four. |
| `internal/tools/servicedesk.go` | new | `jira_list_service_desks`, `jira_list_queues`, `jira_get_queue_issues`, `jira_get_development_info`. |
| `internal/tools/register.go` | modified | Registered the 4 tools (all `readOnly`). |
| `README.md` | modified | Added the service-management/development tool table; refreshed the scope section (incl. the ProForma/OAuth note); fixed stale notes (assignee-by-email, tool count). |

## Tools added

| Tool | Endpoint |
| --- | --- |
| `jira_list_service_desks` | `GET /rest/servicedeskapi/servicedesk` |
| `jira_list_queues` | `GET /rest/servicedeskapi/servicedesk/{id}/queue` |
| `jira_get_queue_issues` | `GET …/queue/{queueId}/issue` |
| `jira_get_development_info` | resolve key→numeric id, then `GET /rest/dev-status/latest/issue/summary?issueId=` |

`jira_get_development_info` first fetches the issue to get its **numeric id** (the
dev-status API doesn't accept keys), then returns the development panel summary
(branch / commit / pull-request / build / deployment counts).

## The ProForma decision (why it's not here)

The Python project's ProForma tools call
`https://api.atlassian.com/jira/forms/cloud/{cloudId}/issue/{key}/form` and its
own client **only sends `Authorization: Bearer`** (OAuth 3LO or PAT). That gateway
host does not accept site **basic auth** (email + API token), and it additionally
needs a cloud-id lookup. Our entire server is basic-auth-against-the-site by
design (MVP scope). So ProForma can't be made to work without first implementing
OAuth 2.0 — which is already the top item under "future work".

Shipping non-functional `jira_*_proforma_form` tools would mislead the model into
trying an action that always 401s. The honest choice is to **defer ProForma and
document it** as gated behind OAuth, rather than register dead tools.

## Tests

`go test ./...`, `go vet ./...`, `go build ./...` all green. New client tests for
the three service-desk calls and the dev summary.

## Live verification (KAN, real MCP stdio)

| Tool | Result |
| --- | --- |
| `jira_get_development_info {KAN-1}` | returned the dev summary — all counts 0 (no VCS connected to this site), proving the call + numeric-id resolution work |
| `jira_list_service_desks` | API `403 Forbidden` surfaced cleanly — this free instance has no Jira Service Management, so there's nothing to authorize |

Service-desk tools are correct but can't be data-verified without a JSM-licensed
instance; the request paths and parsing are unit-tested. Development info is
verified end-to-end (empty is the correct answer here).

## Known follow-ups

- ProForma forms — revisit once OAuth 2.0 lands.
- Service-desk tools want one live pass on a JSM-enabled instance.
- The dev-status `summary` is returned raw; a future tidy could flatten it to a
  compact `{branches, commits, pull_requests}` shape like the other tools.
