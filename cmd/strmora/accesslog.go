package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

type recordedResponse struct {
	http.ResponseWriter
	status int
}

func (w *recordedResponse) WriteHeader(code int) {
	if w.status != 0 {
		return
	}
	if code < 200 {
		w.ResponseWriter.WriteHeader(code)
		return
	}
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
func (w *recordedResponse) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}
func (w *recordedResponse) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// accessLog deliberately excludes query strings, bodies, headers and redirect targets.
func accessLog(next http.Handler, logger *log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		path := r.URL.Path
		rw := &recordedResponse{ResponseWriter: w}
		next.ServeHTTP(rw, r)
		status := rw.status
		if status == 0 {
			status = http.StatusOK
		}
		b, _ := json.Marshal(struct {
			Time       string `json:"time"`
			Method     string `json:"method"`
			Path       string `json:"path"`
			Status     int    `json:"status"`
			DurationMS int64  `json:"duration_ms"`
			Client     string `json:"client"`
			Transport  string `json:"auth_transport"`
		}{time.Now().UTC().Format(time.RFC3339Nano), r.Method, path, status, time.Since(started).Milliseconds(), clientCategory(r), authTransport(r)})
		logger.Print(string(b))
	})
}

// Only fixed categories are emitted. No raw UA, device identifiers or credentials are logged.
func clientCategory(r *http.Request) string {
	value := strings.ToLower(r.UserAgent())
	for _, header := range []string{"Authorization", "X-Emby-Authorization"} {
		for _, part := range strings.Split(r.Header.Get(header), ",") {
			part = strings.TrimSpace(part)
			for _, prefix := range []string{"MediaBrowser ", "Emby "} {
				part = strings.TrimPrefix(part, prefix)
			}
			if strings.HasPrefix(strings.ToLower(part), "client=") {
				value += " " + strings.ToLower(strings.Trim(strings.SplitN(part, "=", 2)[1], "\""))
			}
		}
	}
	for _, candidate := range []struct{ needle, label string }{{"infuse", "Infuse"}, {"vidhub", "VidHub"}, {"popcorn", "Popcorn"}, {"爆米花", "Popcorn"}} {
		if strings.Contains(value, candidate.needle) {
			return candidate.label
		}
	}
	return "other"
}
func authTransport(r *http.Request) string {
	if r.Header.Get("X-Emby-Token") != "" || r.Header.Get("X-MediaBrowser-Token") != "" {
		return "header_token"
	}
	if r.URL.Query().Get("api_key") != "" {
		return "query_api"
	}
	if r.URL.Query().Get("X-Emby-Token") != "" {
		return "query_emby"
	}
	if r.Header.Get("Authorization") != "" || r.Header.Get("X-Emby-Authorization") != "" {
		return "authorization"
	}
	return "none"
}
