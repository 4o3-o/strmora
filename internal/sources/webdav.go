package sources

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

type WebDAVSource struct {
	cfg    Config
	client *http.Client
}

func NewWebDAV(cfg Config) (*WebDAVSource, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("webdav endpoint is required")
	}
	u, err := url.Parse(cfg.Endpoint)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid webdav endpoint URL")
	}
	return &WebDAVSource{
		cfg: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (s *WebDAVSource) ID() string { return s.cfg.ID }
func (s *WebDAVSource) Type() Type { return TypeWebDAV }
func (s *WebDAVSource) Capabilities() Capabilities {
	return Capabilities{
		ReadOnly:            s.cfg.ReadOnly,
		SupportsBatchMove:   false,
		SupportsBatchRename: false,
	}
}

type multistatus struct {
	Responses []davResponse `xml:"response"`
}

type davResponse struct {
	Href     string   `xml:"href"`
	Propstat propstat `xml:"propstat"`
}

type propstat struct {
	Prop   prop   `xml:"prop"`
	Status string `xml:"status"`
}

type prop struct {
	ResourceType  resourceType `xml:"resourcetype"`
	ContentLength string       `xml:"getcontentlength"`
	LastModified  string       `xml:"getlastmodified"`
}

type resourceType struct {
	Collection *struct{} `xml:"collection"`
}

func (s *WebDAVSource) urlFor(relPath string) string {
	base := strings.TrimRight(s.cfg.Endpoint, "/")
	if relPath == "" {
		return base + "/"
	}
	return base + "/" + strings.TrimLeft(relPath, "/")
}

func (s *WebDAVSource) Ping(ctx context.Context) error {
	_, err := s.List(ctx, "")
	return err
}

func (s *WebDAVSource) List(ctx context.Context, relPath string) ([]Entry, error) {
	targetURL := s.urlFor(relPath)
	req, err := http.NewRequestWithContext(ctx, "PROPFIND", targetURL, strings.NewReader(`<?xml version="1.0" encoding="utf-8" ?>
<D:propfind xmlns:D="DAV:">
  <D:prop>
    <D:resourcetype/>
    <D:getcontentlength/>
    <D:getlastmodified/>
  </D:prop>
</D:propfind>`))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	if s.cfg.Username != "" || s.cfg.Password != "" {
		req.SetBasicAuth(s.cfg.Username, s.cfg.Password)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("webdav propfind returned status %d", resp.StatusCode)
	}

	var ms multistatus
	if err := xml.NewDecoder(resp.Body).Decode(&ms); err != nil {
		return nil, err
	}

	out := make([]Entry, 0, len(ms.Responses))
	uTarget, _ := url.Parse(targetURL)
	targetCleanPath := strings.TrimRight(uTarget.Path, "/")

	for _, r := range ms.Responses {
		rHref, err := url.PathUnescape(r.Href)
		if err != nil {
			rHref = r.Href
		}
		rClean := strings.TrimRight(rHref, "/")
		if rClean == targetCleanPath || rClean == "" {
			continue // skip self
		}

		name := path.Base(rClean)
		isDir := r.Propstat.Prop.ResourceType.Collection != nil
		size, _ := strconv.ParseInt(r.Propstat.Prop.ContentLength, 10, 64)
		modTime, _ := http.ParseTime(r.Propstat.Prop.LastModified)

		out = append(out, Entry{
			Name:     name,
			Path:     path.Join(relPath, name),
			IsDir:    isDir,
			Size:     size,
			Modified: modTime,
		})
	}
	return out, nil
}

func (s *WebDAVSource) Open(ctx context.Context, relPath string) (io.ReadCloser, Entry, error) {
	targetURL := s.urlFor(relPath)
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return nil, Entry{}, err
	}
	if s.cfg.Username != "" || s.cfg.Password != "" {
		req.SetBasicAuth(s.cfg.Username, s.cfg.Password)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, Entry{}, err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		resp.Body.Close()
		return nil, Entry{}, fmt.Errorf("webdav get returned status %d", resp.StatusCode)
	}

	return resp.Body, Entry{
		Name:     path.Base(relPath),
		Path:     relPath,
		Size:     resp.ContentLength,
		Modified: time.Now(),
	}, nil
}
