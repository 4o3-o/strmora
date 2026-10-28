package inventory

import (
	"encoding/binary"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestTreeUTF16PathsAndUnknownLeaf(t *testing.T) {
	text := "|——根目录\r\n| |-电影\r\n| | |-作品 (2025)\r\n\r\n| | | |-影片.mkv\r\n| | | |-影片.zh.srt\r\n| | | |-未知条目\r\n"
	b := []byte{0xff, 0xfe}
	for _, u := range utf16.Encode([]rune(text)) {
		b = binary.LittleEndian.AppendUint16(b, u)
	}
	tree, err := Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if tree.Root != "根目录" || len(tree.Entries) != 5 || tree.Entries[2].Path != "电影/作品 (2025)/影片.mkv" || tree.Entries[4].Kind != "unknown" || tree.Entries[0].Kind != "directory" {
		t.Fatalf("unexpected tree: %+v", tree)
	}
}
func TestTreeRejectsUnsafeAndBrokenInput(t *testing.T) {
	for _, s := range []string{"|——根目录\n| | |-跳层", "|——根目录\n| |-..", "|——根目录\n| |-a/b", "|——根目录\ninvalid", "|——根目录\n| |-a\u0000b"} {
		if _, err := Parse([]byte(s)); err == nil {
			t.Errorf("accepted %q", s)
		}
	}
	for _, b := range [][]byte{{0xff, 0xfe, 0x00}, {0xff, 0xfe, 0x00, 0xd8}, {0xff, 0xfe, 0, 0xdc}, {0xff}} {
		if _, err := Parse(b); err == nil {
			t.Errorf("accepted bytes %x", b)
		}
	}
}
func TestTreePreservesNamesAndAllowsNoFinalNewline(t *testing.T) {
	tree, err := Parse([]byte("\ufeff|——根目录\n| |- 名称 - part1.mkv "))
	if err != nil {
		t.Fatal(err)
	}
	if tree.Entries[0].Path != " 名称 - part1.mkv " {
		t.Fatal(tree)
	}
	if strings.Contains(tree.Entries[0].Path, "根目录") {
		t.Fatal(tree)
	}
}

func TestTreeReportsDuplicatePathsWithoutLosingRows(t *testing.T) {
	tree, err := Parse([]byte("|——根目录\n| |-a\n| |-a"))
	if err != nil {
		t.Fatal(err)
	}
	if len(tree.Entries) != 2 || len(tree.DuplicatePaths) != 1 || tree.DuplicatePaths[0] != "a" {
		t.Fatalf("lost duplicates: %+v", tree)
	}
}
