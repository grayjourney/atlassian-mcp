# Jira full-parity expansion

Grow the Go MCP's Jira surface from the 5-tool MVP toward parity with the Python
[`mcp-atlassian`](https://github.com/sooperset/mcp-atlassian) project (~50 Jira
tools), so Claude can perform virtually any Jira action a user asks for on
Atlassian Cloud. Built **phase by phase**, each phase strict-TDD and
live-verified against the `KAN` board on `grayjourney.atlassian.net`.

| Phase | Theme | Status |
| --- | --- | --- |
| [P1](P1-issue-fields.md) | Issue power-ups + field resolution | ✅ done, live-verified |
| [P2](P2-comments-worklog.md) | Comments, worklog, dates, watchers, users | ✅ done, live-verified |
| [P3](P3-attachments.md) | Attachments & content (download) | ✅ done, live-verified |
| [P4](P4-agile.md) | Agile: boards & sprints (current sprint) | ✅ done, live-verified |
| [P5](P5-projects-links.md) | Projects, versions (milestones), components, links | ✅ done, live-verified |
| [P6](P6-reporting.md) | Reporting over sprint / board / milestone | ✅ done, live-verified |
| [P7](P7-servicedesk-dev.md) | Service Desk + Development info (ProForma deferred — needs OAuth) | ✅ done, live-verified |

**Suggested branch:** `feature/jira-full-parity` (not created — yours to trigger).
