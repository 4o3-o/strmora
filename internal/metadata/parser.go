package metadata

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type MediaType string

const (
	TypeMovie  MediaType = "movie"
	TypeSeries MediaType = "series"
	TypeOther  MediaType = "other"
)

type ParsedInfo struct {
	Title      string
	Year       int
	Season     int
	Episode    int
	Part       string
	TMDBID     int
	Resolution string
	MediaType  MediaType
}

var (
	tmdbIDRegexp   = regexp.MustCompile(`(?i)tmdb-(\d+)`)
	yearRegexp     = regexp.MustCompile(`[\(\[\.\s](\d{4})[\)\]\.\s]?`)
	seasonEpRegexp = regexp.MustCompile(`(?i)(?:^|[\.\s_])S(\d{1,2})E(\d{1,3})(?:-E(\d{1,3}))?`)
	partRegexp     = regexp.MustCompile(`(?i)[\.\-_\s](part\d+|cd\d+)`)
)

func ParseName(raw string, isDir bool) ParsedInfo {
	var info ParsedInfo
	clean := raw
	if !isDir {
		clean = strings.TrimSuffix(clean, filepath.Ext(clean))
	}

	// 1. Extract explicit tmdb ID if present
	if m := tmdbIDRegexp.FindStringSubmatch(clean); len(m) > 1 {
		info.TMDBID, _ = strconv.Atoi(m[1])
		clean = tmdbIDRegexp.ReplaceAllString(clean, "")
	}

	// 2. Extract year first before trimming by season/ep
	if m := yearRegexp.FindStringSubmatch(clean); len(m) > 1 {
		info.Year, _ = strconv.Atoi(m[1])
	}

	// 3. Extract part
	if m := partRegexp.FindStringSubmatch(clean); len(m) > 1 {
		info.Part = strings.ToLower(m[1])
		clean = partRegexp.ReplaceAllString(clean, "")
	}

	// 4. Extract season / episode
	if m := seasonEpRegexp.FindStringSubmatch(clean); len(m) > 1 {
		info.Season, _ = strconv.Atoi(m[1])
		info.Episode, _ = strconv.Atoi(m[2])
		info.MediaType = TypeSeries
		loc := seasonEpRegexp.FindStringIndex(clean)
		if loc[0] > 0 {
			clean = clean[:loc[0]]
		} else {
			clean = ""
		}
	}

	// If year was found in clean, strip it from title
	if info.Year > 0 {
		clean = regexp.MustCompile(`[\(\[\.\s]`+strconv.Itoa(info.Year)+`[\)\]\.\s]?`).ReplaceAllString(clean, "")
	}

	// 5. Clean title
	title := clean
	tagReg := regexp.MustCompile(`\[.*?\]`)
	title = tagReg.ReplaceAllString(title, "")
	title = strings.ReplaceAll(title, ".", " ")
	title = strings.TrimSpace(title)

	info.Title = title
	if info.MediaType == "" {
		info.MediaType = TypeMovie
	}

	return info
}
