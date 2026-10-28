package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type TMDBConfig struct {
	APIKey   string
	BaseURL  string
	Language string
}

type TMDBClient struct {
	cfg    TMDBConfig
	client *http.Client
}

type TMDBMovieResult struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	ReleaseDate string `json:"release_date"`
	PosterPath  string `json:"poster_path"`
	Overview    string `json:"overview"`
}

type TMDBEpisodeGroup struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Order       int    `json:"order"`
	GroupCount  int    `json:"group_count"`
	Description string `json:"description"`
}

type TMDBEpisodeGroupDetail struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Groups []struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Order    int    `json:"order"`
		Episodes []struct {
			ID            int    `json:"id"`
			Name          string `json:"name"`
			EpisodeNumber int    `json:"episode_number"`
			Order         int    `json:"order"`
		} `json:"episodes"`
	} `json:"groups"`
}

func NewTMDBClient(cfg TMDBConfig) *TMDBClient {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.themoviedb.org"
	}
	if cfg.Language == "" {
		cfg.Language = "zh-CN"
	}
	return &TMDBClient{
		cfg: cfg,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *TMDBClient) doRequest(ctx context.Context, endpoint string, q url.Values, v any) error {
	u := strings.TrimRight(c.cfg.BaseURL, "/") + endpoint
	if q == nil {
		q = url.Values{}
	}
	q.Set("api_key", c.cfg.APIKey)
	if q.Get("language") == "" {
		q.Set("language", c.cfg.Language)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u+"?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tmdb api returned status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(v)
}

func (c *TMDBClient) SearchMovie(ctx context.Context, query string, year int) ([]TMDBMovieResult, error) {
	q := url.Values{}
	q.Set("query", query)
	if year > 0 {
		q.Set("year", strconv.Itoa(year))
	}

	var res struct {
		Results []TMDBMovieResult `json:"results"`
	}
	err := c.doRequest(ctx, "/3/search/movie", q, &res)
	return res.Results, err
}

func (c *TMDBClient) GetTVEpisodeGroups(ctx context.Context, tvID int) ([]TMDBEpisodeGroup, error) {
	var res struct {
		Results []TMDBEpisodeGroup `json:"results"`
	}
	err := c.doRequest(ctx, fmt.Sprintf("/3/tv/%d/episode_groups", tvID), nil, &res)
	return res.Results, err
}

func (c *TMDBClient) GetTVEpisodeGroupDetail(ctx context.Context, groupID string) (TMDBEpisodeGroupDetail, error) {
	var detail TMDBEpisodeGroupDetail
	err := c.doRequest(ctx, fmt.Sprintf("/3/tv/episode_group/%s", groupID), nil, &detail)
	return detail, err
}
