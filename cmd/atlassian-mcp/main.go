// Command atlassian-mcp is an MCP server exposing Jira and Confluence tools for
// Atlassian Cloud. It also hosts a loopback setup dashboard for entering
// credentials. Communicates with the MCP client over stdio.
//
// IMPORTANT: stdout is the MCP transport — all human-facing logging goes to
// stderr.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/grayjourney/atlassian-mcp/internal/config"
	"github.com/grayjourney/atlassian-mcp/internal/dashboard"
	"github.com/grayjourney/atlassian-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const version = "0.2.0"

func main() {
	log.SetOutput(os.Stderr)
	log.SetPrefix("atlassian-mcp: ")
	log.SetFlags(0)

	checkConfig := flag.Bool("check-config", false, "exit 0 if configured, 1 otherwise (for the setup hook)")
	showVersion := flag.Bool("version", false, "print version and exit")
	port := flag.Int("dashboard-port", defaultDashboardPort(), "port for the loopback setup dashboard")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if *checkConfig {
		if cfg.IsConfigured() {
			fmt.Println("configured")
			return
		}
		fmt.Println("not configured")
		os.Exit(1)
	}

	cfgPath, err := config.Path()
	if err != nil {
		log.Fatalf("resolve config path: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Setup dashboard (also the live config provider for the tools).
	dash := dashboard.New(*port, cfgPath, cfg)
	go func() {
		if err := dash.Start(ctx); err != nil {
			log.Printf("dashboard stopped: %v", err)
		}
	}()
	log.Printf("setup dashboard at %s", dash.URL())
	if !cfg.IsConfigured() {
		log.Printf("not configured yet — open %s to add credentials", dash.URL())
	}

	// MCP server over stdio.
	srv := mcp.NewServer(&mcp.Implementation{Name: "atlassian-mcp", Version: version}, nil)
	tools.NewServer(dash.Config, dash.URL()).Register(srv)

	if err := srv.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatalf("mcp server: %v", err)
	}
}

// defaultDashboardPort honors ATLASSIAN_MCP_DASHBOARD_PORT, falling back to 24285.
func defaultDashboardPort() int {
	if v := os.Getenv("ATLASSIAN_MCP_DASHBOARD_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			return p
		}
	}
	return 24285
}
