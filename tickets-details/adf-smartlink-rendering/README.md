# Bug — smart links & mentions dropped from Jira text (ADF → text)

## Overview

`jira_get_issue` (and any tool that renders a Jira rich-text field) silently
dropped **smart links**, leaving Claude blind to URLs pasted into descriptions.
Reported against **CUR-29265**, whose description contains a Bitbucket PR link
(`…/pull-requests/1251/diff`) that never appeared in the tool output — Claude
concluded "no PR is linked," which was wrong.

## Investigation

Compared the raw issue (Jira XML export) against the tool output:

- XML description clearly contains the PR link, a Google-doc link, a
  `feature/abu-engine` branch link, and several `@mentions`.
- Tool output rendered: `"PR to be merged -   c/o  "` — the URL and the mention
  between them were gone. The planning-ticket link (`…/browse/CUR-29264`)
  *did* survive.

The survivors had one thing in common: their visible **text was the URL itself**
(a plain link). The casualties were all **smart links** (`title="smart-link"` in
the HTML) and **mentions**.

## Root cause

`internal/content/adf.go`, `inlineText()` (pre-fix): it returned a node's `text`
field and otherwise recursed into `content`. In Atlassian Document Format:

- A **smart link** is an `inlineCard` node — its URL is in `attrs.url`, and it has
  **no `text` child and no `content`**. So `inlineText` returned `""`.
- A **mention** is a `mention` node with the label in `attrs.text` — also dropped.
- A **link mark** on a text node carries the href in the mark's `attrs.href`,
  which was ignored (fine when label == URL, lossy otherwise).

Plain links survived only because their label text happened to equal the URL.

## Steps to reproduce

1. On any Jira issue, paste a URL so it renders as a smart link (or @mention someone) in the description.
2. `jira_get_issue {issue_key}` → the smart link / mention is absent from `description`.

Captured as a unit test (`TestADFToTextRendersSmartLinksMentionsAndLinkMarks`)
that fails before the fix with `got: "PR to be merged -  c/o \ndesign doc"`.

## The fix

`inlineText()` now switches on node type and surfaces the attr-borne content:

| Node | Now renders |
| --- | --- |
| `inlineCard` / `blockCard` / `embedCard` | `attrs.url` (or `attrs.data.url`) |
| `mention` | `attrs.text` (e.g. `@Kevin Dang`) |
| `emoji` | `attrs.text` / `attrs.shortName` |
| `text` with a `link` mark | `label (href)` when the href differs from the label |
| `hardBreak` | newline |

Helpers added: `textWithLink`, `linkHref`, `cardURL`, `attrString`.

## Manual test guide

```bash
go test ./internal/content/   # TestADFToTextRendersSmartLinksMentionsAndLinkMarks passes
go test ./... && go vet ./...  # all green
```

Live (verified against the real ticket):

```bash
JIRA_URL=https://matchmove.atlassian.net JIRA_USERNAME=… JIRA_API_TOKEN=… \
  atlassian-mcp   # drive jira_get_issue {"issue_key":"CUR-29265"}
```

Expected (post-fix) — the description now includes:
`https://bitbucket.org/matchmove/platform-svc-card-tokenization/pull-requests/1251/diff … @Kevin Dang`,
plus the Google-doc and branch links and the cc mentions.

## Ships in

Patch release **v0.2.1** (`cmd/atlassian-mcp/main.go`, `plugin/.claude-plugin/plugin.json`,
`.claude-plugin/marketplace.json` all bumped together).
