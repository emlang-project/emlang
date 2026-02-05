package serve

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync"
	"time"

	"github.com/emlang-project/emlang/internal/config"
	"github.com/emlang-project/emlang/internal/diagram"
	"github.com/emlang-project/emlang/internal/parser"
)

const pollJS = `<script>
(function() {
  var hash = "";
  setInterval(function() {
    fetch("/hash").then(function(r) { return r.text(); }).then(function(h) {
      if (hash && h !== hash) location.reload();
      hash = h;
    });
  }, 1000);
})();
</script>`

// wrapHTML wraps an HTML fragment in a full HTML page with live-reload script.
func wrapHTML(fragment []byte) []byte {
	return []byte("<!DOCTYPE html>\n<html><head><meta charset=\"utf-8\"><title>emlang diagram</title></head>\n<body>\n" +
		string(fragment) +
		pollJS + "\n</body></html>\n")
}

// hashBytes returns a hex-encoded SHA-256 hash of the given bytes.
func hashBytes(b []byte) string {
	h := sha256.Sum256(b)
	return fmt.Sprintf("%x", h)
}

type state struct {
	mu      sync.RWMutex
	html    []byte
	hash    string
	lastMod time.Time
}

func (s *state) update(html []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.html = html
	s.hash = hashBytes(html)
}

func (s *state) getHTML() []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.html
}

func (s *state) getHash() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hash
}

// generate parses the file and generates the wrapped HTML page.
func generate(filePath string, cfg *config.Config) ([]byte, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	doc, err := parser.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	gen := diagram.New()
	gen.CSSOverrides = cfg.Diagram.CSS
	fragment, err := gen.Generate(doc)
	if err != nil {
		return nil, fmt.Errorf("diagram generation error: %w", err)
	}

	return wrapHTML(fragment), nil
}

// openBrowser tries to open the given URL in the default browser.
// Errors are silently ignored.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	_ = cmd.Start()
}

// Start starts the live-reload HTTP server for the given file.
func Start(filePath string, addr string, port int, cfg *config.Config) error {
	html, err := generate(filePath, cfg)
	if err != nil {
		return err
	}

	s := &state{}
	s.update(html)

	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	s.lastMod = info.ModTime()

	// File watcher goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				info, err := os.Stat(filePath)
				if err != nil {
					continue
				}
				s.mu.RLock()
				changed := info.ModTime().After(s.lastMod)
				s.mu.RUnlock()
				if !changed {
					continue
				}
				newHTML, err := generate(filePath, cfg)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Regeneration error: %v\n", err)
					continue
				}
				s.mu.Lock()
				s.lastMod = info.ModTime()
				s.mu.Unlock()
				s.update(newHTML)
				fmt.Println("Diagram updated.")
			}
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(s.getHTML())
	})
	mux.HandleFunc("/hash", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, s.getHash())
	})

	listenAddr := fmt.Sprintf("%s:%d", addr, port)
	server := &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	// Graceful shutdown on SIGINT/SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down server...")
		cancel()
		server.Shutdown(context.Background())
	}()

	displayHost := addr
	if displayHost == "" || displayHost == "0.0.0.0" {
		displayHost = "localhost"
	}
	url := fmt.Sprintf("http://%s:%d", displayHost, port)
	fmt.Printf("Serving diagram at %s\n", url)
	openBrowser(url)

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}
