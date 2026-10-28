// Package emby is the P0 protocol prototype, not a complete Emby implementation.
package emby

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

type Options struct{ Username, Password, MediaURL, DataDir string }
type Item struct {
	ID                string `json:"Id"`
	Name              string `json:"Name"`
	Type              string `json:"Type"`
	ParentID          string `json:"ParentId,omitempty"`
	IsFolder          bool   `json:"IsFolder"`
	IndexNumber       int    `json:"IndexNumber,omitempty"`
	ParentIndexNumber int    `json:"ParentIndexNumber,omitempty"`
	SeriesID          string `json:"SeriesId,omitempty"`
	SeasonID          string `json:"SeasonId,omitempty"`
	RunTimeTicks      int64  `json:"RunTimeTicks,omitempty"`
}
type Server struct {
	db    *sql.DB
	mux   *http.ServeMux
	opts  Options
	id    string
	items []Item
}

func New(db *sql.DB, o Options) (*Server, error) {
	if o.Username == "" || len(o.Password) < 12 {
		return nil, fmt.Errorf("P0 username required, password minimum 12 bytes")
	}
	if err := validURL(o.MediaURL); err != nil {
		return nil, err
	}
	s := &Server{db: db, opts: o, mux: http.NewServeMux(), items: catalog()}
	var hash string
	err := db.QueryRow("SELECT password_hash FROM users WHERE id='user-p0'").Scan(&hash)
	if err == sql.ErrNoRows {
		hash, err = hashPassword(o.Password)
		if err == nil {
			_, err = db.Exec("INSERT INTO users(id,name,password_hash) VALUES('user-p0',?,?)", o.Username, hash)
		}
	} else if err == nil {
		var name string
		err = db.QueryRow("SELECT name FROM users WHERE id='user-p0'").Scan(&name)
		if err == nil && (name != o.Username || !verifyPassword(hash, o.Password)) {
			return nil, fmt.Errorf("configured P0 credentials differ from stored user")
		}
	}
	if err != nil {
		return nil, err
	}
	_, err = db.Exec("INSERT OR IGNORE INTO settings(key,value) VALUES('server_id',?)", randomID())
	if err != nil {
		return nil, err
	}
	if err = db.QueryRow("SELECT value FROM settings WHERE key='server_id'").Scan(&s.id); err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Join(o.DataDir, "strm"), 0700); err != nil {
		return nil, err
	}
	for _, item := range s.items {
		if item.Type != "Movie" && item.Type != "Episode" {
			continue
		}
		if err = s.ensureSTRM(item.ID, o.MediaURL); err != nil {
			return nil, err
		}
		if item.ID == "movie-versions" {
			if err = s.ensureSTRM(item.ID+"-alternate", o.MediaURL); err != nil {
				return nil, err
			}
		}
	}
	s.routes()
	return s, nil
}
func (s *Server) ensureSTRM(id, target string) error {
	p := filepath.Join(s.opts.DataDir, "strm", id+".strm")
	f, e := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	_, e = f.WriteString(target + "\n")
	ce := f.Close()
	if e != nil {
		return e
	}
	return ce
}
func randomID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func tokenHash(t string) string { h := sha256.Sum256([]byte(t)); return hex.EncodeToString(h[:]) }
func hashPassword(p string) (string, error) {
	salt := make([]byte, 16)
	if _, e := rand.Read(salt); e != nil {
		return "", e
	}
	key := argon2.IDKey([]byte(p), salt, 2, 32768, 1, 32)
	return base64.RawStdEncoding.EncodeToString(salt) + ":" + base64.RawStdEncoding.EncodeToString(key), nil
}
func verifyPassword(h, p string) bool {
	parts := strings.Split(h, ":")
	if len(parts) != 2 {
		return false
	}
	salt, e := base64.RawStdEncoding.DecodeString(parts[0])
	if e != nil {
		return false
	}
	want, e := base64.RawStdEncoding.DecodeString(parts[1])
	if e != nil {
		return false
	}
	got := argon2.IDKey([]byte(p), salt, 2, 32768, 1, 32)
	return subtle.ConstantTimeCompare(got, want) == 1
}
func validURL(raw string) error {
	u, e := url.Parse(raw)
	if e != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.Fragment != "" || strings.ContainsAny(raw, "\r\n") {
		return fmt.Errorf("media URL must be absolute HTTP(S), without embedded credentials or fragment")
	}
	return nil
}
func respond(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, code int, msg string) {
	respond(w, code, map[string]any{"ResponseStatus": map[string]string{"ErrorCode": http.StatusText(code), "Message": msg}})
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if e := json.NewDecoder(r.Body).Decode(v); e != nil {
		fail(w, 400, "invalid JSON body")
		return false
	}
	return true
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.URL.Path == "/emby" || strings.HasPrefix(r.URL.Path, "/emby/") {
		r = r.Clone(r.Context())
		u := *r.URL
		r.URL = &u
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/emby")
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
	}
	public := (r.Method == "GET" && (r.URL.Path == "/healthz" || r.URL.Path == "/System/Info/Public")) || (r.Method == "POST" && r.URL.Path == "/Users/AuthenticateByName")
	if !public {
		token := r.Header.Get("X-Emby-Token")
		if token == "" {
			token = r.Header.Get("X-MediaBrowser-Token")
		}
		if token == "" {
			token = r.URL.Query().Get("api_key")
		}
		if token == "" {
			auth := r.Header.Get("Authorization")
			if auth == "" {
				auth = r.Header.Get("X-Emby-Authorization")
			}
			for _, v := range strings.Split(auth, ",") {
				v = strings.TrimSpace(v)
				if strings.HasPrefix(v, "MediaBrowser ") {
					v = strings.TrimPrefix(v, "MediaBrowser ")
				}
				if strings.HasPrefix(v, "Token=") {
					token = strings.Trim(strings.TrimPrefix(v, "Token="), "\"")
				}
			}
		}
		var user string
		if token == "" || s.db.QueryRow("SELECT user_id FROM devices WHERE token_hash=?", tokenHash(token)).Scan(&user) != nil {
			fail(w, 401, "valid device token required")
			return
		}
		if u := r.URL.Query().Get("UserId"); u != "" && u != user {
			fail(w, 403, "user mismatch")
			return
		}
	}
	s.mux.ServeHTTP(w, r)
}
func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if e := s.db.PingContext(r.Context()); e != nil {
			fail(w, 503, "database unavailable")
			return
		}
		respond(w, 200, map[string]any{"status": "ok", "phase": "P0"})
	})
	s.mux.HandleFunc("GET /System/Info/Public", s.info)
	s.mux.HandleFunc("GET /System/Info", s.info)
	s.mux.HandleFunc("POST /Users/AuthenticateByName", s.login)
	s.mux.HandleFunc("GET /Users/Me", s.currentUser)
	s.mux.HandleFunc("GET /Users/{user}", s.currentUser)
	s.mux.HandleFunc("GET /Users/{user}/Views", s.views)
	s.mux.HandleFunc("GET /Library/VirtualFolders", s.virtualFolders)
	s.mux.HandleFunc("GET /DisplayPreferences/{id}", s.displayPreferences)
	s.mux.HandleFunc("GET /Users/{user}/Items", s.list)
	s.mux.HandleFunc("GET /Users/{user}/Items/Latest", s.list)
	s.mux.HandleFunc("GET /Users/{user}/Items/Resume", s.list)
	s.mux.HandleFunc("GET /Users/{user}/Items/{id}", s.detail)
	s.mux.HandleFunc("GET /Items", s.list)
	s.mux.HandleFunc("GET /Items/{id}", s.detail)
	s.mux.HandleFunc("GET /Shows/{id}/Seasons", s.seasons)
	s.mux.HandleFunc("GET /Shows/{id}/Episodes", s.episodes)
	s.mux.HandleFunc("GET /Items/{id}/Images/{kind}", s.image)
	s.mux.HandleFunc("GET /Items/{id}/PlaybackInfo", s.playback)
	s.mux.HandleFunc("POST /Items/{id}/PlaybackInfo", s.playback)
	s.mux.HandleFunc("GET /Videos/{id}/stream", s.stream)
	s.mux.HandleFunc("GET /Videos/{id}/stream.mkv", s.stream)
	s.mux.HandleFunc("GET /Videos/{id}/{source}/Subtitles/{index}/Stream.srt", s.subtitle)
	for _, p := range []string{"/Sessions/Playing", "/Sessions/Playing/Progress", "/Sessions/Playing/Stopped"} {
		s.mux.HandleFunc("POST "+p, s.progress)
	}
	s.mux.HandleFunc("POST /Users/{user}/PlayedItems/{id}", func(w http.ResponseWriter, r *http.Request) { s.flag(w, r, "played", true) })
	s.mux.HandleFunc("DELETE /Users/{user}/PlayedItems/{id}", func(w http.ResponseWriter, r *http.Request) { s.flag(w, r, "played", false) })
	s.mux.HandleFunc("POST /Users/{user}/FavoriteItems/{id}", func(w http.ResponseWriter, r *http.Request) { s.flag(w, r, "favorite", true) })
	s.mux.HandleFunc("DELETE /Users/{user}/FavoriteItems/{id}", func(w http.ResponseWriter, r *http.Request) { s.flag(w, r, "favorite", false) })
	s.mux.HandleFunc("POST /Sessions/Logout", s.logout)
}
func (s *Server) info(w http.ResponseWriter, r *http.Request) {
	respond(w, 200, map[string]any{"Id": s.id, "ServerName": "Strmora P0", "Version": "0.0.1", "ProductName": "Strmora", "StartupWizardCompleted": true})
}
func (s *Server) currentUser(w http.ResponseWriter, r *http.Request) {
	if !s.allowed(w, r) {
		return
	}
	respond(w, http.StatusOK, s.user())
}
func (s *Server) user() map[string]any {
	return map[string]any{"Id": "user-p0", "Name": s.opts.Username, "ServerId": s.id, "HasPassword": true, "Policy": map[string]any{"IsAdministrator": false, "EnableAllFolders": true, "EnableMediaPlayback": true, "EnableAudioPlaybackTranscoding": false, "EnableVideoPlaybackTranscoding": false}}
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var b struct{ Username, Pw string }
	if !decode(w, r, &b) {
		return
	}
	var h string
	if e := s.db.QueryRow("SELECT password_hash FROM users WHERE name=?", b.Username).Scan(&h); e != nil || !verifyPassword(h, b.Pw) {
		fail(w, 401, "invalid credentials")
		return
	}
	token := randomID()
	device := r.Header.Get("X-Emby-Device-Id")
	if device == "" {
		device = "p0-client"
	}
	_, e := s.db.Exec("INSERT INTO devices(token_hash,user_id,device_id,created_at) VALUES(?,'user-p0',?,?)", tokenHash(token), device, time.Now().UTC().Format(time.RFC3339Nano))
	if e != nil {
		fail(w, 500, "cannot persist device")
		return
	}
	respond(w, 200, map[string]any{"User": s.user(), "AccessToken": token, "ServerId": s.id, "SessionInfo": map[string]any{"Id": device, "UserId": "user-p0"}})
}
func (s *Server) allowed(w http.ResponseWriter, r *http.Request) bool {
	if u := r.PathValue("user"); u != "" && u != "user-p0" {
		fail(w, 403, "user mismatch")
		return false
	}
	return true
}
func (s *Server) find(id string) (Item, bool) {
	for _, i := range s.items {
		if i.ID == id {
			return i, true
		}
	}
	return Item{}, false
}
func (s *Server) item(w http.ResponseWriter, r *http.Request) (Item, bool) {
	i, ok := s.find(r.PathValue("id"))
	if !ok {
		fail(w, 404, "item not found")
	}
	return i, ok
}
func (s *Server) data(id string) map[string]any {
	var pos int64
	var played, fav bool
	_ = s.db.QueryRow("SELECT position_ticks,played,favorite FROM user_data WHERE user_id='user-p0' AND item_id=?", id).Scan(&pos, &played, &fav)
	return map[string]any{"PlaybackPositionTicks": pos, "Played": played, "IsFavorite": fav, "Key": id}
}
func (s *Server) dto(i Item) map[string]any {
	b, _ := json.Marshal(i)
	v := map[string]any{}
	_ = json.Unmarshal(b, &v)
	v["ServerId"] = s.id
	v["UserData"] = s.data(i.ID)
	v["ImageTags"] = map[string]string{"Primary": "p0"}
	if i.Type == "Movie" || i.Type == "Episode" {
		v["MediaType"] = "Video"
		v["Container"] = "mkv"
		v["MediaSources"] = s.sources(i)
	}
	return v
}

