package metadata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTMDBClientSearchAndEpisodeGroups(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.URL.Query().Get("api_key")
		if apiKey != "test-tmdb-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/3/search/movie":
			json.NewEncoder(w).Encode(map[string]any{
				"results": []map[string]any{
					{
						"id":           589499,
						"title":        "欲火狂蜜",
						"release_date": "1995-01-01",
						"poster_path":  "/poster.jpg",
					},
				},
			})
		case "/3/tv/30983/episode_groups":
			json.NewEncoder(w).Encode(map[string]any{
				"results": []map[string]any{
					{
						"id":          "5af83d88c3a3682ac8007b28",
						"name":        "按年份排序",
						"order":       1,
						"group_count": 28,
					},
				},
			})
		case "/3/tv/episode_group/5af83d88c3a3682ac8007b28":
			json.NewEncoder(w).Encode(map[string]any{
				"id":   "5af83d88c3a3682ac8007b28",
				"name": "按年份排序",
				"groups": []map[string]any{
					{
						"id":    "grp-1996",
						"name":  "1996",
						"order": 1,
						"episodes": []map[string]any{
							{"id": 1, "name": "云霄飞车杀人事件", "episode_number": 1, "order": 0},
						},
					},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	client := NewTMDBClient(TMDBConfig{
		APIKey:   "test-tmdb-key",
		BaseURL:  ts.URL,
		Language: "zh-CN",
	})

	ctx := context.Background()

	// 1. Test Search Movie
	movies, err := client.SearchMovie(ctx, "欲火狂蜜", 1995)
	if err != nil {
		t.Fatalf("search movie failed: %v", err)
	}
	if len(movies) != 1 || movies[0].ID != 589499 {
		t.Fatalf("unexpected movies: %+v", movies)
	}

	// 2. Test TV Episode Groups
	groups, err := client.GetTVEpisodeGroups(ctx, 30983)
	if err != nil {
		t.Fatalf("get episode groups: %v", err)
	}
	if len(groups) != 1 || groups[0].ID != "5af83d88c3a3682ac8007b28" {
		t.Fatalf("unexpected groups: %+v", groups)
	}

	// 3. Test TV Episode Group Detail
	detail, err := client.GetTVEpisodeGroupDetail(ctx, "5af83d88c3a3682ac8007b28")
	if err != nil {
		t.Fatalf("get episode group detail: %v", err)
	}
	if len(detail.Groups) != 1 || detail.Groups[0].Name != "1996" {
		t.Fatalf("unexpected group detail: %+v", detail)
	}
}
