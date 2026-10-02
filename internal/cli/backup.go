package cli

import (
	"fmt"
	"io"

	"zlanpiko/internal/app"
	"zlanpiko/internal/backup"
	"zlanpiko/internal/database"
)

const backupHelp = `Usage:
  zlanpiko backup create [--full] [--out FILE]
  zlanpiko backup list [--format text|json]
  zlanpiko backup verify FILE
  zlanpiko backup restore FILE --yes [--overwrite-data]

create defaults to a full timestamped zip in backups/ (db-only without
--full); existing destinations are refused. verify re-checks every hash.
restore needs an empty target or --overwrite-data (a safety backup is taken
first); newer-schema backups are refused.
`

func runBackup(ctx *app.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, backupHelp)
		return ExitUsage
	}
	pos, f, err := parseMixed(args[1:])
	if err != nil {
		return usageError(stderr, "%v\n%s", err, backupHelp)
	}
	switch args[0] {
	case "create":
		if len(pos) != 0 {
			return usageError(stderr, "create takes no positional arguments\n%s", backupHelp)
		}
		if err := rejectUnknown(f, "full", "out"); err != nil {
			return usageError(stderr, "%v", err)
		}
		m, err := backup.Create(ctx.DB, ctx.DataRoot, f.get("out"), f.has("full"), ctx.Logger)
		if err != nil {
			return runtimeError(stderr, err)
		}
		path := f.get("out")
		if path == "" {
			if list, lerr := backup.List(ctx.DataRoot); lerr == nil && len(list) > 0 {
				path = list[0].Path // newest first
			}
		}
		fmt.Fprintf(stdout, "backup created: %s (%d files)\n", path, len(m.Files))
		return ExitOK
	case "list":
		if len(pos) != 0 {
			return usageError(stderr, "list takes no positional arguments\n%s", backupHelp)
		}
		if err := rejectUnknown(f, "format"); err != nil {
			return usageError(stderr, "%v", err)
		}
		format, err := outputFormat(f)
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		list, err := backup.List(ctx.DataRoot)
		if err != nil {
			return runtimeError(stderr, err)
		}
		if format == "json" {
			return encode(stdout, stderr, list)
		}
		rows := make([][]string, 0, len(list))
		for _, b := range list {
			rows = append(rows, []string{b.Path, b.Kind, b.Created.Format("2006-01-02 15:04"), fmt.Sprint(b.Files)})
		}
		printTable(stdout, []string{"PATH", "KIND", "CREATED", "FILES"}, rows)
		return ExitOK
	case "verify":
		if len(pos) != 1 {
			return usageError(stderr, "verify needs exactly one FILE\n%s", backupHelp)
		}
		if err := rejectUnknown(f, "format"); err != nil {
			return usageError(stderr, "%v", err)
		}
		m, err := backup.Verify(pos[0])
		if err != nil {
			return runtimeError(stderr, err)
		}
		fmt.Fprintf(stdout, "backup ok: %s (%d files, %s)\n", m.Kind, len(m.Files), m.CreatedAt.Format("2006-01-02 15:04"))
		return ExitOK
	case "restore":
		if len(pos) != 1 {
			return usageError(stderr, "restore needs exactly one FILE\n%s", backupHelp)
		}
		if err := rejectUnknown(f, "yes", "overwrite-data"); err != nil {
			return usageError(stderr, "%v", err)
		}
		if !f.yes() {
			return usageError(stderr, "restore requires --yes")
		}
		overwrite := f.has("overwrite-data")
		target := database.Path(ctx.DataRoot)
		empty, err := backup.IsEmpty(ctx.DB)
		if err != nil {
			return runtimeError(stderr, err)
		}
		if !empty && !overwrite {
			return runtimeError(stderr, fmt.Errorf("target holds data (re-run with --overwrite-data; a safety backup is taken first)"))
		}
		if !empty && overwrite {
			if _, err := backup.Create(ctx.DB, ctx.DataRoot, "", false, ctx.Logger); err != nil {
				return runtimeError(stderr, fmt.Errorf("safety backup failed, aborting restore: %w", err))
			}
			fmt.Fprintln(stdout, "safety backup recorded")
		}
		// The database file is replaced underneath us: close first.
		if err := ctx.Close(); err != nil {
			return runtimeError(stderr, err)
		}
		if err := backup.Restore(ctx.DataRoot, pos[0], true, ctx.Logger); err != nil {
			// Reopen whatever is on disk so the context stays usable.
			if db, oerr := database.Open(target); oerr == nil {
				ctx.DB = db
			} else {
				ctx.DB = nil
			}
			return runtimeError(stderr, err)
		}
		db, err := database.Open(target)
		if err != nil {
			ctx.DB = nil
			return runtimeError(stderr, err)
		}
		ctx.DB = db
		fmt.Fprintln(stdout, "restore complete")
		return ExitOK
	default:
		return usageError(stderr, "unknown backup command %q\n%s", args[0], backupHelp)
	}
}
