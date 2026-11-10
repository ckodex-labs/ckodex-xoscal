//go:build unix

package resilience

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
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

// WithDirLock coordinates both in-process goroutines (via sync.RWMutex) and separate OS processes
// (via advisory syscall.Flock on <dir>/.checkpoint.lock) before executing fn.
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
