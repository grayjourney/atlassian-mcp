package atlassian

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// FieldSchema describes a Jira field's type, mirroring the subset we act on.
type FieldSchema struct {
	Type   string `json:"type"`   // string, number, array, option, user, date, datetime, ...
	Items  string `json:"items"`  // element type when Type == "array"
	Custom string `json:"custom"` // custom field plugin key, e.g. ...:gh-story-points
	System string `json:"system"` // system field key when not custom
}

// Field is a Jira field definition from GET /rest/api/3/field.
type Field struct {
	ID     string      `json:"id"`
	Key    string      `json:"key"`
	Name   string      `json:"name"`
	Custom bool        `json:"custom"`
	Schema FieldSchema `json:"schema"`
}

// GetFields lists every field (system + custom) defined on the Jira instance.
// The tool layer caches this per call to resolve field names to IDs.
func (j *JiraClient) GetFields(ctx context.Context) ([]Field, error) {
	var out []Field
	if err := j.rc.do(ctx, http.MethodGet, "/rest/api/3/field", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ChangelogItem is one field change within a changelog entry.
type ChangelogItem struct {
	Field      string `json:"field"`
	FromString string `json:"fromString"`
	ToString   string `json:"toString"`
}

// ChangelogEntry is one history record (a set of field changes by one author).
type ChangelogEntry struct {
	ID     string `json:"id"`
	Author struct {
		DisplayName string `json:"displayName"`
	} `json:"author"`
	Created string          `json:"created"`
	Items   []ChangelogItem `json:"items"`
}

// ChangelogPage is a page of an issue's change history.
type ChangelogPage struct {
	Values     []ChangelogEntry `json:"values"`
	StartAt    int              `json:"startAt"`
	MaxResults int              `json:"maxResults"`
	Total      int              `json:"total"`
	IsLast     bool             `json:"isLast"`
}

// GetChangelog returns a page of an issue's change history.
func (j *JiraClient) GetChangelog(ctx context.Context, key string, startAt, maxResults int) (*ChangelogPage, error) {
	q := url.Values{}
	q.Set("startAt", strconv.Itoa(startAt))
	if maxResults > 0 {
		q.Set("maxResults", strconv.Itoa(maxResults))
	}
	var out ChangelogPage
	if err := j.rc.do(ctx, http.MethodGet, "/rest/api/3/issue/"+key+"/changelog", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteIssue permanently deletes an issue (and its subtasks when deleteSubtasks).
func (j *JiraClient) DeleteIssue(ctx context.Context, key string, deleteSubtasks bool) error {
	q := url.Values{}
	if deleteSubtasks {
		q.Set("deleteSubtasks", "true")
	}
	return j.rc.do(ctx, http.MethodDelete, "/rest/api/3/issue/"+key, q, nil, nil)
}

// User is a Jira Cloud user as returned by user search.
type User struct {
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
	Email       string `json:"emailAddress"`
	Active      bool   `json:"active"`
}

// SearchUsers finds users matching query (email or display name), used to
// resolve an assignee to an account ID.
func (j *JiraClient) SearchUsers(ctx context.Context, query string, maxResults int) ([]User, error) {
	q := url.Values{}
	q.Set("query", query)
	if maxResults > 0 {
		q.Set("maxResults", strconv.Itoa(maxResults))
	}
	var out []User
	if err := j.rc.do(ctx, http.MethodGet, "/rest/api/3/user/search", q, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// FieldOption is one allowed value of a select/multi-select custom field.
type FieldOption struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

// GetFieldOptions lists the allowed values of a custom field. It resolves the
// field's first context, then that context's options (the Cloud API requires
// both steps).
func (j *JiraClient) GetFieldOptions(ctx context.Context, fieldID string) ([]FieldOption, error) {
	var contexts struct {
		Values []struct {
			ID string `json:"id"`
		} `json:"values"`
	}
	if err := j.rc.do(ctx, http.MethodGet, "/rest/api/3/field/"+fieldID+"/context", nil, nil, &contexts); err != nil {
		return nil, err
	}
	if len(contexts.Values) == 0 {
		return nil, nil
	}
	ctxID := contexts.Values[0].ID
	var opts struct {
		Values []FieldOption `json:"values"`
	}
	path := "/rest/api/3/field/" + fieldID + "/context/" + ctxID + "/option"
	if err := j.rc.do(ctx, http.MethodGet, path, nil, nil, &opts); err != nil {
		return nil, err
	}
	return opts.Values, nil
}
