// Package content converts between Markdown (what the LLM speaks) and the rich
// formats Atlassian Cloud requires: Jira ADF and Confluence storage XHTML.
//
// This is a pragmatic block-level subset — headings, paragraphs and bullet
// lists — not a full Markdown engine. Inline marks (bold/italic/links) are
// passed through as literal text. Higher fidelity is tracked as future work.
package content

import (
	"regexp"
	"strings"
)

type blockKind int

const (
	kindParagraph blockKind = iota
	kindHeading
	kindBullet
)

// block is one parsed Markdown block.
type block struct {
	kind  blockKind
	level int      // heading level (1-6)
	text  string   // paragraph / heading text
	items []string // bullet list items
}

var (
	reHeading = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	reBullet  = regexp.MustCompile(`^[-*]\s+(.*)$`)
)

// parseBlocks splits Markdown into a flat sequence of blocks.
func parseBlocks(md string) []block {
	var blocks []block
	var para []string
	var bullets []string

	flushPara := func() {
		if len(para) > 0 {
			blocks = append(blocks, block{kind: kindParagraph, text: strings.Join(para, " ")})
			para = nil
		}
	}
	flushBullets := func() {
		if len(bullets) > 0 {
			blocks = append(blocks, block{kind: kindBullet, items: bullets})
			bullets = nil
		}
	}

	for line := range strings.SplitSeq(md, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			flushPara()
			flushBullets()
		case reHeading.MatchString(trimmed):
			flushPara()
			flushBullets()
			m := reHeading.FindStringSubmatch(trimmed)
			blocks = append(blocks, block{kind: kindHeading, level: len(m[1]), text: m[2]})
		case reBullet.MatchString(trimmed):
			flushPara()
			m := reBullet.FindStringSubmatch(trimmed)
			bullets = append(bullets, m[1])
		default:
			flushBullets()
			para = append(para, trimmed)
		}
	}
	flushPara()
	flushBullets()
	return blocks
}
