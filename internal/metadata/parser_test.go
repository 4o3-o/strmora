package metadata

import (
	"testing"
)

func TestParseMediaInfo(t *testing.T) {
	cases := []struct {
		input       string
		isDir       bool
		wantName    string
		wantYear    int
		wantSeason  int
		wantEpisode int
		wantPart    string
		wantTMDBID  int
		wantType    MediaType
	}{
		{
			input:      "欲火狂蜜 (1995) tmdb-589499",
			isDir:      true,
			wantName:   "欲火狂蜜",
			wantYear:   1995,
			wantTMDBID: 589499,
			wantType:   TypeMovie,
		},
		{
			input:       "Black.Mirror.S07E06.2025.2160p.NF.WEB-DL.mkv",
			isDir:       false,
			wantName:    "Black Mirror",
			wantYear:    2025,
			wantSeason:  7,
			wantEpisode: 6,
			wantType:    TypeSeries,
		},
		{
			input:    "蜡笔小新.part1.mkv",
			isDir:    false,
			wantName: "蜡笔小新",
			wantPart: "part1",
			wantType: TypeMovie,
		},
		{
			input:       "S01E01-E02.mkv",
			isDir:       false,
			wantSeason:  1,
			wantEpisode: 1,
			wantType:    TypeSeries,
		},
		{
			input:    "Guinea Pig Devil Woman Doctor(1986)[日语发音][内封简中][DVDrip][souyouzi].mkv",
			isDir:    false,
			wantName: "Guinea Pig Devil Woman Doctor",
			wantYear: 1986,
			wantType: TypeMovie,
		},
	}

	for _, c := range cases {
		info := ParseName(c.input, c.isDir)
		if c.wantName != "" && info.Title != c.wantName {
			t.Errorf("[%s] got Title=%s, want %s", c.input, info.Title, c.wantName)
		}
		if c.wantYear != 0 && info.Year != c.wantYear {
			t.Errorf("[%s] got Year=%d, want %d", c.input, info.Year, c.wantYear)
		}
		if c.wantTMDBID != 0 && info.TMDBID != c.wantTMDBID {
			t.Errorf("[%s] got TMDBID=%d, want %d", c.input, info.TMDBID, c.wantTMDBID)
		}
		if c.wantSeason != 0 && info.Season != c.wantSeason {
			t.Errorf("[%s] got Season=%d, want %d", c.input, info.Season, c.wantSeason)
		}
		if c.wantEpisode != 0 && info.Episode != c.wantEpisode {
			t.Errorf("[%s] got Episode=%d, want %d", c.input, info.Episode, c.wantEpisode)
		}
		if c.wantPart != "" && info.Part != c.wantPart {
			t.Errorf("[%s] got Part=%s, want %s", c.input, info.Part, c.wantPart)
		}
	}
}
