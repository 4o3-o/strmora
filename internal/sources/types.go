package sources

import (
	"context"
	"io"
	"time"
)

type Type string

const (
	TypeLocal    Type = "local"
	TypeOpenList Type = "openlist"
	TypeWebDAV   Type = "webdav"
)

type Config struct {
	ID       string `json:"id"`
	Type     Type   `json:"type"`
	Name     string `json:"name"`
	Endpoint string `json:"endpoint"`
	RootPath string `json:"root_path"`
	Token    string `json:"token,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	ReadOnly bool   `json:"read_only"`
}

type Capabilities struct {
	ReadOnly            bool `json:"read_only"`
	SupportsBatchMove   bool `json:"supports_batch_move"`
	SupportsBatchRename bool `json:"supports_batch_rename"`
}

type Entry struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	IsDir    bool      `json:"is_dir"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

type Source interface {
	ID() string
	Type() Type
	Capabilities() Capabilities
	Ping(ctx context.Context) error
	List(ctx context.Context, relPath string) ([]Entry, error)
	Open(ctx context.Context, relPath string) (io.ReadCloser, Entry, error)
}
