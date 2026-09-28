package managed

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/user"
	"path/filepath"
	"time"
)

// A lock file next to the state (<state>.lock) keeps two applies of one
// playbook apart. It is created exclusively, so it works on every platform;
// a lock left by a run that died is removed with state unlock and its ID,
// as terraform force-unlock does.

// LockInfo says who holds a lock
type LockInfo struct {
	ID        string    `json:"id"`
	Who       string    `json:"who"`
	PID       int       `json:"pid"`
	Operation string    `json:"operation"`
	Created   time.Time `json:"created"`
}

// Lock is a held state lock
type Lock struct {
	path string
	info LockInfo
}

// LockedError: another run holds the lock
type LockedError struct {
	Path string
	Info LockInfo
}

func (e *LockedError) Error() string {
	return fmt.Sprintf("managed state %s is locked by %s (pid %d, %s) since %s, lock ID %s; "+
		"if that run is gone: onigirazu state unlock PLAYBOOK %s",
		e.Path, e.Info.Who, e.Info.PID, e.Info.Operation, e.Info.Created.Format(time.RFC3339), e.Info.ID, e.Info.ID)
}

func lockPath(statePath string) string { return statePath + ".lock" }

// AcquireLock locks the state at statePath, waiting up to timeout for
// another run to finish
func AcquireLock(statePath, operation string, timeout time.Duration) (*Lock, error) {
	if err := os.MkdirAll(filepath.Dir(statePath), 0o700); err != nil {
		return nil, err
	}
	info := LockInfo{ID: newLockID(), Who: whoAmI(), PID: os.Getpid(), Operation: operation, Created: time.Now().UTC()}
	data, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(timeout)
	for {
		f, err := os.OpenFile(lockPath(statePath), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) // #nosec G304 -- the playbook's own state
		if err == nil {
			_, werr := f.Write(append(data, '\n'))
			cerr := f.Close()
			if werr != nil || cerr != nil {
				_ = os.Remove(lockPath(statePath))
				return nil, errors.Join(werr, cerr)
			}
			return &Lock{path: lockPath(statePath), info: info}, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		if time.Now().After(deadline) {
			held, rerr := ReadLock(statePath)
			if rerr != nil {
				return nil, fmt.Errorf("managed state %s is locked: %w", statePath, rerr)
			}
			return nil, &LockedError{Path: statePath, Info: held}
		}
		time.Sleep(time.Second)
	}
}

// Release removes the lock if it is still this run's
func (l *Lock) Release() error {
	if l == nil {
		return nil
	}
	data, err := os.ReadFile(l.path)
	if err != nil {
		return err
	}
	var held LockInfo
	if json.Unmarshal(data, &held) == nil && held.ID != l.info.ID {
		return fmt.Errorf("lock %s now belongs to %s", l.path, held.ID)
	}
	return os.Remove(l.path)
}

// ReadLock returns the holder of the state's lock
func ReadLock(statePath string) (LockInfo, error) {
	var info LockInfo
	data, err := os.ReadFile(lockPath(statePath)) // #nosec G304 -- the playbook's own state
	if err != nil {
		return info, err
	}
	err = json.Unmarshal(data, &info)
	return info, err
}

// ForceUnlock removes a lock left by a run that died; id must match
func ForceUnlock(statePath, id string) error {
	held, err := ReadLock(statePath)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("managed state %s is not locked", statePath)
	}
	if err != nil {
		return err
	}
	if held.ID != id {
		return fmt.Errorf("lock ID %s does not match the lock of %s (%s)", id, statePath, held.ID)
	}
	return os.Remove(lockPath(statePath))
}

func newLockID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func whoAmI() string {
	name := "unknown"
	if u, err := user.Current(); err == nil {
		name = u.Username
	}
	host, _ := os.Hostname()
	return name + "@" + host
}

// randomInt is a random number below n (backoff jitter)
func randomInt(n int64) int64 {
	v, err := rand.Int(rand.Reader, big.NewInt(n))
	if err != nil {
		return 0
	}
	return v.Int64()
}
