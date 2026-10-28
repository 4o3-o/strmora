package database

import (
	"path/filepath"
	"testing"
)

func TestSourceAndSnapshotPersistence(t *testing.T) {
	p := filepath.Join(t.TempDir(), "test_sources.db")
	db, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// 1. Insert a media source
	_, err = db.Exec("INSERT INTO media_sources(id, name, type, endpoint, root_path, read_only) VALUES('src-1', 'My OpenList', 'openlist', 'https://example.com', '/115', 1)")
	if err != nil {
		t.Fatalf("insert source failed: %v", err)
	}

	// 2. Insert source entries
	_, err = db.Exec("INSERT INTO source_entries(source_id, path, is_dir, size, status) VALUES('src-1', 'Movies/A.mkv', 0, 1048576, 'active')")
	if err != nil {
		t.Fatalf("insert entry failed: %v", err)
	}

	// 3. Mark entry missing only on verified scan (deletion safety)
	_, err = db.Exec("UPDATE source_entries SET status='missing' WHERE source_id='src-1' AND path='Movies/A.mkv'")
	if err != nil {
		t.Fatalf("update status failed: %v", err)
	}

	var status string
	err = db.QueryRow("SELECT status FROM source_entries WHERE source_id='src-1' AND path='Movies/A.mkv'").Scan(&status)
	if err != nil || status != "missing" {
		t.Fatalf("expected missing status, got %s, err: %v", status, err)
	}
}
