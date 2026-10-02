package filesystem

import (
	"database/sql"
	"testing"

	"zlanpiko/internal/database"
	"zlanpiko/migrations"
)

// openTestDB opens a migrated database for filesystem tests (test-only).
func openTestDB(t *testing.T, root string) *sql.DB {
	t.Helper()
	db, err := database.Open(database.Path(root))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := database.Migrate(db, migrations.FS, 1); err != nil {
		t.Fatal(err)
	}
	return db
}

func execSQL(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatal(err)
	}
}
