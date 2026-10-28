package database

import (
	"path/filepath"
	"testing"
)

func TestDatabaseMigrationAndProgressSurviveReopen(t *testing.T) {
	p := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	var mode string
	if err = db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil || mode != "wal" {
		t.Fatalf("wal: %s %v", mode, err)
	}
	_, err = db.Exec("INSERT INTO user_data(user_id,item_id,position_ticks,played,favorite) VALUES('u','movie',123,0,1)")
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	db, err = Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var pos int64
	var fav bool
	if err = db.QueryRow("SELECT position_ticks,favorite FROM user_data WHERE user_id='u' AND item_id='movie'").Scan(&pos, &fav); err != nil || pos != 123 || !fav {
		t.Fatalf("progress: %d %t %v", pos, fav, err)
	}
}
