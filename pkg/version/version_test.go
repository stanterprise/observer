package version

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerReturnsInjectedValues(t *testing.T) {
	origV, origC, origD := Version, Commit, BuildDate
	t.Cleanup(func() { Version, Commit, BuildDate = origV, origC, origD })
	Version, Commit, BuildDate = "v1.2.3", "abc123", "2026-10-01T00:00:00Z"

	rec := httptest.NewRecorder()
	Handler(rec, httptest.NewRequest(http.MethodGet, "/api/version", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q", ct)
	}
	var got Info
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Version != "v1.2.3" || got.Commit != "abc123" || got.BuildDate != "2026-10-01T00:00:00Z" {
		t.Fatalf("unexpected info: %+v", got)
	}
	if got.GoVersion == "" {
		t.Fatal("goVersion should be populated")
	}
}
