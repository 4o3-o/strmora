package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"strmora/internal/database"
	"strmora/internal/emby"
	"strmora/internal/inventory"
)

func main() {
	if e := run(); e != nil {
		log.Fatal(e)
	}
}
func run() error {
	if len(os.Args) > 1 && os.Args[1] == "tree" {
		f := flag.NewFlagSet("tree", flag.ContinueOnError)
		input := f.String("file", "", "local directory-tree TXT")
		full := f.Bool("entries", false, "include private paths in JSON output")
		if e := f.Parse(os.Args[2:]); e != nil {
			return e
		}
		if *input == "" {
			return fmt.Errorf("tree -file is required")
		}
		b, e := os.ReadFile(*input)
		if e != nil {
			return e
		}
		tree, e := inventory.Parse(b)
		if e != nil {
			return e
		}
		if *full {
			return json.NewEncoder(os.Stdout).Encode(tree)
		}
		dirs := 0
		for _, x := range tree.Entries {
			if x.Kind == "directory" {
				dirs++
			}
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"bytes": len(b), "encoding": tree.Encoding, "entries": len(tree.Entries), "duplicate_paths": len(tree.DuplicatePaths), "known_directories": dirs, "unknown_leaves": len(tree.Entries) - dirs, "completeness": "not proven by TXT syntax"})
	}
	f := flag.NewFlagSet("strmora", flag.ContinueOnError)
	addr := f.String("listen", "127.0.0.1:8096", "listen address (P0: trusted local testing only)")
	dir := f.String("data", "data/p0", "local P0 data directory")
	accessPath := f.String("access-log", "", "optional sanitized JSONL access log")
	if e := f.Parse(os.Args[1:]); e != nil {
		return e
	}
	root, e := filepath.Abs(*dir)
	if e != nil {
		return e
	}
	if e = os.MkdirAll(root, 0700); e != nil {
		return e
	}
	db, e := database.Open(filepath.Join(root, "strmora.db"))
	if e != nil {
		return e
	}
	defer db.Close()
	handler, e := emby.New(db, emby.Options{Username: os.Getenv("STRMORA_P0_USER"), Password: os.Getenv("STRMORA_P0_PASSWORD"), MediaURL: os.Getenv("STRMORA_P0_MEDIA_URL"), DataDir: root})
	if e != nil {
		return e
	}
	var httpHandler http.Handler = handler
	if *accessPath != "" {
		file, err := os.OpenFile(*accessPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			return err
		}
		defer file.Close()
		httpHandler = accessLog(handler, log.New(file, "", 0))
	}
	server := &http.Server{Addr: *addr, Handler: httpHandler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan error, 1)
	go func() {
		log.Printf("Strmora P0 listening on %s (synthetic library; not client-certified)", *addr)
		done <- server.ListenAndServe()
	}()
	select {
	case e := <-done:
		if e == http.ErrServerClosed {
			return nil
		}
		return e
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}
