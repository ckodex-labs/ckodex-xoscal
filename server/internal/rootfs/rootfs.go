// Package rootfs reads derived paths through an operator-selected directory handle.
// Relative symlinks may resolve within that directory; absolute links are rejected
// by os.Root, including links whose targets happen to be inside the directory.
package rootfs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ReadFile reads a regular file without resolving a derived path outside root.
func ReadFile(root *os.Root, name string) ([]byte, error) {
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err == nil && !info.Mode().IsRegular() {
		err = fmt.Errorf("%s is not a regular file", name)
	}
	if err != nil {
		return nil, errors.Join(err, f.Close())
	}
	data, err := io.ReadAll(f)
	return data, errors.Join(err, f.Close())
}

// ReadDir enumerates names through the same handle used to read their contents.
func ReadDir(root *os.Root, name string) ([]os.DirEntry, error) {
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	entries, err := f.ReadDir(-1)
	return entries, errors.Join(err, f.Close())
}

// OptionalStat permits absent metadata, but propagates forbidden, unreadable,
// or broken symbolic links, including links in intermediate path components.
func OptionalStat(root *os.Root, name string) (os.FileInfo, error) {
	info, err := root.Stat(name)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		return info, err
	}
	prefix := ""
	for _, part := range strings.Split(filepath.Clean(name), string(filepath.Separator)) {
		prefix = filepath.Join(prefix, part)
		entry, linkErr := root.Lstat(prefix)
		if errors.Is(linkErr, os.ErrNotExist) {
			return nil, nil
		}
		if linkErr != nil {
			return nil, linkErr
		}
		if entry.Mode()&os.ModeSymlink != 0 {
			if _, linkErr = root.Stat(prefix); linkErr != nil {
				return nil, linkErr
			}
		}
	}
	return nil, err
}

// OptionalReadFile returns nil for an absent optional metadata file.
func OptionalReadFile(root *os.Root, name string) ([]byte, error) {
	info, err := OptionalStat(root, name)
	if err != nil || info == nil {
		return nil, err
	}
	return ReadFile(root, name)
}
