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

// inlineText concatenates all descendant text nodes of n.
func inlineText(n any) string {
	m, ok := n.(map[string]any)
	if !ok {
		return ""
	}
	if t, ok := m["text"].(string); ok {
		return t
	}
	var sb strings.Builder
	for _, c := range childContent(m) {
		sb.WriteString(inlineText(c))
	}
	return sb.String()
}

func childContent(m map[string]any) []any {
	c, _ := m["content"].([]any)
	return c
}
