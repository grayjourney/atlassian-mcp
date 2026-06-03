package tools

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/grayjourney/atlassian-mcp/internal/atlassian"
)

func TestResolveAttachment(t *testing.T) {
	atts := []atlassian.Attachment{
		{ID: "1", Filename: "spec.txt"},
		{ID: "2", Filename: "Diagram.PNG"},
	}
	tests := []struct {
		ref    string
		wantID string
		wantOK bool
	}{
		{"1", "1", true},
		{"spec.txt", "1", true},
		{"diagram.png", "2", true}, // case-insensitive filename
		{"missing", "", false},
	}
	for _, tt := range tests {
		a, ok := resolveAttachment(atts, tt.ref)
		if ok != tt.wantOK || (ok && a.ID != tt.wantID) {
			t.Errorf("resolveAttachment(%q) = %q,%v want %q,%v", tt.ref, a.ID, ok, tt.wantID, tt.wantOK)
		}
	}
}

func TestIsTextual(t *testing.T) {
	for _, m := range []string{"text/plain", "application/json", "text/csv", "application/xml"} {
		if !isTextual(m) {
			t.Errorf("isTextual(%q) = false, want true", m)
		}
	}
	for _, m := range []string{"image/png", "application/pdf", "application/octet-stream"} {
		if isTextual(m) {
			t.Errorf("isTextual(%q) = true, want false", m)
		}
	}
}

// TestDownloadAttachmentWritesFile drives the full download tool against a fake
// Jira (issue metadata + content endpoint) and checks the file lands on disk.
func TestDownloadAttachmentWritesFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/attachment/content/9"):
			w.Header().Set("Content-Type", "text/plain")
			_, _ = io.WriteString(w, "file body")
		default: // GET issue?fields=attachment
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"fields":{"attachment":[{"id":"9","filename":"notes.txt","size":9,"mimeType":"text/plain","content":"`+srvContentURL(r)+`"}]}}`)
		}
	}))
	defer srv.Close()
	s := newToolServer(t, srv)

	dir := t.TempDir()
	res, _, err := s.jiraDownloadAttachment(context.Background(), nil, jiraDownloadAttachmentInput{
		IssueKey: "KAN-1", Attachment: "notes.txt", Dir: dir,
	})
	if err != nil {
		t.Fatalf("jiraDownloadAttachment: %v", err)
	}
	m := resultText(t, res)
	path, _ := m["path"].(string)
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read saved file: %v", readErr)
	}
	if string(got) != "file body" {
		t.Errorf("saved content = %q", string(got))
	}
}

// srvContentURL builds the absolute content URL for the test server from the
// inbound request's host.
func srvContentURL(r *http.Request) string {
	return "http://" + r.Host + "/rest/api/3/attachment/content/9"
}
