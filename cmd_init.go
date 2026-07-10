package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gsd/internal/store"
)

func cmdInit() error {
	dir := store.Dir
	if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
		fmt.Println("gsd: already initialized")
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// merge=union: git resolves concurrent appends by keeping both sides'
	// lines — safe because each line is self-contained and replay orders
	// by timestamp, not file position.
	files := map[string]string{
		".gitattributes": "log.jsonl merge=union\n",
		".gitignore":     "cache.db\ncache.db-journal\n",
		"log.jsonl":      "",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			return err
		}
	}
	fmt.Println("initialized .gsd/ — log.jsonl is the source of truth, commit it; cache.db is ignored")
	fmt.Println("next: gsd task add \"first task\"")
	return nil
}
