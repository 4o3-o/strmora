package sources

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type LocalSource struct {
	cfg Config
}

func NewLocal(cfg Config) (*LocalSource, error) {
	if cfg.RootPath == "" {
		return nil, fmt.Errorf("local source root path is required")
	}
	abs, err := filepath.Abs(cfg.RootPath)
	if err != nil {
		return nil, err
	}
	cfg.RootPath = abs
	return &LocalSource{cfg: cfg}, nil
}

func (s *LocalSource) ID() string { return s.cfg.ID }
func (s *LocalSource) Type() Type { return TypeLocal }
func (s *LocalSource) Capabilities() Capabilities {
	return Capabilities{
		ReadOnly:            s.cfg.ReadOnly,
		SupportsBatchMove:   false,
		SupportsBatchRename: false,
	}
}

func (s *LocalSource) resolve(relPath string) (string, error) {
	cleanRel := filepath.Clean(filepath.FromSlash(relPath))
	if strings.HasPrefix(cleanRel, "..") {
		return "", fmt.Errorf("path traversal forbidden")
	}
	target := filepath.Join(s.cfg.RootPath, cleanRel)
	if target != s.cfg.RootPath && !strings.HasPrefix(target, s.cfg.RootPath+string(filepath.Separator)) {
		return "", fmt.Errorf("path traversal forbidden")
	}
	return target, nil
}

func (s *LocalSource) Ping(ctx context.Context) error {
	fi, err := os.Stat(s.cfg.RootPath)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("root path is not a directory")
	}
	return nil
}

func (s *LocalSource) List(ctx context.Context, relPath string) ([]Entry, error) {
	target, err := s.resolve(relPath)
	if err != nil {
		return nil, err
	}
	dirEntries, err := os.ReadDir(target)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(dirEntries))
	for _, de := range dirEntries {
		info, err := de.Info()
		if err != nil {
			continue
		}
		itemRel := filepath.ToSlash(filepath.Join(relPath, de.Name()))
		out = append(out, Entry{
			Name:     de.Name(),
			Path:     itemRel,
			IsDir:    de.IsDir(),
			Size:     info.Size(),
			Modified: info.ModTime(),
		})
	}
	return out, nil
}

func (s *LocalSource) Open(ctx context.Context, relPath string) (io.ReadCloser, Entry, error) {
	target, err := s.resolve(relPath)
	if err != nil {
		return nil, Entry{}, err
	}
	f, err := os.Open(target)
	if err != nil {
		return nil, Entry{}, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, Entry{}, err
	}
	return f, Entry{
		Name:     info.Name(),
		Path:     relPath,
		IsDir:    info.IsDir(),
		Size:     info.Size(),
		Modified: info.ModTime(),
	}, nil
}
