package organizer

import (
	"fmt"
	"strconv"
	"strings"
)

type TemplateData struct {
	Title   string
	Year    int
	Season  int
	Episode int
	Part    string
	Version string
	TMDBID  int
	Ext     string
}

type Template struct {
	pattern string
}

func NewTemplate(pattern string) *Template {
	if pattern == "" {
		pattern = "{title} ({year})"
	}
	return &Template{pattern: pattern}
}

func (t *Template) Format(d TemplateData) string {
	res := t.pattern

	res = strings.ReplaceAll(res, "{title}", d.Title)
	if d.Year > 0 {
		res = strings.ReplaceAll(res, "{year}", strconv.Itoa(d.Year))
	} else {
		res = strings.ReplaceAll(res, " ({year})", "")
		res = strings.ReplaceAll(res, "{year}", "")
	}

	res = strings.ReplaceAll(res, "{season}", fmt.Sprintf("%02d", d.Season))
	res = strings.ReplaceAll(res, "{episode}", fmt.Sprintf("%02d", d.Episode))

	if d.Part != "" {
		res = strings.ReplaceAll(res, "{part}", d.Part)
	} else {
		res = strings.ReplaceAll(res, " - {part}", "")
		res = strings.ReplaceAll(res, "{part}", "")
	}

	if d.Version != "" {
		res = strings.ReplaceAll(res, "{version}", d.Version)
	} else {
		res = strings.ReplaceAll(res, " - {version}", "")
		res = strings.ReplaceAll(res, "{version}", "")
	}

	if d.TMDBID > 0 {
		res = strings.ReplaceAll(res, "{tmdbid}", strconv.Itoa(d.TMDBID))
	} else {
		res = strings.ReplaceAll(res, "{tmdbid}", "")
	}

	if d.Ext != "" {
		res = strings.ReplaceAll(res, "{ext}", strings.TrimPrefix(d.Ext, "."))
	} else {
		res = strings.ReplaceAll(res, ".{ext}", "")
		res = strings.ReplaceAll(res, "{ext}", "")
	}

	return strings.TrimSpace(res)
}
