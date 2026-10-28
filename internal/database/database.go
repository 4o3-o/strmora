package database

import (
	"database/sql"
	"fmt"
	"net/url"

	_ "github.com/mattn/go-sqlite3"
)

// Open applies a transactional, idempotent P0 migration. Only local files are supported.
func Open(filename string) (*sql.DB, error) {
	dsn := (&url.URL{Scheme: "file", Path: filename}).String() + "?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000&_synchronous=NORMAL"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	tx, err := db.Begin()
	if err != nil {
		db.Close()
		return nil, err
	}
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations(version INTEGER PRIMARY KEY);
 CREATE TABLE IF NOT EXISTS settings(key TEXT PRIMARY KEY,value TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS users(id TEXT PRIMARY KEY,name TEXT UNIQUE NOT NULL,password_hash TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS devices(token_hash TEXT PRIMARY KEY,user_id TEXT NOT NULL REFERENCES users(id),device_id TEXT NOT NULL,created_at TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS user_data(user_id TEXT NOT NULL,item_id TEXT NOT NULL,position_ticks INTEGER NOT NULL DEFAULT 0,played INTEGER NOT NULL DEFAULT 0,favorite INTEGER NOT NULL DEFAULT 0,updated_at TEXT,PRIMARY KEY(user_id,item_id));
 CREATE TABLE IF NOT EXISTS playback_sessions(id TEXT PRIMARY KEY,user_id TEXT NOT NULL REFERENCES users(id),item_id TEXT NOT NULL,position_ticks INTEGER NOT NULL,updated_at TEXT NOT NULL);
 	CREATE TABLE IF NOT EXISTS media_sources(
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		endpoint TEXT,
		root_path TEXT NOT NULL,
		token TEXT,
		username TEXT,
		password TEXT,
		read_only INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);
	CREATE TABLE IF NOT EXISTS source_entries(
		source_id TEXT NOT NULL REFERENCES media_sources(id),
		path TEXT NOT NULL,
		is_dir INTEGER NOT NULL DEFAULT 0,
		size INTEGER NOT NULL DEFAULT 0,
		modified_at TEXT,
		status TEXT NOT NULL DEFAULT 'active',
		PRIMARY KEY(source_id, path)
	);
	CREATE TABLE IF NOT EXISTS source_snapshots(
		id TEXT PRIMARY KEY,
		source_id TEXT NOT NULL REFERENCES media_sources(id),
		file_path TEXT NOT NULL,
		encoding TEXT NOT NULL,
		entry_count INTEGER NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);
	INSERT OR IGNORE INTO schema_migrations(version) VALUES(1);
`)
	if err != nil {
		tx.Rollback()
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	if err = tx.Commit(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
