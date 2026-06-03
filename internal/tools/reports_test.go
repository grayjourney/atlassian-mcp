package tools

import (
	"testing"

	"github.com/grayjourney/atlassian-mcp/internal/atlassian"
)

func reportIssues() []atlassian.Issue {
	return []atlassian.Issue{
		{Key: "A", Fields: map[string]any{
			"status":         map[string]any{"name": "Done", "statusCategory": map[string]any{"key": "done"}},
			"assignee":       map[string]any{"displayName": "Ada"},
			"issuetype":      map[string]any{"name": "Task"},
			"customfield_sp": float64(3),
		}},
		{Key: "B", Fields: map[string]any{
			"status":         map[string]any{"name": "In Progress", "statusCategory": map[string]any{"key": "indeterminate"}},
			"issuetype":      map[string]any{"name": "Bug"},
			"customfield_sp": float64(5),
		}},
		{Key: "C", Fields: map[string]any{
			"status":    map[string]any{"name": "To Do", "statusCategory": map[string]any{"key": "new"}},
			"assignee":  map[string]any{"displayName": "Ada"},
			"issuetype": map[string]any{"name": "Task"},
		}},
	}
}

func TestAggregateIssues(t *testing.T) {
	rep := aggregateIssues(reportIssues(), "customfield_sp")

	if rep["total"].(int) != 3 {
		t.Errorf("total = %v", rep["total"])
	}
	if rep["done"].(int) != 1 || rep["in_progress"].(int) != 1 || rep["to_do"].(int) != 1 {
		t.Errorf("done/inprog/todo = %v/%v/%v", rep["done"], rep["in_progress"], rep["to_do"])
	}
	byAssignee := rep["by_assignee"].(map[string]int)
	if byAssignee["Ada"] != 2 || byAssignee["Unassigned"] != 1 {
		t.Errorf("by_assignee = %v", byAssignee)
	}
	byType := rep["by_type"].(map[string]int)
	if byType["Task"] != 2 || byType["Bug"] != 1 {
		t.Errorf("by_type = %v", byType)
	}
	byStatus := rep["by_status"].(map[string]int)
	if byStatus["Done"] != 1 || byStatus["In Progress"] != 1 || byStatus["To Do"] != 1 {
		t.Errorf("by_status = %v", byStatus)
	}
	sp := rep["story_points"].(map[string]any)
	if sp["total"].(float64) != 8 || sp["completed"].(float64) != 3 || sp["remaining"].(float64) != 5 {
		t.Errorf("story_points = %v", sp)
	}
}

func TestAggregateIssuesNoStoryPoints(t *testing.T) {
	rep := aggregateIssues(reportIssues(), "")
	if _, ok := rep["story_points"]; ok {
		t.Errorf("story_points should be absent when no field id given: %v", rep["story_points"])
	}
}
