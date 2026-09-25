package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func testHandler() *Handler {
	return newHandler(fstest.MapFS{
		"index.html":           {Data: []byte("<html>app</html>")},
		"assets/index-abc.js":  {Data: []byte("console.log(1)")},
		"assets/index-abc.css": {Data: []byte("body{}")},
		"favicon.svg":          {Data: []byte("<svg/>")},
	})
}

func get(h http.Handler, target string, header ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestServesIndexAtRoot(t *testing.T) {
	rec := get(testHandler(), "/")
	if rec.Code != http.StatusOK || rec.Body.String() != "<html>app</html>" {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", got)
	}
}

func TestHashedAssetsAreImmutable(t *testing.T) {
	rec := get(testHandler(), "/assets/index-abc.js")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Errorf("Cache-Control = %q", got)
	}
}

func TestReturns304WhenETagMatches(t *testing.T) {
	h := testHandler()
	etag := get(h, "/favicon.svg").Header().Get("Etag")
	if etag == "" {
		t.Fatal("expected an ETag")
	}
	if rec := get(h, "/favicon.svg", "If-None-Match", etag); rec.Code != http.StatusNotModified {
		t.Errorf("status = %d, want 304", rec.Code)
	}
}

func TestUnknownRouteFallsBackToIndex(t *testing.T) {
	if rec := get(testHandler(), "/some/route"); rec.Body.String() != "<html>app</html>" {
		t.Errorf("body = %q", rec.Body.String())
	}
}

func TestMissingFileIs404(t *testing.T) {
	if rec := get(testHandler(), "/assets/missing.js"); rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestTraversalStaysInside(t *testing.T) {
	if rec := get(testHandler(), "/../../etc/passwd"); rec.Code != http.StatusOK || rec.Body.String() != "<html>app</html>" {
		t.Errorf("got %d %q", rec.Code, rec.Body.String())
	}
}

func TestPlaceholderWhenNotBuilt(t *testing.T) {
	h := newHandler(fstest.MapFS{".gitkeep": {Data: nil}})
	rec := get(h, "/")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "make web") {
		t.Errorf("got %d %q", rec.Code, rec.Body.String())
	}
}
