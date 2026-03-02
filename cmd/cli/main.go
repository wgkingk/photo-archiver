package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"photo-archiver/internal/core/importer"
	"photo-archiver/internal/core/verifier"
	"photo-archiver/internal/storage/sqlite"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	sub := os.Args[1]
	switch sub {
	case "migrate":
		if err := runMigrate(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "migrate failed: %v\n", err)
			os.Exit(1)
		}
	case "import":
		if err := runImport(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "import failed: %v\n", err)
			os.Exit(1)
		}
	default:
		printUsage()
		os.Exit(2)
	}
}

func runMigrate(args []string) error {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	dbPath := fs.String("db", "./data/photo_archiver.db", "sqlite db path")
	schemaPath := fs.String("schema", "./docs/DB_SCHEMA.sql", "schema sql file path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	store, err := sqlite.Open(*dbPath)
	if err != nil {
		return err
	}
	defer store.Close()
	if err := store.ApplySchema(*schemaPath); err != nil {
		return err
	}
	fmt.Printf("migration completed. db=%s schema=%s\n", *dbPath, *schemaPath)
	return nil
}

func runImport(args []string) error {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	source := fs.String("source", "", "source directory")
	dest := fs.String("dest", "", "archive destination directory")
	dbPath := fs.String("db", "./data/photo_archiver.db", "sqlite db path")
	schemaPath := fs.String("schema", "./docs/DB_SCHEMA.sql", "schema sql file path")
	dryRun := fs.Bool("dry-run", false, "plan only, no copy")
	verifyMode := fs.String("verify", verifier.ModeSize, "verify mode: size|hash")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *source == "" || *dest == "" {
		return fmt.Errorf("--source and --dest are required")
	}
	absSource, err := filepath.Abs(*source)
	if err != nil {
		return err
	}
	absDest, err := filepath.Abs(*dest)
	if err != nil {
		return err
	}
	store, err := sqlite.Open(*dbPath)
	if err != nil {
		return err
	}
	defer store.Close()
	if err := store.ApplySchema(*schemaPath); err != nil {
		return err
	}

	res, err := importer.Run(importer.Request{
		SourceRoot: absSource,
		DestRoot:   absDest,
		DryRun:     *dryRun,
		VerifyMode: *verifyMode,
	}, store)
	if err != nil {
		return err
	}

	fmt.Printf("job=%s status=%s total=%d success=%d skipped=%d failed=%d total_bytes=%d copied_bytes=%d\n",
		res.JobID,
		res.Status,
		res.TotalCount,
		res.SuccessCount,
		res.SkippedCount,
		res.FailedCount,
		res.TotalBytes,
		res.CopiedBytes,
	)
	return nil
}

func printUsage() {
	fmt.Println("photo-archiver CLI")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  photo-archiver migrate --db ./data/photo_archiver.db --schema ./docs/DB_SCHEMA.sql")
	fmt.Println("  photo-archiver import --source /Volumes/SD --dest /Volumes/Backup --db ./data/photo_archiver.db [--dry-run] [--verify size|hash]")
}
