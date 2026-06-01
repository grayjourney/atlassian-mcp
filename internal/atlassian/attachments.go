package atlassian

import (
	"context"
	"net/http"
	"net/url"
)

// Attachment is a file attached to a Jira issue. Content is the absolute
// download URL.
type Attachment struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Size     int    `json:"size"`
	MimeType string `json:"mimeType"`
	Content  string `json:"content"`
	Created  string `json:"created"`
	Author   struct {
		DisplayName string `json:"displayName"`
	} `json:"author"`
}

// GetAttachments lists the attachments on an issue (metadata only).
func (j *JiraClient) GetAttachments(ctx context.Context, key string) ([]Attachment, error) {
	q := url.Values{}
	q.Set("fields", "attachment")
	var out struct {
		Fields struct {
			Attachment []Attachment `json:"attachment"`
		} `json:"fields"`
	}
	if err := j.rc.do(ctx, http.MethodGet, "/rest/api/3/issue/"+key, q, nil, &out); err != nil {
		return nil, err
	}
	return out.Fields.Attachment, nil
}

// DownloadAttachment fetches an attachment's bytes from its content URL,
// returning the data and its Content-Type.
func (j *JiraClient) DownloadAttachment(ctx context.Context, contentURL string) ([]byte, string, error) {
	return j.rc.getBytes(ctx, contentURL)
}
