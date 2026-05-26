// Package dashboard runs a local, loopback-only web UI for entering Atlassian
// credentials — the serena-style setup flow. The MCP server hosts it in a
// goroutine; a Claude Code SessionStart hook opens it in the browser when the
// server is not yet configured.
package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/grayjourney/atlassian-mcp/internal/config"
)

// Dashboard owns the current config (hot-reloadable) and the HTTP UI.
type Dashboard struct {
	port int
	path string
	cur  atomic.Pointer[config.Config]
}

// New creates a dashboard bound to port, persisting to path, seeded with initial.
func New(port int, path string, initial *config.Config) *Dashboard {
	d := &Dashboard{port: port, path: path}
	if initial == nil {
		initial = &config.Config{}
	}
	d.cur.Store(initial)
	return d
}

// Config returns the latest config; pass this method as the tools provider.
func (d *Dashboard) Config() *config.Config { return d.cur.Load() }

// Addr is the host:port the dashboard listens on (loopback only).
func (d *Dashboard) Addr() string { return fmt.Sprintf("127.0.0.1:%d", d.port) }

// URL is the browsable dashboard address.
func (d *Dashboard) URL() string { return "http://" + d.Addr() }

// Handler builds the HTTP routes.
func (d *Dashboard) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", d.handleHealth)
	mux.HandleFunc("/config", d.handleConfig)
	mux.HandleFunc("/", d.handleIndex)
	return mux
}

// Start serves until ctx is cancelled, then shuts down gracefully.
func (d *Dashboard) Start(ctx context.Context) error {
	srv := &http.Server{Addr: d.Addr(), Handler: d.Handler()}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (d *Dashboard) handleHealth(w http.ResponseWriter, _ *http.Request) {
	cfg := d.cur.Load()
	writeJSON(w, http.StatusOK, map[string]any{
		"configured": cfg.IsConfigured(),
		"jira":       cfg.JiraConfigured(),
		"confluence": cfg.ConfluenceConfigured(),
	})
}

func (d *Dashboard) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Return current values so the form can prefill (tokens included; this is
		// loopback-only and the file is already 0600 on disk).
		writeJSON(w, http.StatusOK, d.cur.Load())
	case http.MethodPost:
		var incoming config.Config
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON: " + err.Error()})
			return
		}
		if err := incoming.SaveTo(d.path); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		d.cur.Store(&incoming)
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":         true,
			"configured": incoming.IsConfigured(),
			"jira":       incoming.JiraConfigured(),
			"confluence": incoming.ConfluenceConfigured(),
		})
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (d *Dashboard) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(indexHTML)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
