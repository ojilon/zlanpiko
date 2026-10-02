package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"zlanpiko/internal/app"
	"zlanpiko/internal/config"
)

const configHelp = `Usage:
  zlanpiko config show [--format text|json]
  zlanpiko config set KEY VALUE     (KEY: theme | data_root)
  zlanpiko config path              (print the data root for scripting)

theme: auto | 16 | none. data_root must be absolute; it is created when
missing and recorded in the install pointer.
`

func runConfig(ctx *app.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, configHelp)
		return ExitUsage
	}
	pos, f, err := parseMixed(args[1:])
	if err != nil {
		return usageError(stderr, "%v\n%s", err, configHelp)
	}
	if err := rejectUnknown(f, "format"); err != nil {
		return usageError(stderr, "%v", err)
	}
	format, err := outputFormat(f)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	switch args[0] {
	case "show":
		if len(pos) != 0 {
			return usageError(stderr, "show takes no arguments\n%s", configHelp)
		}
		user, err := config.LoadUser(ctx.DataRoot)
		if err != nil {
			return runtimeError(stderr, err)
		}
		pointer, perr := config.PointerPath()
		if perr != nil {
			pointer = "(unknown: " + perr.Error() + ")"
		}
		view := map[string]any{
			"data_root":   ctx.DataRoot,
			"pointer":     pointer,
			"theme":       user.Theme,
			"last_screen": user.LastScreen,
		}
		if format == "json" {
			return encode(stdout, stderr, view)
		}
		printTable(stdout, []string{"KEY", "VALUE"}, [][]string{
			{"data_root", ctx.DataRoot},
			{"pointer", pointer},
			{"theme", user.Theme},
			{"last_screen", user.LastScreen},
		})
		return ExitOK
	case "set":
		if len(pos) != 2 {
			return usageError(stderr, "set needs KEY and VALUE\n%s", configHelp)
		}
		switch pos[0] {
		case "theme":
			if pos[1] != "auto" && pos[1] != "16" && pos[1] != "none" {
				return usageError(stderr, "invalid theme %q (want auto|16|none)", pos[1])
			}
			user, err := config.LoadUser(ctx.DataRoot)
			if err != nil {
				return runtimeError(stderr, err)
			}
			user.Theme = pos[1]
			if err := config.SaveUser(ctx.DataRoot, user); err != nil {
				return runtimeError(stderr, err)
			}
			fmt.Fprintf(stdout, "theme = %s\n", pos[1])
			return ExitOK
		case "data_root":
			abs, err := filepath.Abs(pos[1])
			if err != nil {
				return runtimeError(stderr, err)
			}
			if !filepath.IsAbs(abs) {
				return usageError(stderr, "data_root must be absolute: %q", pos[1])
			}
			if err := os.MkdirAll(abs, 0o755); err != nil {
				return runtimeError(stderr, err)
			}
			if err := config.SavePointer(&config.Pointer{DataRoot: abs}); err != nil {
				return runtimeError(stderr, err)
			}
			fmt.Fprintf(stdout, "data_root = %s\n", abs)
			return ExitOK
		default:
			return usageError(stderr, "unknown key %q (want theme|data_root)", pos[0])
		}
	case "path":
		if len(pos) != 0 {
			return usageError(stderr, "path takes no arguments\n%s", configHelp)
		}
		fmt.Fprintln(stdout, ctx.DataRoot)
		return ExitOK
	default:
		return usageError(stderr, "unknown config command %q\n%s", args[0], configHelp)
	}
}
