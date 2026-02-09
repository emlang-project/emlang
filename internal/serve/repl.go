package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/emlang-project/emlang/internal/config"
	"github.com/emlang-project/emlang/internal/diagram"
	"github.com/emlang-project/emlang/internal/formatter"
	"github.com/emlang-project/emlang/internal/linter"
	"github.com/emlang-project/emlang/internal/parser"
	"github.com/emlang-project/emlang/internal/repl"
)

type lintIssueJSON struct {
	Rule     string `json:"rule"`
	Message  string `json:"message"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Severity string `json:"severity"`
}

type renderResponse struct {
	HTML  string          `json:"html,omitempty"`
	Error string          `json:"error,omitempty"`
	Lint  []lintIssueJSON `json:"lint,omitempty"`
}

type formatResponse struct {
	YAML  string `json:"yaml,omitempty"`
	Error string `json:"error,omitempty"`
}

// StartRepl starts the interactive REPL HTTP server.
// If filePath is not empty, its content is loaded as the initial editor value.
func StartRepl(filePath string, addr string, port int, cfg *config.Config) error {
	var initialContent string
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("reading %s: %w", filePath, err)
		}
		initialContent = string(data)
	}

	gen := diagram.New()
	gen.CSSOverrides = cfg.Diagram.CSS

	lint := linter.New()
	for _, rule := range cfg.Lint.Ignore {
		lint.IgnoreRules[rule] = true
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, repl.Page)
	})

	mux.HandleFunc("/initial", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, initialContent)
	})

	mux.HandleFunc("/render", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			json.NewEncoder(w).Encode(renderResponse{Error: err.Error()})
			return
		}

		doc, err := parser.Parse(strings.NewReader(string(body)))
		if err != nil {
			json.NewEncoder(w).Encode(renderResponse{Error: err.Error()})
			return
		}

		html, err := gen.Generate(doc)
		if err != nil {
			json.NewEncoder(w).Encode(renderResponse{Error: err.Error()})
			return
		}

		issues := lint.Lint(doc)
		var lintItems []lintIssueJSON
		for _, issue := range issues {
			lintItems = append(lintItems, lintIssueJSON{
				Rule:     issue.Rule,
				Message:  issue.Message,
				Line:     issue.Line,
				Column:   issue.Column,
				Severity: issue.Severity.String(),
			})
		}

		json.NewEncoder(w).Encode(renderResponse{
			HTML: string(html),
			Lint: lintItems,
		})
	})

	mux.HandleFunc("/format", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			json.NewEncoder(w).Encode(formatResponse{Error: err.Error()})
			return
		}

		doc, err := parser.Parse(strings.NewReader(string(body)))
		if err != nil {
			json.NewEncoder(w).Encode(formatResponse{Error: err.Error()})
			return
		}

		keyStyle := "long"
		if cfg.Fmt.Keys != "" {
			keyStyle = cfg.Fmt.Keys
		}

		out := formatter.Format(doc, formatter.Options{KeyStyle: keyStyle})
		json.NewEncoder(w).Encode(formatResponse{YAML: string(out)})
	})

	listenAddr := fmt.Sprintf("%s:%d", addr, port)
	server := &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down REPL server...")
		server.Shutdown(context.Background())
	}()

	displayHost := addr
	if displayHost == "" || displayHost == "0.0.0.0" {
		displayHost = "localhost"
	}
	url := fmt.Sprintf("http://%s:%d", displayHost, port)
	fmt.Printf("REPL running at %s\n", url)
	openBrowser(url)

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}
