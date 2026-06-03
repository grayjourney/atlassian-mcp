# P3 — Attachments & content

## Goal

Let Claude see what's attached to an issue, read text attachments inline, and
download any attachment to disk — covering the user's "get content / download
attachment" need.

## Changes summary

| File | Status | What |
| --- | --- | --- |
| `internal/atlassian/attachments.go` | new | `Attachment` type, `GetAttachments` (issue `fields=attachment`), `DownloadAttachment` (bytes + content-type). |
| `internal/atlassian/client.go` | modified | Added `restClient.getBytes` — GET an absolute URL with auth, return raw body + Content-Type (binary downloads). |
| `internal/atlassian/attachments_test.go` | new | Metadata fetch + a download asserting bytes, content-type, and that auth is sent. |
| `internal/tools/attachments.go` | new | `jira_list_attachments`, `jira_download_attachment`, `jira_read_attachment`; helpers `resolveAttachment`, `isTextual`, `findAttachment`. |
| `internal/tools/attachments_test.go` | new | resolve/isTextual unit tables + a full download test writing to a temp dir. |
| `internal/tools/register.go` | modified | Registered the 3 attachment tools (all `readOnly`). |
| `README.md` | modified | Added the attachments tool table. |

## Tools added

| Tool | Notes |
| --- | --- |
| `jira_list_attachments` | filename, size, mime type, author, created |
| `jira_read_attachment` | text only (`text/*`, json/xml/csv/yaml/html/markdown); capped at 100 KiB, reports `truncated` |
| `jira_download_attachment` | saves to `dir` (default a temp dir) at mode `0600`, returns absolute path |

Both content tools accept the attachment by **id or filename** (case-insensitive)
via `findAttachment`, which returns a helpful "available: …" error on a miss.

## Solution & why

The attachment `content` field is an **absolute** URL (it can redirect to a
signed media host), so the existing `restClient.do` (which joins a path onto the
base URL and decodes JSON) didn't fit. Added a small `getBytes` that requests the
full URL, sends the auth header, and returns raw bytes — Go's client drops the
auth header on the cross-host redirect to the pre-signed media URL, which is
exactly right.

`jira_read_attachment` vs `jira_download_attachment` is a deliberate split: text
goes inline for the model to reason over; binary goes to disk and we hand back a
path (returning megabytes of base64 inline would blow the context). Downloaded
filenames are run through `filepath.Base` to prevent path traversal from a
crafted attachment name.

**Alternative considered:** one `jira_get_attachment` that auto-switches between
inline and disk — rejected; explicit tools make the model's intent (and the
read-only-ness) obvious, and avoid surprising multi-MB inline payloads.

## Tests

`go test ./...`, `go vet ./...`, `go build ./...` all green. New: client download
test (bytes/type/auth), `resolveAttachment` + `isTextual` tables, and an
end-to-end download test that reads the saved file back.

## Live verification (KAN-2, real MCP stdio)

Uploaded `notes.txt` to KAN-2 (via the REST API), then:

| Tool | Result |
| --- | --- |
| `jira_list_attachments` | `notes.txt`, id 10000, 28 bytes, text/plain |
| `jira_read_attachment notes.txt` | returned the file's text inline, `truncated: false` |
| `jira_download_attachment notes.txt` | saved `/tmp/p3-dl/notes.txt` (0600), content matched |
| `jira_read_attachment nope.bin` | clean error: *"not found on KAN-2; available: notes.txt"* |

Artifact on KAN-2: the `notes.txt` attachment. Remove when no longer needed.

## Known follow-ups

- **Upload** (`jira_add_attachment`) — not in P3; the reference has it. Candidate
  for a later phase or P7.
- Image attachments for vision (base64 inline) — deferred; `jira_download_attachment`
  handles images to disk today.
