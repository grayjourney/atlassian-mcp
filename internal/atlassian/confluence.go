package atlassian

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// ConfluenceClient wraps the Confluence Cloud REST v1 API (/wiki/rest/api).
//
// MVP uses the v1 content endpoints, which still serve these operations. The
// Confluence Cloud v2 API migration is tracked as future work.
type ConfluenceClient struct {
	rc *restClient
}

// NewConfluenceClient builds a client for baseURL, which should already include
// the /wiki context path (e.g. https://acme.atlassian.net/wiki).
func NewConfluenceClient(baseURL, username, token string, hc *http.Client) *ConfluenceClient {
	return &ConfluenceClient{rc: newRESTClient(baseURL, username, token, hc)}
}

// Page models the subset of a Confluence content object the tools surface.
type Page struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Space  struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	} `json:"space"`
	Version struct {
		Number int `json:"number"`
	} `json:"version"`
	Body struct {
		Storage struct {
			Value string `json:"value"`
		} `json:"storage"`
	} `json:"body"`
	Links map[string]string `json:"_links"`
}

// SearchResult is the response of a CQL search.
type ConfluenceSearchResult struct {
	Results []struct {
		Content struct {
			ID    string `json:"id"`
			Type  string `json:"type"`
			Title string `json:"title"`
		} `json:"content"`
		Title   string `json:"title"`
		Excerpt string `json:"excerpt"`
		URL     string `json:"url"`
	} `json:"results"`
	Size int `json:"size"`
}

// Search runs a CQL query.
func (c *ConfluenceClient) Search(ctx context.Context, cql string, limit int) (*ConfluenceSearchResult, error) {
	if limit <= 0 {
		limit = 10
	}
	q := url.Values{}
	q.Set("cql", cql)
	q.Set("limit", strconv.Itoa(limit))
	var out ConfluenceSearchResult
	if err := c.rc.do(ctx, http.MethodGet, "/rest/api/search", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetPage fetches a content object by id. expand controls expansions, e.g.
// "body.storage,version,space".
func (c *ConfluenceClient) GetPage(ctx context.Context, id, expand string) (*Page, error) {
	q := url.Values{}
	if expand != "" {
		q.Set("expand", expand)
	}
	var out Page
	if err := c.rc.do(ctx, http.MethodGet, "/rest/api/content/"+id, q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreatePage creates a page in spaceKey with storage-format body. ancestorID,
// when non-empty, nests the new page under that parent.
func (c *ConfluenceClient) CreatePage(ctx context.Context, spaceKey, title, storageBody, ancestorID string) (*Page, error) {
	body := map[string]any{
		"type":  "page",
		"title": title,
		"space": map[string]any{"key": spaceKey},
		"body": map[string]any{
			"storage": map[string]any{"value": storageBody, "representation": "storage"},
		},
	}
	if ancestorID != "" {
		body["ancestors"] = []any{map[string]any{"id": ancestorID}}
	}
	var out Page
	if err := c.rc.do(ctx, http.MethodPost, "/rest/api/content", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdatePage replaces a page's title and storage body. currentVersion is the
// page's existing version number; Confluence requires the next number on write.
func (c *ConfluenceClient) UpdatePage(ctx context.Context, id, title, storageBody string, currentVersion int) (*Page, error) {
	body := map[string]any{
		"type":    "page",
		"title":   title,
		"version": map[string]any{"number": currentVersion + 1},
		"body": map[string]any{
			"storage": map[string]any{"value": storageBody, "representation": "storage"},
		},
	}
	var out Page
	if err := c.rc.do(ctx, http.MethodPut, "/rest/api/content/"+id, nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Comment is a created Confluence comment.
type Comment struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

// AddComment adds a footer comment (storage-format body) to a page.
func (c *ConfluenceClient) AddComment(ctx context.Context, pageID, storageBody string) (*Comment, error) {
	body := map[string]any{
		"type":      "comment",
		"container": map[string]any{"id": pageID, "type": "page"},
		"body": map[string]any{
			"storage": map[string]any{"value": storageBody, "representation": "storage"},
		},
	}
	var out Comment
	if err := c.rc.do(ctx, http.MethodPost, "/rest/api/content", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
