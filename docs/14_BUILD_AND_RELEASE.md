# 14 — Build and Release

## Version injection

`configs/app.json` holds `version` (semver, currently `0.1.0-dev`). Builds
inject it plus commit/time via linker flags (defined once in
`configs/build.json`, consumed by `scripts/build.bat`):

```bat
go build -trimpath -ldflags "-X <mod>/internal/app.Version=%VER% -X <mod>/internal/app.Commit=%SHA% -X <mod>/internal/app.BuildDate=%DATE%" -o dist\academic.exe .\cmd\academic
go build -trimpath -ldflags "..." -o dist\academic-installer.exe .\cmd\installer
```

`academic version` prints `name version/commit/date + schema_version`;
dev builds report `-dirty` when the tree has changes.

## Workflows (Windows, no extra toolchain)

| Script | Does |
| ------ | ---- |
| `scripts\test.bat` | `go test ./...` + `go vet ./...` + `gofmt -l` check |
| `scripts\build.bat` | dev builds of both exes into `dist\` |
| `scripts\package.bat` | clean `dist\`, release builds, checksums, assembles `release\` |
| `scripts\release.bat` | runs package + prints tag/release checklist (no auto-publish) |

Targets: `windows/amd64` (primary); `GOARCH=386` smoke-compile only if asked.
`dist/` and `release/` are git-ignored; only scripts + configs are versioned.

## Release package layout

```text
release\
├── academic.exe
├── academic-installer.exe
├── README.txt            # install + first-run (generated from README.md)
├── CHANGELOG.txt         # from CHANGELOG.md section for the version
└── checksums.txt         # SHA-256 of every file above
```

## Versioning and publishing procedure

1. Update `version` in `configs/app.json` + `CHANGELOG.md` (Keep-a-Changelog).
2. Run `test.bat`, `build.bat` (smoke: launch TUI, run `version`, `export`).
3. Run `package.bat`; verify checksums.
4. Tag `vX.Y.Z`, push tag; create GitHub Release with the `release\` files
   and notes from the changelog section. Publishing is manual — scripts never
   push or upload by themselves.
5. Post-release: bump `app.json` to next `-dev`, record in DECISIONS.md if the
   schema version changed.

## Renaming the app / bumping versions

Edit `configs/app.json` (`name`, `exe`, `version`) and the output names in
`configs/build.json`; rebuild. No source literals to chase (enforced by grep
check in `package.bat`: fails if the old exe name appears in `internal/`).

## External build-manager readiness

All version/flag/package knowledge lives in `configs/build.json` + the four
scripts, so an external manager can reproduce a release by reading one JSON
file and invoking the same `go build` lines. No IDE, Node, Python or network
access required at build time (module cache vendored on demand via `go mod`).
