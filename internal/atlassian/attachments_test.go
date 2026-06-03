package atlassian

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJiraGetAttachments(t *testing.T) {
	cap := &capture{}
	resp := `{"fields":{"attachment":[
		{"id":"10000","filename":"spec.txt","size":12,"mimeType":"text/plain",
		 "content":"https://x/rest/api/3/attachment/content/10000","created":"2026-01-01","author":{"displayName":"Ada"}}
	]}}`
	srv := newServer(t, 200, resp, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	atts, err := jc.GetAttachments(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatalf("GetAttachments: %v", err)
	}
	if cap.path != "/rest/api/3/issue/PROJ-1" || cap.query != "fields=attachment" {
		t.Errorf("got %s?%s", cap.path, cap.query)
	}
	if len(atts) != 1 {
		t.Fatalf("len = %d, want 1", len(atts))
	}
	a := atts[0]
	if a.ID != "10000" || a.Filename != "spec.txt" || a.Size != 12 || a.MimeType != "text/plain" {
		t.Errorf("attachment = %+v", a)
	}
}

func TestJiraDownloadAttachment(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/plain")
		_, _ = io.WriteString(w, "hello world")
	}))
	t.Cleanup(srv.Close)
	jc := NewJiraClient(srv.URL, "u", "tok", srv.Client())

	data, ctype, err := jc.DownloadAttachment(context.Background(), srv.URL+"/rest/api/3/attachment/content/1")
	if err != nil {
		t.Fatalf("DownloadAttachment: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("data = %q", string(data))
	}
	if ctype != "text/plain" {
		t.Errorf("content-type = %q", ctype)
	}
	wantBasicAuth(t, gotAuth, "u", "tok")
}
