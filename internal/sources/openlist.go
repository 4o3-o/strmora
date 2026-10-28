package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type OpenListSource struct {
	cfg    Config
	client *http.Client
}

func NewOpenList(cfg Config) (*OpenListSource, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("openlist endpoint is required")
	}
	u, err := url.Parse(cfg.Endpoint)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid openlist endpoint URL")
	}
	return &OpenListSource{
		cfg: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (s *OpenListSource) ID() string { return s.cfg.ID }
func (s *OpenListSource) Type() Type { return TypeOpenList }
func (s *OpenListSource) Capabilities() Capabilities {
	return Capabilities{
		ReadOnly:            s.cfg.ReadOnly,
		SupportsBatchMove:   true,
		SupportsBatchRename: true,
	}
}

func (s *OpenListSource) fullPath(rel string) string {
	base := strings.TrimSpace(s.cfg.RootPath)
	if base == "" {
		base = "/"
	}
	clean := path.Clean("/" + strings.TrimPrefix(rel, "/"))
	if clean == "/" {
		return base
	}
	if base == "/" {
		return clean
	}
	return path.Join(base, clean)
}

func (s *OpenListSource) Ping(ctx context.Context) error {
	_, err := s.List(ctx, "")
	return err
}

func (s *OpenListSource) List(ctx context.Context, relPath string) ([]Entry, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"path":     s.fullPath(relPath),
		"page":     1,
		"per_page": 0,
	})

	apiURL := strings.TrimRight(s.cfg.Endpoint, "/") + "/api/fs/list"
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.Token)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openlist api list returned %d", resp.StatusCode)
	}

	var res struct {
		Code int    `json:"code"`
		Msg  string `json:"message"`
		Data struct {
			Content []struct {
				Name     string `json:"name"`
				Size     int64  `json:"size"`
				IsDir    bool   `json:"is_dir"`
				Modified string `json:"modified"`
			} `json:"content"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	if res.Code != 200 {
		return nil, fmt.Errorf("openlist list error: %s (%d)", res.Msg, res.Code)
	}

	out := make([]Entry, 0, len(res.Data.Content))
	for _, it := range res.Data.Content {
		t, _ := time.Parse(time.RFC3339, it.Modified)
		out = append(out, Entry{
			Name:     it.Name,
			Path:     path.Join(relPath, it.Name),
			IsDir:    it.IsDir,
			Size:     it.Size,
			Modified: t,
		})
	}
	return out, nil
}

func (s *OpenListSource) Open(ctx context.Context, relPath string) (io.ReadCloser, Entry, error) {
	full := s.fullPath(relPath)
	downloadURL := strings.TrimRight(s.cfg.Endpoint, "/") + "/d" + full
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return nil, Entry{}, err
	}
	if s.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.Token)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, Entry{}, err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		resp.Body.Close()
		return nil, Entry{}, fmt.Errorf("openlist download returned %d", resp.StatusCode)
	}

	return resp.Body, Entry{
		Name:     path.Base(relPath),
		Path:     relPath,
		Size:     resp.ContentLength,
		Modified: time.Now(),
	}, nil
}
