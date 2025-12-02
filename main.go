package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"text/tabwriter"

	"github.com/mook-as/zypper-filesearch/database"
	"github.com/mook-as/zypper-filesearch/repository"
	"github.com/mook-as/zypper-filesearch/zypper"
)

func run(ctx context.Context) error {
	verbose := flag.Bool("verbose", false, "Enable debug logging")
	flag.Parse()

	var logOptions slog.HandlerOptions
	if *verbose {
		logOptions.Level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &logOptions)))

	if flag.NArg() != 1 {
		return fmt.Errorf("Usage: zypper file-search [pattern]")
	}
	pattern := flag.Arg(0)

	db, err := database.New(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	repos, err := zypper.ListRepositories(ctx)
	if err != nil {
		return err
	}
	if err := repository.Refresh(ctx, db, repos); err != nil {
		return err
	}
	results, err := db.Search(ctx, pattern)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		return fmt.Errorf("No results found")
	}

	writer := tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
	fmt.Fprint(writer, "Repository\tPackage\tFile\n")
	fmt.Fprint(writer, "---\t---\t---\n")
	for _, result := range results {
		fmt.Fprintf(writer, "%s\t%s\t%s\n", result[0], result[1], result[2])
	}
	return writer.Flush()
}

func main() {
	err := run(context.Background())
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
