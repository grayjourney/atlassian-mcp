package content

import "strings"

// MarkdownToADF converts Markdown into an Atlassian Document Format doc node
// (as map[string]any, ready to embed in a Jira request body).
func MarkdownToADF(md string) map[string]any {
	var nodes []any
	for _, b := range parseBlocks(md) {
		switch b.kind {
		case kindHeading:
			if n := textNode(b.text); n != nil {
				nodes = append(nodes, map[string]any{
					"type":    "heading",
					"attrs":   map[string]any{"level": b.level},
					"content": []any{n},
				})
			}
		case kindParagraph:
			nodes = append(nodes, paragraph(b.text))
		case kindBullet:
			var items []any
			for _, it := range b.items {
				items = append(items, map[string]any{
					"type":    "listItem",
					"content": []any{paragraph(it)},
				})
			}
			nodes = append(nodes, map[string]any{"type": "bulletList", "content": items})
		}
	}
	if nodes == nil {
		// ADF requires at least one content node; an empty paragraph is valid.
		nodes = []any{map[string]any{"type": "paragraph"}}
	}
	return map[string]any{"type": "doc", "version": 1, "content": nodes}
}

// paragraph builds a paragraph node; empty text yields an empty (but valid) one.
func paragraph(text string) map[string]any {
	if n := textNode(text); n != nil {
		return map[string]any{"type": "paragraph", "content": []any{n}}
	}
	return map[string]any{"type": "paragraph"}
}

// textNode returns an ADF text node, or nil for empty text (ADF forbids empty
// text nodes).
func textNode(text string) map[string]any {
	if text == "" {
		return nil
	}
	return map[string]any{"type": "text", "text": text}
}

// ADFToText renders an ADF doc (map[string]any) down to plain text, with block
// nodes separated by newlines. Accepts the decoded-JSON shape returned by the
// Jira API as well.
func ADFToText(adf any) string {
	m, ok := adf.(map[string]any)
	if !ok {
		return ""
	}
	var blocks []string
	for _, c := range childContent(m) {
		if s := renderBlock(c); s != "" {
			blocks = append(blocks, s)
		}
	}
	return strings.Join(blocks, "\n")
}

func renderBlock(n any) string {
	m, _ := n.(map[string]any)
	switch m["type"] {
	case "bulletList", "orderedList":
		var items []string
		for _, it := range childContent(m) {
			items = append(items, "- "+inlineText(it))
		}
		return strings.Join(items, "\n")
	default:
		return inlineText(m)
	}
}

// inlineText concatenates the visible text of n's inline descendants. It also
// surfaces nodes that carry their content in attrs rather than a text child:
// smart links (inlineCard/blockCard → URL), mentions, and emoji. Without this,
// smart-link URLs (e.g. a PR link pasted into a description) render as nothing.
func inlineText(n any) string {
	m, ok := n.(map[string]any)
	if !ok {
		return ""
	}
	switch m["type"] {
	case "text":
		return textWithLink(m)
	case "inlineCard", "blockCard", "embedCard":
		return cardURL(m)
	case "mention":
		return attrString(m, "text")
	case "emoji":
		if s := attrString(m, "text"); s != "" {
			return s
		}
		return attrString(m, "shortName")
	case "hardBreak":
		return "\n"
	}
	var sb strings.Builder
	for _, c := range childContent(m) {
		sb.WriteString(inlineText(c))
	}
	return sb.String()
}

// textWithLink returns a text node's content, appending the link target when the
// node carries a link mark whose href isn't already the visible text.
func textWithLink(m map[string]any) string {
	t, _ := m["text"].(string)
	href := linkHref(m)
	if href != "" && href != t {
		return t + " (" + href + ")"
	}
	return t
}

func linkHref(m map[string]any) string {
	marks, _ := m["marks"].([]any)
	for _, mk := range marks {
		mm, _ := mk.(map[string]any)
		if mm["type"] == "link" {
			if a, ok := mm["attrs"].(map[string]any); ok {
				if h, ok := a["href"].(string); ok {
					return h
				}
			}
		}
	}
	return ""
}

// cardURL pulls the URL out of a smart-link card node (attrs.url, or attrs.data.url).
func cardURL(m map[string]any) string {
	if u := attrString(m, "url"); u != "" {
		return u
	}
	if a, ok := m["attrs"].(map[string]any); ok {
		if d, ok := a["data"].(map[string]any); ok {
			if u, ok := d["url"].(string); ok {
				return u
			}
		}
	}
	return ""
}

func attrString(m map[string]any, key string) string {
	if a, ok := m["attrs"].(map[string]any); ok {
		if s, ok := a[key].(string); ok {
			return s
		}
	}
	return ""
}

func childContent(m map[string]any) []any {
	c, _ := m["content"].([]any)
	return c
}
