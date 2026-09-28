package managed

import (
	"context"
	"time"
)

// Store keeps the managed state of one playbook and its lock: a file next
// to the playbook, or an object in an S3 bucket shared by everyone who
// applies it
type Store interface {
	Load(ctx context.Context) (*State, error)
	Save(ctx context.Context, s *State) error
	// Lock waits up to timeout; a held lock is a *LockedError
	Lock(ctx context.Context, operation string, timeout time.Duration) (Unlocker, error)
	// ForceUnlock removes the lock with this ID (a run that died)
	ForceUnlock(ctx context.Context, id string) error
	// String names the state for messages
	String() string
}

// Unlocker releases a held lock
type Unlocker interface {
	Release() error
}

// FileStore is the state file .onigirazu/<playbook>.state.json
type FileStore struct {
	Path string
}

// NewFileStore is the file store of a playbook
func NewFileStore(playbook string) *FileStore {
	return &FileStore{Path: Path(playbook)}
}

// Load implements Store
func (f *FileStore) Load(context.Context) (*State, error) { return Load(f.Path) }

// Save implements Store
func (f *FileStore) Save(_ context.Context, s *State) error { return s.Save(f.Path) }

// Lock implements Store
func (f *FileStore) Lock(_ context.Context, operation string, timeout time.Duration) (Unlocker, error) {
	return AcquireLock(f.Path, operation, timeout)
}

// ForceUnlock implements Store
func (f *FileStore) ForceUnlock(_ context.Context, id string) error { return ForceUnlock(f.Path, id) }

func (f *FileStore) String() string { return f.Path }
