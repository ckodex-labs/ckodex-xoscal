//go:build !unix

package resilience

import (
	"path/filepath"
	"sync"
)

var (
	dirLockMapMu sync.Mutex
	dirLocks     = make(map[string]*sync.RWMutex)
)

func getInProcessDirLock(cleanDir string) *sync.RWMutex {
	dirLockMapMu.Lock()
	defer dirLockMapMu.Unlock()
	if mu, ok := dirLocks[cleanDir]; ok {
		return mu
	}
	mu := &sync.RWMutex{}
	dirLocks[cleanDir] = mu
	return mu
}

// WithDirLock coordinates in-process goroutines on non-Unix platforms.
func WithDirLock(dir string, exclusive bool, fn func() error) error {
	cleanDir := filepath.Clean(dir)
	if absDir, err := filepath.Abs(cleanDir); err == nil {
		cleanDir = absDir
	}
	procMu := getInProcessDirLock(cleanDir)
	if exclusive {
		procMu.Lock()
		defer procMu.Unlock()
	} else {
		procMu.RLock()
		defer procMu.RUnlock()
	}
	return fn()
}
