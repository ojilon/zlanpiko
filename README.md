# zlanpiko

Local-first academic management for Windows: course units, topics and reading
progress, assignments and deadlines, study files, timeline and analytics —
via a terminal UI and a scriptable CLI. No internet connection required.

> Status: **v0.1.0 released** (tag `v0.1.0`). Start with the user guides:
> `docs/18_CLI_USER_GUIDE.md` and `docs/19_TUI_USER_GUIDE.md`. Known
> limitations and the v0.1.1 plan live in `docs/20`; the maintainer's map
> of the code in `docs/21`.

## Quick start (developers, Windows)

```bat
.\scripts\build.bat
.\dist\zlanpiko.exe help
.\dist\zlanpiko.exe version
.\scripts\test.bat
```

Requires Go 1.26+. Planning docs are in `docs/` (`00`–`17` design,
`18`–`19` user guides, `20` limitations + next release, `21` how it works).
