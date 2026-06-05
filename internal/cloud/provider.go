package cloud

import (
	"fmt"
	"time"
)

// Provider abstracts cloud backup storage backends.
type Provider interface {
	// Name returns the provider identifier (e.g., "github-gist", "codeberg").
	Name() string

	// Push uploads a backup archive to the cloud backend.
	// Returns a provider-specific backup ID for later retrieval.
	Push(archive []byte, meta PushMeta) (id string, err error)

	// Pull downloads a backup archive by its provider-specific ID.
	Pull(id string) (archive []byte, err error)

	// List returns metadata for all backups stored by this provider,
	// in reverse chronological order (newest first).
	List() ([]BackupMeta, error)
}

// PushMeta carries context for a push operation.
type PushMeta struct {
	BackupID  string    // local backup ID (timestamp)
	CreatedAt time.Time
	Hostname  string
	OS        string
	Agents    []string // adapter names included
}

// BackupMeta describes a stored backup without its content.
type BackupMeta struct {
	ID        string    // provider-specific ID
	BackupID  string    // original local backup ID
	CreatedAt time.Time
	Hostname  string
	Size      int64
	URL       string // human-readable link (gist URL, repo URL, etc.)
}

// ProviderRegistry routes commands to named providers.
type ProviderRegistry struct {
	providers   map[string]Provider
	defaultName string
}

// NewProviderRegistry creates a new, empty ProviderRegistry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry under its Name().
// Returns an error if a provider with the same name already exists,
// or if the provider is nil.
func (r *ProviderRegistry) Register(p Provider) error {
	if p == nil {
		return fmt.Errorf("register provider: provider is nil")
	}
	name := p.Name()
	if name == "" {
		return fmt.Errorf("register provider: name is empty")
	}
	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("register provider: %q already registered", name)
	}
	r.providers[name] = p
	return nil
}

// Get returns the provider registered under the given name.
// If name is empty, returns the default provider.
// Returns an error if no provider is found.
func (r *ProviderRegistry) Get(name string) (Provider, error) {
	if name == "" {
		name = r.defaultName
	}
	if name == "" {
		return nil, fmt.Errorf("get provider: no provider name specified and no default set")
	}
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %q", name)
	}
	return p, nil
}

// SetDefault sets the default provider name. The provider must already
// be registered via Register; if it is not, Get("") will fail.
func (r *ProviderRegistry) SetDefault(name string) {
	r.defaultName = name
}
