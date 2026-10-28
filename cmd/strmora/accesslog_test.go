package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAccessLogOmitsCredentialsAndRetainsStatus(t *testing.T) {
	var out bytes.Buffer
	h := accessLog(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusFound) }), log.New(&out, "", 0))
	r := httptest.NewRequest("POST", "http://localhost/emby/Users/AuthenticateByName?api_key=secret-token", strings.NewReader(`{"Pw":"secret-password"}`))
	r.Header.Set("Authorization", "secret-header")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 302 {
		t.Fatal(w.Code)
	}
	if strings.Contains(out.String(), "secret-") || strings.Contains(out.String(), "api_key") {
		t.Fatalf("credential leaked in log: %s", out.String())
	}
	var row struct {
		Method, Path string
		Status       int
	}
	if err := json.Unmarshal(out.Bytes(), &row); err != nil {
		t.Fatal(err)
	}
	if row.Method != "POST" || row.Path != "/emby/Users/AuthenticateByName" || row.Status != 302 {
		t.Fatalf("unexpected log: %+v", row)
	}
}
func TestAccessLogCapturesImplicitOK(t *testing.T) {
	var out bytes.Buffer
	h := accessLog(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) }), log.New(&out, "", 0))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/healthz", nil))
	if !strings.Contains(out.String(), `"status":200`) {
		t.Fatal(out.String())
	}
}
