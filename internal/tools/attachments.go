package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/grayjourney/atlassian-mcp/internal/atlassian"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// maxInlineAttachment caps how many bytes jira_read_attachment will return
// inline, to avoid flooding the model with a huge file.
const maxInlineAttachment = 100 * 1024

// resolveAttachment finds an attachment by id or (case-insensitive) filename.
func resolveAttachment(atts []atlassian.Attachment, ref string) (atlassian.Attachment, bool) {
	for _, a := range atts {
		if a.ID == ref || strings.EqualFold(a.Filename, ref) {
			return a, true
		}
	}
	return atlassian.Attachment{}, false
}

// isTextual reports whether a MIME type is human-readable text we can return
// inline.
func isTextual(mime string) bool {
	mime = strings.ToLower(mime)
	if strings.HasPrefix(mime, "text/") {
		return true
	}
	for _, frag := range []string{"json", "xml", "csv", "yaml", "x-yaml", "javascript", "html", "markdown"} {
		if strings.Contains(mime, frag) {
			return true
		}
	}
	return false
}

func attachmentMeta(a atlassian.Attachment) map[string]any {
	return map[string]any{
		"id": a.ID, "filename": a.Filename, "size": a.Size,
		"mime_type": a.MimeType, "created": a.Created, "author": a.Author.DisplayName,
	}
}

// --- jira_list_attachments ---

type jiraListAttachmentsInput struct {
	IssueKey string `json:"issue_key" jsonschema:"the issue key, e.g. PROJ-123"`
}

func (s *Server) jiraListAttachments(ctx context.Context, _ *mcp.CallToolRequest, in jiraListAttachmentsInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	atts, err := jc.GetAttachments(ctx, in.IssueKey)
	if err != nil {
		return nil, nil, err
	}
	out := make([]map[string]any, 0, len(atts))
	for _, a := range atts {
		out = append(out, attachmentMeta(a))
	}
	return textResult(map[string]any{"key": in.IssueKey, "attachments": out, "count": len(out)})
}

// --- jira_download_attachment ---

type jiraDownloadAttachmentInput struct {
	IssueKey   string `json:"issue_key" jsonschema:"the issue key, e.g. PROJ-123"`
	Attachment string `json:"attachment" jsonschema:"attachment id or filename"`
	Dir        string `json:"dir,omitempty" jsonschema:"directory to save into; defaults to a temp dir"`
}

func (s *Server) jiraDownloadAttachment(ctx context.Context, _ *mcp.CallToolRequest, in jiraDownloadAttachmentInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	a, err := s.findAttachment(ctx, jc, in.IssueKey, in.Attachment)
	if err != nil {
		return nil, nil, err
	}
	data, _, err := jc.DownloadAttachment(ctx, a.Content)
	if err != nil {
		return nil, nil, err
	}
	dir := in.Dir
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "atlassian-mcp")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, nil, fmt.Errorf("create download dir: %w", err)
	}
	// Guard against path traversal from a crafted filename.
	path := filepath.Join(dir, filepath.Base(a.Filename))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return nil, nil, fmt.Errorf("write attachment: %w", err)
	}
	return textResult(map[string]any{
		"key": in.IssueKey, "filename": a.Filename,
		"path": path, "size": len(data), "mime_type": a.MimeType,
	})
}

// --- jira_read_attachment ---

type jiraReadAttachmentInput struct {
	IssueKey   string `json:"issue_key" jsonschema:"the issue key, e.g. PROJ-123"`
	Attachment string `json:"attachment" jsonschema:"attachment id or filename"`
}

func (s *Server) jiraReadAttachment(ctx context.Context, _ *mcp.CallToolRequest, in jiraReadAttachmentInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	a, err := s.findAttachment(ctx, jc, in.IssueKey, in.Attachment)
	if err != nil {
		return nil, nil, err
	}
	if !isTextual(a.MimeType) {
		return nil, nil, fmt.Errorf("attachment %q is %s, not text; use jira_download_attachment to save it", a.Filename, a.MimeType)
	}
	data, _, err := jc.DownloadAttachment(ctx, a.Content)
	if err != nil {
		return nil, nil, err
	}
	truncated := false
	if len(data) > maxInlineAttachment {
		data = data[:maxInlineAttachment]
		truncated = true
	}
	return textResult(map[string]any{
		"key": in.IssueKey, "filename": a.Filename,
		"content": string(data), "truncated": truncated,
	})
}

// findAttachment lists an issue's attachments and resolves one by id/filename,
// returning a helpful error (with available names) when it's missing.
func (s *Server) findAttachment(ctx context.Context, jc *atlassian.JiraClient, key, ref string) (atlassian.Attachment, error) {
	atts, err := jc.GetAttachments(ctx, key)
	if err != nil {
		return atlassian.Attachment{}, err
	}
	a, ok := resolveAttachment(atts, ref)
	if !ok {
		names := make([]string, 0, len(atts))
		for _, x := range atts {
			names = append(names, x.Filename)
		}
		return atlassian.Attachment{}, fmt.Errorf("attachment %q not found on %s; available: %s",
			ref, key, strings.Join(names, ", "))
	}
	return a, nil
}
