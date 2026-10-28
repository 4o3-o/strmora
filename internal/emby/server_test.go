package emby

import (
	"bytes"
	"encoding/json"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"strmora/internal/database"
)

func fixture(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "p0.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	s, err := New(db, Options{Username: "tester", Password: "test-password-123", MediaURL: "https://media.example.invalid/d/movie%20name.mkv", DataDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	return s, dir
}
func call(s *Server, method, path, token, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		r.Header.Set("X-Emby-Token", token)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}
func login(t *testing.T, s *Server) string {
	t.Helper()
	w := call(s, "POST", "/emby/Users/AuthenticateByName", "", `{"Username":"tester","Pw":"test-password-123"}`)
	if w.Code != 200 {
		t.Fatalf("login: %d %s", w.Code, w.Body)
	}
	var v struct{ AccessToken string }
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil || v.AccessToken == "" {
		t.Fatal(w.Body)
	}
	return v.AccessToken
}
func TestProtocolAuthAndCatalog(t *testing.T) {
	s, _ := fixture(t)
	if w := call(s, "GET", "/Users/user-p0/Items", "", ""); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := call(s, "POST", "/Users/AuthenticateByName", "", `{"Username":"tester","Pw":"wrong"}`); w.Code != 401 {
		t.Fatal(w.Code)
	}
	token := login(t, s)
	for _, p := range []string{"/emby/System/Info/Public", "/Users/Me", "/Users/user-p0/Views", "/Users/user-p0/Items?SearchTerm=Movie&Limit=1", "/Shows/series-p0/Seasons", "/Shows/series-p0/Episodes?SeasonId=season-1", "/Items/movie-p0/PlaybackInfo", "/Users/user-p0/Items/movie-p0"} {
		w := call(s, "GET", p, token, "")
		if w.Code != 200 {
			t.Errorf("%s: %d %s", p, w.Code, w.Body)
		}
	}
	if w := call(s, "GET", "/Users/someone-else/Items", token, ""); w.Code != 403 {
		t.Fatal(w.Code)
	}
	if w := call(s, "GET", "/Items/movie-p0/PlaybackInfo?UserId=someone-else", token, ""); w.Code != 403 {
		t.Fatal(w.Code)
	}
	if w := call(s, "GET", "/Users/user-p0/Items?StartIndex=-1", token, ""); w.Code != 400 {
		t.Fatal(w.Code)
	}
}
func TestPlaybackReadsSTRMAndNeverFetchesVideo(t *testing.T) {
	s, dir := fixture(t)
	token := login(t, s)
	target := "https://other.example.invalid/d/%E4%B8%AD%E6%96%87.mkv"
	if err := os.WriteFile(filepath.Join(dir, "strm", "movie-p0.strm"), []byte(target+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	w := call(s, "POST", "/Items/movie-p0/PlaybackInfo", token, `{}`)
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte("/Videos/movie-p0/stream")) {
		t.Fatal(w.Body)
	}
	w = call(s, "GET", "/Videos/movie-p0/stream?MediaSourceId=movie-p0-source", token, "")
	if w.Code != 302 || w.Header().Get("Location") != target {
		t.Fatalf("%d %s", w.Code, w.Header())
	}
	if w = call(s, "GET", "/Videos/movie-p0/stream?MediaSourceId=unknown", token, ""); w.Code != 404 {
		t.Fatal(w.Code)
	}
	if w = call(s, "GET", "/Videos/movie-p0/stream", "", ""); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if err := os.WriteFile(filepath.Join(dir, "strm", "movie-p0.strm"), []byte("file:///etc/passwd"), 0600); err != nil {
		t.Fatal(err)
	}
	if w = call(s, "GET", "/Videos/movie-p0/stream", token, ""); w.Code != 500 {
		t.Fatal(w.Code)
	}
}
func TestProgressSubtitleAndImage(t *testing.T) {
	s, _ := fixture(t)
	token := login(t, s)
	for _, p := range []string{"/Sessions/Playing", "/Sessions/Playing/Progress", "/Sessions/Playing/Stopped"} {
		w := call(s, "POST", p, token, `{"ItemId":"episode-1","PositionTicks":123456789,"PlaySessionId":"test-session"}`)
		if w.Code != 204 {
			t.Fatal(w.Code, w.Body)
		}
	}
	w := call(s, "GET", "/Users/user-p0/Items/episode-1", token, "")
	if !bytes.Contains(w.Body.Bytes(), []byte(`"PlaybackPositionTicks":123456789`)) {
		t.Fatal(w.Body)
	}
	w = call(s, "GET", "/Videos/movie-p0/movie-p0-source/Subtitles/0/Stream.srt", token, "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Strmora P0") {
		t.Fatal(w.Code, w.Body)
	}
	w = call(s, "GET", "/Items/movie-p0/Images/Primary", token, "")
	if w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), "image/png") {
		t.Fatal(w.Code)
	}
	w = call(s, "POST", "/Sessions/Playing/Progress", token, `{"ItemId":"episode-1","PositionTicks":-1}`)
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
	w = call(s, "POST", "/Sessions/Playing/Progress", token, `{"ItemId":"missing","PositionTicks":1}`)
	if w.Code != 404 {
		t.Fatal(w.Code)
	}
}
func TestTokenAndProgressPersist(t *testing.T) {
	s, dir := fixture(t)
	token := login(t, s)
	call(s, "POST", "/Sessions/Playing/Progress", token, `{"ItemId":"movie-p0","PositionTicks":50}`)
	s2, err := New(s.db, Options{Username: "tester", Password: "test-password-123", MediaURL: "https://media.example.invalid/movie.mkv", DataDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	w := call(s2, "GET", "/Users/user-p0/Items/movie-p0", token, "")
	if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte(`"PlaybackPositionTicks":50`)) {
		t.Fatal(w.Code, w.Body)
	}
}

func TestPlaceholderIsDecodablePNG(t *testing.T) {
	s, _ := fixture(t)
	token := login(t, s)
	w := call(s, "GET", "/Items/movie-p0/Images/Primary", token, "")
	if _, err := png.Decode(bytes.NewReader(w.Body.Bytes())); err != nil {
		t.Fatalf("invalid image delivered to player: %v", err)
	}
}

func TestMissingPositionCannotEraseProgress(t *testing.T) {
	s, _ := fixture(t)
	token := login(t, s)
	call(s, "POST", "/Sessions/Playing/Progress", token, `{"ItemId":"movie-p0","PositionTicks":600000000}`)
	for _, path := range []string{"/Sessions/Playing", "/Sessions/Playing/Progress", "/Sessions/Playing/Stopped"} {
		for _, body := range []string{`{"ItemId":"movie-p0"}`, `{"ItemId":"movie-p0","PositionTicks":null}`} {
			if w := call(s, "POST", path, token, body); w.Code != 400 {
				t.Fatalf("%s accepts absent position: %d", path, w.Code)
			}
		}
	}
	w := call(s, "GET", "/Users/user-p0/Items/movie-p0", token, "")
	if !bytes.Contains(w.Body.Bytes(), []byte(`"PlaybackPositionTicks":600000000`)) {
		t.Fatalf("progress lost: %s", w.Body)
	}
	if w = call(s, "POST", "/Sessions/Playing/Progress", token, `{"ItemId":"movie-p0","PositionTicks":0}`); w.Code != 204 {
		t.Fatalf("explicit zero refused: %d", w.Code)
	}
}
