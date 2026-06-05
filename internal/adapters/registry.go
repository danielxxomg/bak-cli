package adapters

import (
	"fmt"
	"sync"
)

// DetectedAdapter pairs an adapter with its detection result.
type DetectedAdapter struct {
	Adapter   Adapter
	ConfigDir string
}

// Registry holds all known adapters and supports auto-discovery.
// It is safe for concurrent use.
type Registry struct {
	mu       sync.RWMutex
	adapters map[string]Adapter
}

// NewRegistry returns an empty adapter registry.
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]Adapter),
	}
}

// Register adds an adapter to the registry. Returns an error if an
// adapter with the same name is already registered.
func (r *Registry) Register(a Adapter) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := a.Name()
	if _, exists := r.adapters[name]; exists {
		return fmt.Errorf("adapter %q already registered", name)
	}
	r.adapters[name] = a
	return nil
}

// DetectAll runs Detect() on every registered adapter and returns only
// those that report as installed. This is the auto-discovery mechanism.
func (r *Registry) DetectAll(homeDir string) []DetectedAdapter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []DetectedAdapter
	for _, a := range r.adapters {
		installed, configDir, err := a.Detect(homeDir)
		if err != nil || !installed {
			continue
		}
		results = append(results, DetectedAdapter{
			Adapter:   a,
			ConfigDir: configDir,
		})
	}
	return results
}

// Get returns the adapter registered under name, or false if not found.
func (r *Registry) Get(name string) (Adapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	a, ok := r.adapters[name]
	return a, ok
}

// GetByName is an alias for Get for caller convenience.
func (r *Registry) GetByName(name string) (Adapter, bool) {
	return r.Get(name)
}

// All returns a slice of every registered adapter.
func (r *Registry) All() []Adapter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Adapter, 0, len(r.adapters))
	for _, a := range r.adapters {
		result = append(result, a)
	}
	return result
}

// List returns the names of all registered adapters.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}
	return names
}
