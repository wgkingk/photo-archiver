package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"photo-archiver/internal/api/httpapi"
	"photo-archiver/internal/storage/sqlite"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:38080", "http listen address")
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

	api := httpapi.New(store)
	fmt.Printf("photo-archiver service listening on http://%s\n", *addr)
	if err := http.ListenAndServe(*addr, api.Handler()); err != nil {
		fmt.Fprintf(os.Stderr, "service stopped: %v\n", err)
		os.Exit(1)
	}
}
