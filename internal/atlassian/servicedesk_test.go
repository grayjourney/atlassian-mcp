package atlassian

import (
	"context"
	"testing"
)

func TestJiraGetServiceDesks(t *testing.T) {
	cap := &capture{}
	resp := `{"values":[{"id":"1","projectId":"10000","projectKey":"SUP","projectName":"Support"}]}`
	srv := newServer(t, 200, resp, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	ds, err := jc.GetServiceDesks(context.Background())
	if err != nil {
		t.Fatalf("GetServiceDesks: %v", err)
	}
	if cap.path != "/rest/servicedeskapi/servicedesk" {
		t.Errorf("path = %q", cap.path)
	}
	if len(ds) != 1 || ds[0].ID != "1" || ds[0].ProjectKey != "SUP" {
		t.Errorf("service desks = %+v", ds)
	}
}

func TestJiraGetQueues(t *testing.T) {
	cap := &capture{}
	resp := `{"values":[{"id":"5","name":"Open requests","issueCount":7}]}`
	srv := newServer(t, 200, resp, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	qs, err := jc.GetQueues(context.Background(), "1")
	if err != nil {
		t.Fatalf("GetQueues: %v", err)
	}
	if cap.path != "/rest/servicedeskapi/servicedesk/1/queue" {
		t.Errorf("path = %q", cap.path)
	}
	if len(qs) != 1 || qs[0].ID != "5" || qs[0].Name != "Open requests" || qs[0].IssueCount != 7 {
		t.Errorf("queues = %+v", qs)
	}
}

func TestJiraGetQueueIssues(t *testing.T) {
	cap := &capture{}
	resp := `{"values":[{"id":"1","key":"SUP-1","fields":{"summary":"help"}}]}`
	srv := newServer(t, 200, resp, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	issues, err := jc.GetQueueIssues(context.Background(), "1", "5", 25)
	if err != nil {
		t.Fatalf("GetQueueIssues: %v", err)
	}
	if cap.path != "/rest/servicedeskapi/servicedesk/1/queue/5/issue" {
		t.Errorf("path = %q", cap.path)
	}
	if len(issues) != 1 || issues[0].Key != "SUP-1" {
		t.Errorf("issues = %+v", issues)
	}
}

func TestJiraGetDevelopmentSummary(t *testing.T) {
	cap := &capture{}
	resp := `{"errors":[],"summary":{"pullrequest":{"overall":{"count":2,"state":"OPEN"}},"branch":{"overall":{"count":1}},"repository":{"overall":{"count":3}}}}`
	srv := newServer(t, 200, resp, cap)
	jc := NewJiraClient(srv.URL, "u", "t", srv.Client())

	sum, err := jc.GetDevelopmentSummary(context.Background(), "10001")
	if err != nil {
		t.Fatalf("GetDevelopmentSummary: %v", err)
	}
	if cap.path != "/rest/dev-status/latest/issue/summary" {
		t.Errorf("path = %q", cap.path)
	}
	if cap.query != "issueId=10001" {
		t.Errorf("query = %q", cap.query)
	}
	pr, _ := sum["pullrequest"].(map[string]any)
	overall, _ := pr["overall"].(map[string]any)
	if overall["count"].(float64) != 2 {
		t.Errorf("pullrequest count = %v", overall["count"])
	}
}
