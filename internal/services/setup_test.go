package services

import (
	"database/sql"
	"log/slog"
	"testing"

	"zlanpiko/internal/database"
	"zlanpiko/migrations"
)

// setupDB opens a migrated database in a temp data root.
func setupDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	root := t.TempDir()
	db, err := database.Open(database.Path(root))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := database.Migrate(db, migrations.FS, 1); err != nil {
		t.Fatal(err)
	}
	return db, root
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
