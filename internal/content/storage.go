package content

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// MarkdownToStorage converts Markdown into Confluence storage-format XHTML.
func MarkdownToStorage(md string) string {
	var sb strings.Builder
	for _, b := range parseBlocks(md) {
		switch b.kind {
		case kindHeading:
			fmt.Fprintf(&sb, "<h%d>%s</h%d>", b.level, html.EscapeString(b.text), b.level)
		case kindParagraph:
			fmt.Fprintf(&sb, "<p>%s</p>", html.EscapeString(b.text))
		case kindBullet:
			sb.WriteString("<ul>")
			for _, it := range b.items {
				fmt.Fprintf(&sb, "<li>%s</li>", html.EscapeString(it))
			}
			sb.WriteString("</ul>")
		}
	}
	return sb.String()
}

var (
	reLiOpen     = regexp.MustCompile(`(?i)<li[^>]*>`)
	reBlockClose = regexp.MustCompile(`(?i)</(p|h[1-6])>`)
	reBr         = regexp.MustCompile(`(?i)<br\s*/?>`)
	reAnyTag     = regexp.MustCompile(`<[^>]+>`)
)

// StorageToText strips Confluence storage XHTML down to readable plain text,
// turning list items into "- " bullets and block boundaries into newlines.
func StorageToText(storage string) string {
	s := reLiOpen.ReplaceAllString(storage, "\n- ")
	s = reBlockClose.ReplaceAllString(s, "\n")
	s = reBr.ReplaceAllString(s, "\n")
	s = reAnyTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)

	var lines []string
	for ln := range strings.SplitSeq(s, "\n") {
		if ln = strings.TrimSpace(ln); ln != "" {
			lines = append(lines, ln)
		}
	}
	return strings.Join(lines, "\n")
}
