package serve

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/emlang-project/emlang/internal/config"
)

func TestWrapHTML(t *testing.T) {
	fragment := []byte(`<style>.test{}</style><div>hello</div>`)
	page := string(wrapHTML(fragment))

	if !strings.HasPrefix(page, "<!DOCTYPE html>") {
		t.Error("expected page to start with DOCTYPE")
	}
	if !strings.Contains(page, "<title>emlang diagram</title>") {
		t.Error("expected page to contain title")
	}
	if !strings.Contains(page, string(fragment)) {
		t.Error("expected page to contain the original fragment")
	}
	if !strings.Contains(page, `fetch("/hash")`) {
		t.Error("expected page to contain polling script")
	}
	if !strings.Contains(page, "</body></html>") {
		t.Error("expected page to end with closing tags")
	}
}

func TestHashBytes(t *testing.T) {
	h1 := hashBytes([]byte("hello"))
	h2 := hashBytes([]byte("hello"))
	h3 := hashBytes([]byte("world"))

	if h1 != h2 {
		t.Error("same input should produce same hash")
	}
	if h1 == h3 {
		t.Error("different input should produce different hash")
	}
	if len(h1) != 64 {
		t.Errorf("expected 64-char hex hash, got %d chars", len(h1))
	}
}

func TestHashHandler(t *testing.T) {
	s := &state{}
	s.update([]byte("<html>test</html>"))

	mux := http.NewServeMux()
	mux.HandleFunc("/hash", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(s.getHash()))
	})

	req := httptest.NewRequest("GET", "/hash", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if len(body) != 64 {
		t.Errorf("expected 64-char hash, got %q", body)
	}
}

func TestRootHandler(t *testing.T) {
	content := []byte("<!DOCTYPE html><html><body>diagram</body></html>")
	s := &state{}
	s.update(content)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(s.getHTML())
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html content type, got %q", ct)
	}
	if rec.Body.String() != string(content) {
		t.Error("expected response body to match stored HTML")
	}
}

func TestStateUpdate(t *testing.T) {
	s := &state{}
	s.update([]byte("version1"))
	hash1 := s.getHash()

	s.update([]byte("version2"))
	hash2 := s.getHash()

	if hash1 == hash2 {
		t.Error("hash should change when content changes")
	}
	if string(s.getHTML()) != "version2" {
		t.Error("HTML should be updated")
	}
}

func TestFileChangeDetection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")

	content1 := []byte("content1")
	if err := os.WriteFile(path, content1, 0644); err != nil {
		t.Fatal(err)
	}

	info1, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	// Write new content (may have same mtime on fast filesystems, but tests the stat path)
	content2 := []byte("content2")
	if err := os.WriteFile(path, content2, 0644); err != nil {
		t.Fatal(err)
	}

	info2, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	// On most filesystems, the mtime will be >= the original
	if info2.ModTime().Before(info1.ModTime()) {
		t.Error("new file mtime should not be before original")
	}
}

func TestReloadReportsErrorOnce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "model.yaml")
	cfg := &config.Config{}

	write := func(content string, mtime time.Time) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, mtime, mtime); err != nil {
			t.Fatal(err)
		}
	}

	start := time.Now().Add(-time.Hour)
	s := &state{lastMod: start}
	s.update([]byte("initial"))

	var stdout, stderr bytes.Buffer

	// A broken file is reported once, however many times it is checked.
	write("slices: [", start.Add(time.Second))
	for i := 0; i < 3; i++ {
		s.reload(path, cfg, &stdout, &stderr)
	}
	if n := strings.Count(stderr.String(), "Regeneration error"); n != 1 {
		t.Errorf("expected 1 error report, got %d:\n%s", n, stderr.String())
	}
	if string(s.getHTML()) != "initial" {
		t.Error("a broken file should keep the last good diagram")
	}

	// Fixing the file renders it, even with the same modification time.
	write("slices:\n  A:\n    - e: E\n", start.Add(time.Second))
	s.reload(path, cfg, &stdout, &stderr)
	if !strings.Contains(stdout.String(), "Diagram updated.") {
		t.Errorf("expected an update, got stdout:\n%s", stdout.String())
	}
	if string(s.getHTML()) == "initial" {
		t.Error("the fixed file should be rendered")
	}

	// No change: nothing is reported.
	stdout.Reset()
	s.reload(path, cfg, &stdout, &stderr)
	if stdout.Len() != 0 {
		t.Errorf("expected no output without a change, got:\n%s", stdout.String())
	}

	// Breaking the file again reports the error again.
	write("slices: [", start.Add(2*time.Second))
	s.reload(path, cfg, &stdout, &stderr)
	if n := strings.Count(stderr.String(), "Regeneration error"); n != 2 {
		t.Errorf("expected 2 error reports, got %d:\n%s", n, stderr.String())
	}
}
