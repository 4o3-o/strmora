package organizer

import (
	"testing"
)

func TestNamingTemplateFormat(t *testing.T) {
	// 1. Movie folder template default: {title} ({year})
	mFolderTpl := NewTemplate("{title} ({year})")
	res := mFolderTpl.Format(TemplateData{
		Title: "你的名字",
		Year:  2016,
	})
	if res != "你的名字 (2016)" {
		t.Fatalf("expected '你的名字 (2016)', got '%s'", res)
	}

	// 2. Movie file template with part & ext
	mFileTpl := NewTemplate("{title} - {part}.{ext}")
	res = mFileTpl.Format(TemplateData{
		Title: "蜡笔小新",
		Part:  "part1",
		Ext:   "mkv",
	})
	if res != "蜡笔小新 - part1.mkv" {
		t.Fatalf("expected '蜡笔小新 - part1.mkv', got '%s'", res)
	}

	// 3. TV Episode file template: S{season:02d}E{episode:02d}
	tvFileTpl := NewTemplate("{title} - S{season}E{episode}.{ext}")
	res = tvFileTpl.Format(TemplateData{
		Title:   "一拳超人",
		Season:  1,
		Episode: 2,
		Ext:     "mkv",
	})
	if res != "一拳超人 - S01E02.mkv" {
		t.Fatalf("expected '一拳超人 - S01E02.mkv', got '%s'", res)
	}
}
