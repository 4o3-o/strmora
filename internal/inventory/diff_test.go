package inventory

import (
	"testing"
)

func TestSnapshotDiff(t *testing.T) {
	oldTree := Tree{
		Root: "根目录",
		Entries: []Entry{
			{Path: "Movies/A.mkv", Kind: "unknown"},
			{Path: "Movies/B.mkv", Kind: "unknown"},
			{Path: "Movies/C.mkv", Kind: "unknown"},
		},
	}

	newTree := Tree{
		Root: "根目录",
		Entries: []Entry{
			{Path: "Movies/A.mkv", Kind: "unknown"},
			{Path: "Movies/C.mkv", Kind: "unknown"},
			{Path: "Movies/D.mkv", Kind: "unknown"},
		},
	}

	diff := ComputeDiff(oldTree, newTree)

	if len(diff.Added) != 1 || diff.Added[0].Path != "Movies/D.mkv" {
		t.Fatalf("unexpected added: %+v", diff.Added)
	}
	if len(diff.Removed) != 1 || diff.Removed[0].Path != "Movies/B.mkv" {
		t.Fatalf("unexpected removed: %+v", diff.Removed)
	}
	if len(diff.Unchanged) != 2 {
		t.Fatalf("unexpected unchanged: %+v", diff.Unchanged)
	}
}
