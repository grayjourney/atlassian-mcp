package tools

import (
	"testing"

	"github.com/grayjourney/atlassian-mcp/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestRegisterAllTools ensures every tool's input struct produces a valid JSON
// schema — AddTool infers and validates the schema at registration time, so a
// bad jsonschema tag would panic here rather than in production.
func TestRegisterAllTools(t *testing.T) {
	cfg := &config.Config{}
	s := NewServer(func() *config.Config { return cfg }, "http://127.0.0.1:24285")
	srv := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0.0.0"}, nil)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Register panicked: %v", r)
		}
	}()
	s.Register(srv)
}
