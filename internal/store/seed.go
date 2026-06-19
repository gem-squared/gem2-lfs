package store

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"strings"
)

//go:embed seed/*.sql
var seedFS embed.FS

// SeedData executes all SQL files in the embedded seed/ directory.
// Uses INSERT OR IGNORE for idempotent seeding — safe to re-run.
func (s *DB) SeedData() error {
	entries, err := fs.ReadDir(seedFS, "seed")
	if err != nil {
		return fmt.Errorf("read seed directory: %w", err)
	}

	seeded := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		data, err := seedFS.ReadFile("seed/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read seed file %s: %w", entry.Name(), err)
		}

		if _, err := s.db.Exec(string(data)); err != nil {
			return fmt.Errorf("execute seed file %s: %w", entry.Name(), err)
		}

		log.Printf("  [seed] loaded %s", entry.Name())
		seeded++
	}

	if seeded > 0 {
		log.Printf("  [seed] %d seed files loaded", seeded)
	}
	return nil
}
