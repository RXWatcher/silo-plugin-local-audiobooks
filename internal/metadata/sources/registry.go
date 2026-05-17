package sources

import "sync"

// Registry maps source IDs to constructed Source impls. Populated at startup
// and read concurrently by the metadata gRPC handlers / enrichment worker, so
// access is mutex-guarded (a reconfigure that re-Registers must not race
// concurrent ForID/All reads).
type Registry struct {
	mu      sync.RWMutex
	sources map[string]Source
}

// NewRegistry constructs an empty registry.
func NewRegistry() *Registry {
	return &Registry{sources: map[string]Source{}}
}

// Register adds a source. Last registration wins for a given ID.
func (r *Registry) Register(s Source) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sources[s.ID()] = s
}

// ForID returns the source for ID, or nil if not registered.
func (r *Registry) ForID(id string) Source {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sources[id]
}

// All returns the registered sources in unspecified order.
func (r *Registry) All() []Source {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Source, 0, len(r.sources))
	for _, s := range r.sources {
		out = append(out, s)
	}
	return out
}
