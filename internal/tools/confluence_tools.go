package tools

import (
	"context"

	"github.com/grayjourney/atlassian-mcp/internal/content"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- confluence_search ---

type confluenceSearchInput struct {
	CQL   string `json:"cql" jsonschema:"CQL query, e.g. \"type = page AND space = DOCS AND text ~ 'onboarding'\""`
	Limit int    `json:"limit,omitempty" jsonschema:"max results, default 10"`
}

func (s *Server) confluenceSearch(ctx context.Context, _ *mcp.CallToolRequest, in confluenceSearchInput) (*mcp.CallToolResult, any, error) {
	cc, _, err := s.confluence()
	if err != nil {
		return nil, nil, err
	}
	res, err := cc.Search(ctx, in.CQL, in.Limit)
	if err != nil {
		return nil, nil, err
	}
	results := make([]map[string]any, 0, len(res.Results))
	for _, r := range res.Results {
		results = append(results, map[string]any{
			"id":      r.Content.ID,
			"type":    r.Content.Type,
			"title":   firstNonEmpty(r.Content.Title, r.Title),
			"excerpt": content.StorageToText(r.Excerpt),
		})
	}
	return textResult(map[string]any{"results": results, "count": len(results)})
}

// --- confluence_get_page ---

type confluenceGetPageInput struct {
	PageID string `json:"page_id" jsonschema:"the Confluence page (content) id"`
}

func (s *Server) confluenceGetPage(ctx context.Context, _ *mcp.CallToolRequest, in confluenceGetPageInput) (*mcp.CallToolResult, any, error) {
	cc, cfg, err := s.confluence()
	if err != nil {
		return nil, nil, err
	}
	page, err := cc.GetPage(ctx, in.PageID, "body.storage,version,space")
	if err != nil {
		return nil, nil, err
	}
	out := map[string]any{
		"id":      page.ID,
		"title":   page.Title,
		"space":   page.Space.Key,
		"version": page.Version.Number,
		"text":    content.StorageToText(page.Body.Storage.Value),
	}
	if webui := page.Links["webui"]; webui != "" {
		out["url"] = pageURL(cfg.ConfluenceURL, webui)
	}
	return textResult(out)
}

// --- confluence_create_page ---

type confluenceCreatePageInput struct {
	SpaceKey string `json:"space_key" jsonschema:"the space key the page belongs to, e.g. DOCS"`
	Title    string `json:"title" jsonschema:"page title"`
	Content  string `json:"content" jsonschema:"page body in Markdown"`
	ParentID string `json:"parent_id,omitempty" jsonschema:"optional parent page id to nest under"`
}

func (s *Server) confluenceCreatePage(ctx context.Context, _ *mcp.CallToolRequest, in confluenceCreatePageInput) (*mcp.CallToolResult, any, error) {
	cc, cfg, err := s.confluence()
	if err != nil {
		return nil, nil, err
	}
	storage := content.MarkdownToStorage(in.Content)
	page, err := cc.CreatePage(ctx, in.SpaceKey, in.Title, storage, in.ParentID)
	if err != nil {
		return nil, nil, err
	}
	out := map[string]any{"id": page.ID, "title": page.Title}
	if webui := page.Links["webui"]; webui != "" {
		out["url"] = pageURL(cfg.ConfluenceURL, webui)
	}
	return textResult(out)
}

// --- confluence_update_page ---

type confluenceUpdatePageInput struct {
	PageID  string `json:"page_id" jsonschema:"the Confluence page (content) id"`
	Title   string `json:"title,omitempty" jsonschema:"new title; keeps the current title if omitted"`
	Content string `json:"content" jsonschema:"new page body in Markdown (replaces existing body)"`
}

func (s *Server) confluenceUpdatePage(ctx context.Context, _ *mcp.CallToolRequest, in confluenceUpdatePageInput) (*mcp.CallToolResult, any, error) {
	cc, _, err := s.confluence()
	if err != nil {
		return nil, nil, err
	}
	// Confluence needs the current version number to write the next one.
	current, err := cc.GetPage(ctx, in.PageID, "version")
	if err != nil {
		return nil, nil, err
	}
	title := in.Title
	if title == "" {
		title = current.Title
	}
	storage := content.MarkdownToStorage(in.Content)
	page, err := cc.UpdatePage(ctx, in.PageID, title, storage, current.Version.Number)
	if err != nil {
		return nil, nil, err
	}
	return textResult(map[string]any{
		"id":      page.ID,
		"title":   page.Title,
		"version": page.Version.Number,
	})
}

// --- confluence_add_comment ---

type confluenceAddCommentInput struct {
	PageID  string `json:"page_id" jsonschema:"the Confluence page (content) id to comment on"`
	Content string `json:"content" jsonschema:"comment body in Markdown"`
}

func (s *Server) confluenceAddComment(ctx context.Context, _ *mcp.CallToolRequest, in confluenceAddCommentInput) (*mcp.CallToolResult, any, error) {
	cc, _, err := s.confluence()
	if err != nil {
		return nil, nil, err
	}
	storage := content.MarkdownToStorage(in.Content)
	comment, err := cc.AddComment(ctx, in.PageID, storage)
	if err != nil {
		return nil, nil, err
	}
	return textResult(map[string]any{"id": comment.ID, "page_id": in.PageID})
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// pageURL joins the Confluence base (which includes /wiki) with a webui link.
func pageURL(base, webui string) string {
	base = trimTrailingSlash(base)
	// _links.webui is relative to the wiki root, e.g. "/spaces/DS/pages/123".
	// The base already ends in /wiki, so strip a duplicate /wiki if present.
	return base + webui
}

func trimTrailingSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
