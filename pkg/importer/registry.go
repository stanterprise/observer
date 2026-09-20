package importer

import (
	"fmt"
	"sort"
	"sync"
)

type Registry struct {
	mu       sync.RWMutex
	adapters map[string]Importer
}

func NewRegistry(adapters ...Importer) (*Registry, error) {
	r := &Registry{adapters: make(map[string]Importer)}
	for _, adapter := range adapters {
		if err := r.Register(adapter); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *Registry) Register(adapter Importer) error {
	if adapter == nil {
		return fmt.Errorf("importer is nil")
	}
	t := adapter.ReportType()
	if t.Producer == "" || t.Format == "" || t.Version == "" {
		return fmt.Errorf("importer report type is incomplete")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.adapters[t.Key()]; exists {
		return fmt.Errorf("importer already registered for %s", t.Key())
	}
	r.adapters[t.Key()] = adapter
	return nil
}

func (r *Registry) Lookup(t ReportType) (Importer, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	adapter, ok := r.adapters[t.Key()]
	return adapter, ok
}

func (r *Registry) SupportedVersions(producer, format string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	versions := make([]string, 0)
	for _, adapter := range r.adapters {
		t := adapter.ReportType()
		if t.Producer == producer && t.Format == format {
			versions = append(versions, t.Version)
		}
	}
	sort.Strings(versions)
	return versions
}

func (r *Registry) Capabilities() []Capability {
	r.mu.RLock()
	defer r.mu.RUnlock()
	capabilities := make([]Capability, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		t := adapter.ReportType()
		capabilities = append(capabilities, Capability{
			ReportType: t,
			UploadPath: "/v1/imports/" + t.Producer + "/" + t.Format + "/" + t.Version,
			Limits:     adapter.Limits(),
		})
	}
	sort.Slice(capabilities, func(i, j int) bool {
		return capabilities[i].Key() < capabilities[j].Key()
	})
	return capabilities
}
