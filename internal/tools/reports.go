package tools

import (
	"context"
	"fmt"

	"github.com/grayjourney/atlassian-mcp/internal/atlassian"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// reportMax is how many issues a report aggregates in one pass.
const reportMax = 100

// aggregateIssues rolls a set of issues into a status/assignee/type breakdown
// with done/in-progress/to-do counts (by status category) and, when spID is
// non-empty, a story-point total / completed / remaining split.
func aggregateIssues(issues []atlassian.Issue, spID string) map[string]any {
	byStatus := map[string]int{}
	byAssignee := map[string]int{}
	byType := map[string]int{}
	done, inProgress, toDo := 0, 0, 0
	var spTotal, spDone float64

	for _, iss := range issues {
		f := iss.Fields
		cat := ""
		if st, ok := f["status"].(map[string]any); ok {
			if name, ok := st["name"].(string); ok {
				byStatus[name]++
			}
			if sc, ok := st["statusCategory"].(map[string]any); ok {
				cat, _ = sc["key"].(string)
			}
		}
		switch cat {
		case "done":
			done++
		case "indeterminate":
			inProgress++
		default:
			toDo++
		}

		name := "Unassigned"
		if a, ok := f["assignee"].(map[string]any); ok {
			if dn, ok := a["displayName"].(string); ok {
				name = dn
			}
		}
		byAssignee[name]++

		if it, ok := f["issuetype"].(map[string]any); ok {
			if tn, ok := it["name"].(string); ok {
				byType[tn]++
			}
		}

		if spID != "" {
			if v, ok := toFloat(f[spID]); ok {
				spTotal += v
				if cat == "done" {
					spDone += v
				}
			}
		}
	}

	report := map[string]any{
		"total":       len(issues),
		"done":        done,
		"in_progress": inProgress,
		"to_do":       toDo,
		"by_status":   byStatus,
		"by_assignee": byAssignee,
		"by_type":     byType,
	}
	if spID != "" {
		report["story_points"] = map[string]any{
			"total": spTotal, "completed": spDone, "remaining": spTotal - spDone,
		}
	}
	return report
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	default:
		return 0, false
	}
}

// storyPointsID returns the instance's Story Points field id, or "" if none.
func (s *Server) storyPointsID(ctx context.Context, jc *atlassian.JiraClient) string {
	r, err := s.jiraFields(ctx, jc)
	if err != nil {
		return ""
	}
	if f, ok := r.storyPointsField(); ok {
		return f.ID
	}
	return ""
}

// --- jira_board_report ---

type jiraBoardReportInput struct {
	BoardID int `json:"board_id" jsonschema:"the board id"`
}

func (s *Server) jiraBoardReport(ctx context.Context, _ *mcp.CallToolRequest, in jiraBoardReportInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	issues, err := jc.GetBoardIssues(ctx, in.BoardID, "", reportMax)
	if err != nil {
		return nil, nil, err
	}
	rep := aggregateIssues(issues, s.storyPointsID(ctx, jc))
	rep["board_id"] = in.BoardID
	return textResult(rep)
}

// --- jira_sprint_report ---

type jiraSprintReportInput struct {
	SprintID int `json:"sprint_id" jsonschema:"the sprint id"`
}

func (s *Server) jiraSprintReport(ctx context.Context, _ *mcp.CallToolRequest, in jiraSprintReportInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	issues, err := jc.GetSprintIssues(ctx, in.SprintID, "", reportMax)
	if err != nil {
		return nil, nil, err
	}
	rep := aggregateIssues(issues, s.storyPointsID(ctx, jc))
	rep["sprint_id"] = in.SprintID
	return textResult(rep)
}

// --- jira_version_report ---

type jiraVersionReportInput struct {
	ProjectKey string `json:"project_key" jsonschema:"the project key, e.g. KAN"`
	Version    string `json:"version" jsonschema:"the version (fix version) name"`
}

func (s *Server) jiraVersionReport(ctx context.Context, _ *mcp.CallToolRequest, in jiraVersionReportInput) (*mcp.CallToolResult, any, error) {
	jc, _, err := s.jira()
	if err != nil {
		return nil, nil, err
	}
	spID := s.storyPointsID(ctx, jc)
	fields := []string{"status", "assignee", "issuetype"}
	if spID != "" {
		fields = append(fields, spID)
	}
	jql := fmt.Sprintf("project = %q AND fixVersion = %q", in.ProjectKey, in.Version)
	res, err := jc.Search(ctx, atlassian.SearchRequest{JQL: jql, Fields: fields, MaxResults: reportMax})
	if err != nil {
		return nil, nil, err
	}
	rep := aggregateIssues(res.Issues, spID)
	rep["project"] = in.ProjectKey
	rep["version"] = in.Version
	return textResult(rep)
}
