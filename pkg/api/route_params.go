package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func routeParamOrPath(r *http.Request, key, prefix, suffix string) string {
	if value := chi.URLParam(r, key); value != "" {
		return value
	}

	value := strings.TrimPrefix(r.URL.Path, prefix)
	if value == r.URL.Path {
		return ""
	}
	if suffix != "" {
		if !strings.HasSuffix(value, suffix) {
			return ""
		}
		value = strings.TrimSuffix(value, suffix)
	}

	return strings.Trim(value, "/")
}

func runAndTestParams(r *http.Request) (string, string) {
	runID := chi.URLParam(r, "runId")
	testID := chi.URLParam(r, "testId")
	if runID != "" || testID != "" {
		return runID, testID
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/runs/")
	if path == r.URL.Path {
		return "", ""
	}

	parts := strings.SplitN(path, "/tests/", 2)
	if len(parts) != 2 {
		return "", ""
	}

	return strings.Trim(parts[0], "/"), strings.Trim(parts[1], "/")
}
