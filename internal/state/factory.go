package state

import (
	"fmt"

	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
)

// BackendFactory creates state backend instances based on configuration
type BackendFactory struct {
	config *Config
}

// NewBackendFactory creates a new backend factory
func NewBackendFactory(cfg *Config) *BackendFactory {
	if cfg == nil {
		cfg = NewDefaultConfig()
	}
	return &BackendFactory{config: cfg}
}

// CreateBackend creates and returns a StateBackend instance based on configuration
func (f *BackendFactory) CreateBackend(stateFile string) (interfaces.StateBackend, error) {
	if err := f.config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	switch f.config.Backend {
	case BackendTypeFile:
		return NewFileBackend(f.config.File, stateFile)

	case BackendTypeSQLite:
		return NewSQLiteBackend(f.config.SQLite)

	case BackendTypeRemote:
		return nil, fmt.Errorf("remote backend not yet implemented")

	default:
		return nil, fmt.Errorf("unknown backend type: %s", f.config.Backend)
	}
}

// CreateFileBackend creates a file backend directly
func (f *BackendFactory) CreateFileBackend(stateFile string) (interfaces.StateBackend, error) {
	return NewFileBackend(f.config.File, stateFile)
}

// CreateSQLiteBackend creates a SQLite backend directly
func (f *BackendFactory) CreateSQLiteBackend() (interfaces.StateBackend, error) {
	return NewSQLiteBackend(f.config.SQLite)
}

// GetBackendType returns the currently configured backend type
func (f *BackendFactory) GetBackendType() BackendType {
	return f.config.Backend
}

// SetBackendType sets the backend type
func (f *BackendFactory) SetBackendType(backend BackendType) {
	f.config.Backend = backend
}
