package serve

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
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

// minimapJS draws the minimap enabled by diagram.serve.minimap. See the
// package comment for how it works.
const minimapJS = `<script>
(function() {
  var MAX = 240, MIN = 40;
  var selector = ".emlang-trigger, .emlang-command, .emlang-view, .emlang-event, .emlang-exception";
  var canvas = document.createElement("canvas");
  canvas.style.cssText = "position: fixed; top: 8px; right: 8px; z-index: 1000;" +
    "background: rgba(248, 249, 250, 0.9); border: 1px solid #adb5bd; border-radius: 4px;" +
    "cursor: pointer; opacity: 0.6; transition: opacity 0.15s; touch-action: none;";
  canvas.addEventListener("mouseenter", function() { canvas.style.opacity = "1"; });
  canvas.addEventListener("mouseleave", function() { canvas.style.opacity = "0.6"; });
  document.body.appendChild(canvas);

  var sx = 1, sy = 1, pending = false;

  function draw() {
    pending = false;
    var root = document.documentElement;
    var docW = root.scrollWidth, docH = root.scrollHeight;
    var viewW = root.clientWidth, viewH = root.clientHeight;
    if (docW <= viewW && docH <= viewH) {
      canvas.style.display = "none";
      return;
    }
    canvas.style.display = "block";

    var scale = Math.min(MAX / docW, MAX / docH);
    var w = Math.max(MIN, Math.round(docW * scale));
    var h = Math.max(MIN, Math.round(docH * scale));
    sx = w / docW;
    sy = h / docH;

    var dpr = window.devicePixelRatio || 1;
    canvas.style.width = w + "px";
    canvas.style.height = h + "px";
    canvas.width = Math.round(w * dpr);
    canvas.height = Math.round(h * dpr);
    var ctx = canvas.getContext("2d");
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, w, h);

    var items = document.querySelectorAll(selector);
    for (var i = 0; i < items.length; i++) {
      var r = items[i].getBoundingClientRect();
      ctx.fillStyle = getComputedStyle(items[i]).backgroundColor;
      ctx.fillRect((r.left + window.scrollX) * sx, (r.top + window.scrollY) * sy,
        Math.max(1, r.width * sx), Math.max(1, r.height * sy));
    }

    ctx.fillStyle = "rgba(73, 80, 87, 0.15)";
    ctx.strokeStyle = "#495057";
    ctx.lineWidth = 1;
    var vx = window.scrollX * sx, vy = window.scrollY * sy;
    ctx.fillRect(vx, vy, viewW * sx, viewH * sy);
    ctx.strokeRect(vx + 0.5, vy + 0.5, Math.max(1, viewW * sx - 1), Math.max(1, viewH * sy - 1));
  }

  function schedule() {
    if (!pending) {
      pending = true;
      requestAnimationFrame(draw);
    }
  }

  // Scrolls the page so that the viewport is centered on the point under
  // the pointer.
  function scrollToPointer(e) {
    var b = canvas.getBoundingClientRect();
    var root = document.documentElement;
    window.scrollTo((e.clientX - b.left) / sx - root.clientWidth / 2,
      (e.clientY - b.top) / sy - root.clientHeight / 2);
  }

  var dragging = false;
  canvas.addEventListener("pointerdown", function(e) {
    dragging = true;
    canvas.setPointerCapture(e.pointerId);
    scrollToPointer(e);
    e.preventDefault();
  });
  canvas.addEventListener("pointermove", function(e) {
    if (dragging) scrollToPointer(e);
  });
  canvas.addEventListener("pointerup", function() { dragging = false; });
  canvas.addEventListener("pointercancel", function() { dragging = false; });

  window.addEventListener("scroll", schedule);
  window.addEventListener("resize", schedule);
  window.addEventListener("load", schedule);
  if (window.ResizeObserver) new ResizeObserver(schedule).observe(document.body);
  schedule();
})();
</script>`

// wrapHTML wraps an HTML fragment in a full HTML page with live-reload
// script. It adds the minimap script when minimap is true.
func wrapHTML(fragment []byte, minimap bool) []byte {
	scripts := pollJS
	if minimap {
		scripts += "\n" + minimapJS
	}
	return []byte("<!DOCTYPE html>\n<html><head><meta charset=\"utf-8\"><title>emlang diagram</title></head>\n<body>\n" +
		string(fragment) +
		scripts + "\n</body></html>\n")
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

	// lastErr is the last render error reported, so the same error is
	// reported once while the file stays broken. Only the watcher
	// goroutine uses it.
	lastErr string
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

// reload renders the file again if it changed since the last good render.
// It reports a new diagram on stdout and a render error on stderr. A broken
// file is rendered again on every call, since an editor may still be
// writing it, but the same error is reported only once.
func (s *state) reload(filePath string, cfg *config.Config, stdout, stderr io.Writer) {
	info, err := os.Stat(filePath)
	if err != nil {
		return
	}
	s.mu.RLock()
	changed := info.ModTime().After(s.lastMod)
	s.mu.RUnlock()
	if !changed {
		return
	}

	html, err := generate(filePath, cfg)
	if err != nil {
		if err.Error() != s.lastErr {
			s.lastErr = err.Error()
			fmt.Fprintf(stderr, "Regeneration error: %v\n", err)
		}
		return
	}
	s.lastErr = ""

	s.mu.Lock()
	s.lastMod = info.ModTime()
	s.mu.Unlock()
	s.update(html)
	fmt.Fprintln(stdout, "Diagram updated.")
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

	return wrapHTML(fragment, cfg.Diagram.Serve.Minimap), nil
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
				s.reload(filePath, cfg, os.Stdout, os.Stderr)
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
