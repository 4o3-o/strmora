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
 INSERT OR IGNORE INTO schema_migrations(version) VALUES(1);`)
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
