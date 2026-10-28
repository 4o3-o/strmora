package emby

// Synthetic identities remain stable across restarts. No private library is bundled.
func catalog() []Item {
	return []Item{
		{ID: "movies", Name: "P0 Movies", Type: "CollectionFolder", IsFolder: true},
		{ID: "tv", Name: "P0 TV", Type: "CollectionFolder", IsFolder: true},
		{ID: "movie-p0", Name: "P0 Movie", Type: "Movie", ParentID: "movies"},
		{ID: "movie-versions", Name: "P0 Movie Versions", Type: "Movie", ParentID: "movies"},
		{ID: "series-p0", Name: "P0 Series", Type: "Series", ParentID: "tv", IsFolder: true},
		{ID: "season-1", Name: "Season 01", Type: "Season", ParentID: "series-p0", SeriesID: "series-p0", IsFolder: true, IndexNumber: 1},
		{ID: "season-2", Name: "1997", Type: "Season", ParentID: "series-p0", SeriesID: "series-p0", IsFolder: true, IndexNumber: 2},
		{ID: "episode-1", Name: "P0 Episode 1", Type: "Episode", ParentID: "season-1", SeriesID: "series-p0", SeasonID: "season-1", IndexNumber: 1, ParentIndexNumber: 1},
		{ID: "episode-2", Name: "P0 Episode 2", Type: "Episode", ParentID: "season-2", SeriesID: "series-p0", SeasonID: "season-2", IndexNumber: 1, ParentIndexNumber: 2},
	}
}
