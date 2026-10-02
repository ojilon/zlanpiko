package app

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"zlanpiko/internal/config"
	"zlanpiko/internal/database"
	"zlanpiko/migrations"
)

// SkeletonDirs are the data-root subdirectories ensured on every Open.
// The database/ files and per-unit folders are created alongside records
// in later phases; Open only guarantees the top-level layout (see docs/06).
var SkeletonDirs = []string{
	"database",
	"config",
	"backups",
	"exports",
	"reports",
	"units",
	"inbox",
}

// Context carries the opened application state. Fronts (TUI, CLI, installer)
// share it; it is created by Open and released by Close.
type Context struct {
	DataRoot string
	DB       *sql.DB
	Logger   *slog.Logger
}

// OpenOptions tunes Open.
type OpenOptions struct {
	// DataRoot explicitly overrides resolution (flag value, may be "").
	DataRoot string
	// Logger is used for diagnostics; nil means a discard logger.
	Logger *slog.Logger
}

// Open resolves the data root, ensures the skeleton layout, opens the
// database and applies pending migrations. It creates the configured root's
// skeleton on first use (first-run init); an unconfigured root (no flag, env
// or pointer file) is an error, never an implicit second database.
func Open(o OpenOptions) (*Context, error) {
	root, err := config.ResolveDataRoot(o.DataRoot)
	if err != nil {
		return nil, err
	}
	for _, dir := range SkeletonDirs {
		path := filepath.Join(root, dir)
		if err := os.MkdirAll(path, 0o755); err != nil {
			return nil, fmt.Errorf("app: create %s: %w", path, err)
		}
	}
	db, err := database.Open(database.Path(root))
	if err != nil {
		return nil, err
	}
	if _, err := database.Migrate(db, migrations.FS, SchemaVersion); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := loadOrInitUser(root); err != nil {
		db.Close()
		return nil, err
	}
	logger := o.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Context{DataRoot: root, DB: db, Logger: logger}, nil
}

// loadOrInitUser returns existing preferences, writing defaults when absent.
func loadOrInitUser(root string) (config.User, error) {
	u, err := config.LoadUser(root)
	if err != nil {
		return config.User{}, err
	}
	present := true
	if _, serr := os.Stat(config.UserPath(root)); serr != nil {
		present = false
	}
	if !present {
		if err := config.SaveUser(root, u); err != nil {
			return config.User{}, err
		}
	}
	return u, nil
}

// Close releases the context's resources.
func (c *Context) Close() error {
	if c.DB != nil {
		return c.DB.Close()
	}
	return nil
}
