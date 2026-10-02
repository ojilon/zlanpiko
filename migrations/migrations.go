// Package migrations embeds the ordered SQL migration files from this
// directory so the installed executable carries them (see docs/05).
// Files are forward-only and named NNNN_name.sql.
package migrations

import "embed"

// FS holds all *.sql migration files in this directory.
//
//go:embed *.sql
var FS embed.FS
