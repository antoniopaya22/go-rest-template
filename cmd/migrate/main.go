// Command migrate manages the database schema.
//
//	migrate up       apply every pending migration
//	migrate down     roll back the most recent migration
//	migrate status   list migrations and whether they are applied
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/antoniopaya22/go-rest-template/v3/internal/config"
	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/database"
)

func main() {
	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	if err := run(context.Background(), command); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, command string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	db, err := database.Open(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	switch command {
	case "up":
		return db.Migrate(ctx)
	case "down":
		return db.MigrateDown(ctx)
	case "status":
		statuses, err := db.MigrationStatus(ctx)
		if err != nil {
			return err
		}
		for _, status := range statuses {
			state := "pending"
			if status.Applied {
				state = "applied"
			}
			fmt.Printf("%-8s %s\n", state, status.Name)
		}
		return nil
	default:
		return fmt.Errorf("unknown command %q (use up, down or status)", command)
	}
}
