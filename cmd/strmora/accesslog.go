package main

import (
	"encoding/json"
	"log"
	"net/http"
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
		}{time.Now().UTC().Format(time.RFC3339Nano), r.Method, path, status, time.Since(started).Milliseconds()})
		logger.Print(string(b))
	})
}
