//go:build unix

package resilience

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

type dirLockEntry struct {
	mu   sync.RWMutex
	refs int
}

var (
	dirLocksMu sync.Mutex
	dirLocks   = make(map[string]*dirLockEntry)
)

func acquireInProcessDirLock(cleanDir string, exclusive bool) *dirLockEntry {
	dirLocksMu.Lock()
	entry, ok := dirLocks[cleanDir]
	if !ok {
		entry = &dirLockEntry{}
		dirLocks[cleanDir] = entry
	}
	entry.refs++
	dirLocksMu.Unlock()

	if exclusive {
		entry.mu.Lock()
	} else {
		entry.mu.RLock()
	}
	return entry
}

func releaseInProcessDirLock(cleanDir string, entry *dirLockEntry, exclusive bool) {
	if exclusive {
		entry.mu.Unlock()
	} else {
		entry.mu.RUnlock()
	}

	dirLocksMu.Lock()
	entry.refs--
	if entry.refs <= 0 && dirLocks[cleanDir] == entry {
		delete(dirLocks, cleanDir)
	}
	dirLocksMu.Unlock()
}

// ActiveDirLockCount returns the current number of active in-process directory lock entries.
func ActiveDirLockCount() int {
	dirLocksMu.Lock()
	defer dirLocksMu.Unlock()
	return len(dirLocks)
}

// WithDirLock coordinates both in-process goroutines (via reference-counted sync.RWMutex) and separate OS processes
// (via advisory syscall.Flock on <dir>/.checkpoint.lock) before executing fn.
func WithDirLock(dir string, exclusive bool, fn func() error) error {
	cleanDir := filepath.Clean(dir)
	if absDir, err := filepath.Abs(cleanDir); err == nil {
		cleanDir = absDir
	}

	entry := acquireInProcessDirLock(cleanDir, exclusive)
	defer releaseInProcessDirLock(cleanDir, entry, exclusive)

	if exclusive {
		if err := os.MkdirAll(cleanDir, 0o750); err != nil {
			return fmt.Errorf("create lock directory %s: %w", cleanDir, err)
		}
	} else {
		if _, err := os.Stat(cleanDir); err != nil {
			return fn()
		}
	}

	lockPath := filepath.Join(cleanDir, ".checkpoint.lock")
	// #nosec G304 -- lockPath is strictly ".checkpoint.lock" inside cleanDir.
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		if !exclusive && os.IsPermission(err) {
			return fn()
		}
		return fmt.Errorf("open directory lock %s: %w", lockPath, err)
	}
	defer func() { _ = lockFile.Close() }()

	how := syscall.LOCK_SH
	if exclusive {
		how = syscall.LOCK_EX
	}
	if err := syscall.Flock(int(lockFile.Fd()), how); err != nil {
		return fmt.Errorf("acquire flock on %s: %w", lockPath, err)
	}
	defer func() {
		_ = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
	}()

	return fn()
}
