// Package version exposes build metadata injected at link time via -ldflags -X.
package version

import (
	"encoding/json"
	"net/http"
	"runtime"
	"runtime/debug"
)

// Set via -ldflags "-X github.com/stanterprise/observer/pkg/version.Version=...".
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// Info describes the running build.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
	GoVersion string `json:"goVersion"`
}

// Get returns build info, falling back to Go VCS stamping for local builds without ldflags.
func Get() Info {
	info := Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
	}
	if info.Commit != "unknown" && info.BuildDate != "unknown" {
		return info
	}
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return info
	}
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			if info.Commit == "unknown" && s.Value != "" {
				info.Commit = s.Value
			}
		case "vcs.time":
			if info.BuildDate == "unknown" && s.Value != "" {
				info.BuildDate = s.Value
			}
		}
	}
	return info
}

// Handler serves Get() as JSON.
func Handler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(Get())
}
