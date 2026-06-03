package safefs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ReadOptions struct {
	MaxSize int64
}

type ReadOption func(*ReadOptions)

func WithMaxSize(n int64) ReadOption {
	return func(o *ReadOptions) {
		o.MaxSize = n
	}
}

func defaultReadOpts() *ReadOptions {
	return &ReadOptions{MaxSize: 0} // 0 = no limit
}

func ReadFile(path string, opts ...ReadOption) ([]byte, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("%s is a directory", path)
	}

	cfg := defaultReadOpts()
	for _, o := range opts {
		o(cfg)
	}
	if cfg.MaxSize > 0 && fi.Size() > cfg.MaxSize {
		return nil, fmt.Errorf("file too large: %d bytes (max %d)", fi.Size(), cfg.MaxSize)
	}

	return os.ReadFile(path)
}

func WriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	// atomic write: write to temp, then rename
	tmp, err := os.CreateTemp(dir, ".tmp.*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}

func SafePath(root, unsafe string) (string, error) {
	if filepath.IsAbs(unsafe) {
		return "", errors.New("path traversal: absolute path not allowed")
	}
	cleaned := filepath.Clean(unsafe)
	if strings.HasPrefix(cleaned, "..") {
		return "", errors.New("path traversal: '..' not allowed")
	}
	full := filepath.Join(root, cleaned)
	// verify no traversal after symlink resolution
	absRoot, _ := filepath.Abs(root)
	absFull, _ := filepath.Abs(full)
	if !strings.HasPrefix(absFull, absRoot) {
		return "", errors.New("path traversal: resolved path outside root")
	}
	return full, nil
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
