package main

import (
	"flag"
	"fmt"
	"os"

	"photo-archiver/internal/gui"
	"photo-archiver/internal/storage/sqlite"
)

func main() {
	dbPath := flag.String("db", "./data/photo_archiver.db", "sqlite db path")
	schemaPath := flag.String("schema", "./docs/DB_SCHEMA.sql", "schema sql file path")
	flag.Parse()

	store, err := sqlite.Open(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open db failed: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	if err := store.ApplySchema(*schemaPath); err != nil {
		fmt.Fprintf(os.Stderr, "apply schema failed: %v\n", err)
		os.Exit(1)
	}

	app := gui.New(store)
	app.Run()
}
