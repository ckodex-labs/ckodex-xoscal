//go:build !unix

package resilience

import (
	"path/filepath"
	"sync"
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

// WithDirLock coordinates in-process goroutines on non-Unix platforms.
func WithDirLock(dir string, exclusive bool, fn func() error) error {
	cleanDir := filepath.Clean(dir)
	if absDir, err := filepath.Abs(cleanDir); err == nil {
		cleanDir = absDir
	}
	entry := acquireInProcessDirLock(cleanDir, exclusive)
	defer releaseInProcessDirLock(cleanDir, entry, exclusive)
	return fn()
}