// P0 has no saved display overrides, so reads expose the actual default settings.
func (s *Server) displayPreferences(w http.ResponseWriter, r *http.Request) {
	for _, key := range []string{"UserId", "userId", "userid"} {
		for _, user := range r.URL.Query()[key] {
			if user != "" && user != "user-p0" {
				fail(w, http.StatusForbidden, "user mismatch")
				return
			}
		}
	}
	client := r.URL.Query().Get("Client")
	if client == "" {
		client = r.URL.Query().Get("client")
	}
	if client == "" {
		client = "emby"
	}
	respond(w, http.StatusOK, map[string]any{"Id": r.PathValue("id"), "Client": client, "SortBy": "SortName", "SortOrder": "Ascending", "CustomPrefs": map[string]string{}})
}

// virtualFolders exposes logical libraries, never filesystem or source credentials.
func (s *Server) virtualFolders(w http.ResponseWriter, r *http.Request) {
	folders := []map[string]any{}
	for _, i := range s.items {
		if i.Type != "CollectionFolder" {
			continue
		}
		collectionType := "movies"
		if i.ID == "tv" {
			collectionType = "tvshows"
		}
		folders = append(folders, map[string]any{"Name": i.Name, "ItemId": i.ID, "Id": i.ID, "CollectionType": collectionType, "Locations": []string{}, "LibraryOptions": map[string]any{"ContentType": collectionType}})
	}
	respond(w, http.StatusOK, folders)
}
func (s *Server) views(w http.ResponseWriter, r *http.Request) {
	if !s.allowed(w, r) {
		return
	}
	out := []any{}
	for _, i := range s.items {
		if i.Type == "CollectionFolder" {
			v := s.dto(i)
			if i.ID == "movies" {
				v["CollectionType"] = "movies"
			} else {
				v["CollectionType"] = "tvshows"
			}
			out = append(out, v)
		}
	}
	respond(w, 200, map[string]any{"Items": out, "TotalRecordCount": len(out)})
}
func (s *Server) detail(w http.ResponseWriter, r *http.Request) {
	if !s.allowed(w, r) {
		return
	}
	if i, ok := s.item(w, r); ok {
		respond(w, 200, s.dto(i))
	}
}
func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	if !s.allowed(w, r) {
		return
	}
	s.listWhere(w, r, func(i Item) bool {
		q := r.URL.Query()
		if parent := q.Get("ParentId"); parent != "" && i.ParentID != parent && q.Get("Recursive") != "true" {
			return false
		}
		if parent := q.Get("ParentId"); parent != "" && q.Get("Recursive") == "true" && !s.descendant(i, parent) {
			return false
		}
		if types := q.Get("IncludeItemTypes"); types != "" && !strings.Contains(","+strings.ToLower(types)+",", ","+strings.ToLower(i.Type)+",") {
			return false
		}
		if !strings.Contains(strings.ToLower(i.Name), strings.ToLower(q.Get("SearchTerm"))) {
			return false
		}
		if strings.HasSuffix(r.URL.Path, "/Resume") {
			d := s.data(i.ID)
			return d["PlaybackPositionTicks"].(int64) > 0 && !d["Played"].(bool)
		}
		return i.Type != "CollectionFolder"
	})
}
func (s *Server) descendant(i Item, parent string) bool {
	for i.ParentID != "" {
		if i.ParentID == parent {
			return true
		}
		var ok bool
		i, ok = s.find(i.ParentID)
		if !ok {
			return false
		}
	}
	return false
}
func (s *Server) listWhere(w http.ResponseWriter, r *http.Request, predicate func(Item) bool) {
	start, limit := 0, 100
	var err error
	for k, p := range map[string]*int{"StartIndex": &start, "Limit": &limit} {
		if v := r.URL.Query().Get(k); v != "" {
			*p, err = strconv.Atoi(v)
			if err != nil || *p < 0 {
				fail(w, 400, "invalid pagination")
				return
			}
		}
	}
	if limit > 1000 {
		limit = 1000
	}
	all := []any{}
	for _, i := range s.items {
		if predicate(i) {
			all = append(all, s.dto(i))
		}
	}
	total := len(all)
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	if strings.HasSuffix(r.URL.Path, "/Latest") {
		respond(w, 200, all[start:end])
		return
	}
	respond(w, 200, map[string]any{"Items": all[start:end], "TotalRecordCount": total, "StartIndex": start})
}
func (s *Server) seasons(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.item(w, r); !ok {
		return
	}
	s.listWhere(w, r, func(i Item) bool { return i.Type == "Season" && i.SeriesID == r.PathValue("id") })
}
func (s *Server) episodes(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.item(w, r); !ok {
		return
	}
	s.listWhere(w, r, func(i Item) bool {
		return i.Type == "Episode" && i.SeriesID == r.PathValue("id") && (r.URL.Query().Get("SeasonId") == "" || i.SeasonID == r.URL.Query().Get("SeasonId"))
	})
}
func (s *Server) sources(i Item) []map[string]any {
	ids := []string{i.ID}
	if i.ID == "movie-versions" {
		ids = append(ids, i.ID+"-alternate")
	}
	out := []map[string]any{}
	for n, id := range ids {
		p := "/Videos/" + i.ID + "/stream?MediaSourceId=" + id + "-source"
		out = append(out, map[string]any{"Id": id + "-source", "Name": fmt.Sprintf("P0 version %d (technical properties unknown)", n+1), "Path": p, "Protocol": "Http", "Type": "Default", "Container": "mkv", "SupportsDirectPlay": false, "SupportsDirectStream": true, "SupportsTranscoding": false, "DirectStreamUrl": p, "MediaStreams": []any{map[string]any{"Index": 0, "Type": "Subtitle", "Codec": "srt", "Language": "eng", "IsExternal": true, "IsTextSubtitleStream": true, "DeliveryMethod": "External", "DeliveryUrl": "/Videos/" + i.ID + "/" + id + "-source/Subtitles/0/Stream.srt"}}})
	}
	return out
}
func (s *Server) playback(w http.ResponseWriter, r *http.Request) {
	i, ok := s.item(w, r)
	if !ok {
		return
	}
	if i.Type != "Movie" && i.Type != "Episode" {
		fail(w, 400, "item is not playable")
		return
	}
	respond(w, 200, map[string]any{"MediaSources": s.sources(i), "PlaySessionId": randomID()})
}
func (s *Server) sourceID(i Item, raw string) (string, bool) {
	if raw == "" || raw == i.ID+"-source" {
		return i.ID, true
	}
	if i.ID == "movie-versions" && raw == i.ID+"-alternate-source" {
		return i.ID + "-alternate", true
	}
	return "", false
}
func (s *Server) stream(w http.ResponseWriter, r *http.Request) {
	i, ok := s.item(w, r)
	if !ok {
		return
	}
	if i.Type != "Movie" && i.Type != "Episode" {
		fail(w, 404, "not playable")
		return
	}
	id, ok := s.sourceID(i, r.URL.Query().Get("MediaSourceId"))
	if !ok {
		fail(w, 404, "source not found")
		return
	}
	b, e := os.ReadFile(filepath.Join(s.opts.DataDir, "strm", id+".strm"))
	if e != nil {
		fail(w, 500, "cannot read STRM")
		return
	}
	target := strings.TrimSpace(string(b))
	if e = validURL(target); e != nil {
		fail(w, 500, "invalid STRM URL")
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
}
func (s *Server) subtitle(w http.ResponseWriter, r *http.Request) {
	i, ok := s.item(w, r)
	if !ok {
		return
	}
	_, ok = s.sourceID(i, r.PathValue("source"))
	if !ok || r.PathValue("index") != "0" {
		fail(w, 404, "subtitle not found")
		return
	}
	w.Header().Set("Content-Type", "application/x-subrip; charset=utf-8")
	http.ServeContent(w, r, "p0.srt", time.Time{}, strings.NewReader("1\n00:00:00,000 --> 00:00:05,000\nStrmora P0 external subtitle\n"))
}
func (s *Server) image(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.item(w, r); !ok {
		return
	}
	b, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAADElEQVR4nGP4//8/AAX+Av4N70a4AAAAAElFTkSuQmCC")
	w.Header().Set("Content-Type", "image/png")
	http.ServeContent(w, r, "placeholder.png", time.Time{}, strings.NewReader(string(b)))
}
func (s *Server) progress(w http.ResponseWriter, r *http.Request) {
	var b struct {
		ItemID   string `json:"ItemId"`
		Position *int64 `json:"PositionTicks"`
		Session  string `json:"PlaySessionId"`
	}
	if !decode(w, r, &b) {
		return
	}
	i, ok := s.find(b.ItemID)
	if !ok {
		fail(w, 404, "item not found")
		return
	}
	if b.Position == nil || *b.Position < 0 || i.IsFolder {
		fail(w, 400, "invalid position or item")
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	tx, e := s.db.Begin()
	if e != nil {
		fail(w, 500, "database unavailable")
		return
	}
	defer tx.Rollback()
	_, e = tx.Exec("INSERT INTO user_data(user_id,item_id,position_ticks,updated_at) VALUES('user-p0',?,?,?) ON CONFLICT(user_id,item_id) DO UPDATE SET position_ticks=excluded.position_ticks,updated_at=excluded.updated_at", b.ItemID, *b.Position, now)
	if e == nil {
		if b.Session == "" {
			b.Session = b.ItemID
		}
		_, e = tx.Exec("INSERT INTO playback_sessions(id,user_id,item_id,position_ticks,updated_at) VALUES(?,'user-p0',?,?,?) ON CONFLICT(id) DO UPDATE SET item_id=excluded.item_id,position_ticks=excluded.position_ticks,updated_at=excluded.updated_at", b.Session, b.ItemID, *b.Position, now)
	}
	if e == nil {
		e = tx.Commit()
	}
	if e != nil {
		fail(w, 500, "cannot save progress")
		return
	}
	w.WriteHeader(204)
}
func (s *Server) flag(w http.ResponseWriter, r *http.Request, column string, value bool) {
	if !s.allowed(w, r) {
		return
	}
	i, ok := s.item(w, r)
	if !ok {
		return
	}
	_, e := s.db.Exec("INSERT INTO user_data(user_id,item_id,"+column+") VALUES('user-p0',?,?) ON CONFLICT(user_id,item_id) DO UPDATE SET "+column+"=excluded."+column, i.ID, value)
	if e != nil {
		fail(w, 500, "cannot save state")
		return
	}
	respond(w, 200, s.data(i.ID))
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) { // P0 revokes all devices for this sole test user.
	if _, e := s.db.Exec("DELETE FROM devices WHERE user_id='user-p0'"); e != nil {
		fail(w, 500, "cannot revoke devices")
		return
	}
	w.WriteHeader(204)
}
