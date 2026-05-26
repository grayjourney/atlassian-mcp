package content

import (
	"encoding/json"
	"strings"
	"testing"
)

// node walks an ADF doc (as map[string]any) and returns the content slice.
func docContent(t *testing.T, adf map[string]any) []any {
	t.Helper()
	if adf["type"] != "doc" {
		t.Fatalf("root type = %v, want doc", adf["type"])
	}
	c, ok := adf["content"].([]any)
	if !ok {
		t.Fatalf("doc.content not a slice: %T", adf["content"])
	}
	return c
}

func firstText(node any) string {
	m, _ := node.(map[string]any)
	content, _ := m["content"].([]any)
	if len(content) == 0 {
		return ""
	}
	t, _ := content[0].(map[string]any)
	s, _ := t["text"].(string)
	return s
}

func TestMarkdownToADFHeadingAndParagraph(t *testing.T) {
	adf := MarkdownToADF("# Title\n\nHello world")
	nodes := docContent(t, adf)
	if len(nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(nodes))
	}
	h := nodes[0].(map[string]any)
	if h["type"] != "heading" {
		t.Errorf("node0 type = %v, want heading", h["type"])
	}
	if attrs := h["attrs"].(map[string]any); attrs["level"].(int) != 1 {
		t.Errorf("heading level = %v, want 1", attrs["level"])
	}
	if firstText(h) != "Title" {
		t.Errorf("heading text = %q", firstText(h))
	}
	p := nodes[1].(map[string]any)
	if p["type"] != "paragraph" || firstText(p) != "Hello world" {
		t.Errorf("paragraph = %+v", p)
	}
}

func TestMarkdownToADFBulletList(t *testing.T) {
	adf := MarkdownToADF("- one\n- two")
	nodes := docContent(t, adf)
	if len(nodes) != 1 {
		t.Fatalf("got %d nodes, want 1 bulletList", len(nodes))
	}
	list := nodes[0].(map[string]any)
	if list["type"] != "bulletList" {
		t.Fatalf("type = %v, want bulletList", list["type"])
	}
	items := list["content"].([]any)
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	li := items[0].(map[string]any)
	if li["type"] != "listItem" {
		t.Errorf("item type = %v", li["type"])
	}
	para := li["content"].([]any)[0]
	if firstText(para) != "one" {
		t.Errorf("first item text = %q", firstText(para))
	}
}

func TestMarkdownToADFIsValidJSON(t *testing.T) {
	adf := MarkdownToADF("# H\n\npara")
	raw, err := json.Marshal(adf)
	if err != nil {
		t.Fatalf("marshal ADF: %v", err)
	}
	if adf["version"].(int) != 1 {
		t.Errorf("version = %v, want 1", adf["version"])
	}
	if !strings.Contains(string(raw), `"type":"doc"`) {
		t.Errorf("marshaled ADF missing doc type: %s", raw)
	}
}

func TestADFToText(t *testing.T) {
	adf := map[string]any{
		"type":    "doc",
		"version": 1,
		"content": []any{
			map[string]any{"type": "heading", "attrs": map[string]any{"level": 1},
				"content": []any{map[string]any{"type": "text", "text": "Title"}}},
			map[string]any{"type": "paragraph",
				"content": []any{map[string]any{"type": "text", "text": "Hello "},
					map[string]any{"type": "text", "text": "world"}}},
		},
	}
	got := ADFToText(adf)
	want := "Title\nHello world"
	if got != want {
		t.Errorf("ADFToText = %q, want %q", got, want)
	}
}

func TestADFRoundTrip(t *testing.T) {
	adf := MarkdownToADF("# Heading\n\nBody text here")
	got := ADFToText(adf)
	if !strings.Contains(got, "Heading") || !strings.Contains(got, "Body text here") {
		t.Errorf("round trip lost content: %q", got)
	}
}

func TestMarkdownToStorage(t *testing.T) {
	tests := []struct {
		name string
		md   string
		want string
	}{
		{"heading", "# Title", "<h1>Title</h1>"},
		{"paragraph", "Hello world", "<p>Hello world</p>"},
		{"escapes html", "a < b & c", "<p>a &lt; b &amp; c</p>"},
		{"bullets", "- one\n- two", "<ul><li>one</li><li>two</li></ul>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MarkdownToStorage(tt.md); got != tt.want {
				t.Errorf("MarkdownToStorage(%q) = %q, want %q", tt.md, got, tt.want)
			}
		})
	}
}

func TestStorageToText(t *testing.T) {
	tests := []struct {
		name    string
		storage string
		want    string
	}{
		{"paragraph", "<p>Hello world</p>", "Hello world"},
		{"strip inline", "<p>Hello <strong>world</strong></p>", "Hello world"},
		{"entities", "<p>a &lt; b &amp; c</p>", "a < b & c"},
		{"list items", "<ul><li>one</li><li>two</li></ul>", "- one\n- two"},
		{"multiple blocks", "<h1>T</h1><p>body</p>", "T\nbody"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StorageToText(tt.storage); got != tt.want {
				t.Errorf("StorageToText(%q) = %q, want %q", tt.storage, got, tt.want)
			}
		})
	}
}
