@echo off
REM Run the full test suite plus static checks (Go + frontend).
setlocal
go vet ./...
if %errorlevel% neq 0 exit /b %errorlevel%
gofmt -l internal cmd main.go > "%TEMP%\gofmt_zlanpiko.txt" 2>nul
for %%A in ("%TEMP%\gofmt_zlanpiko.txt") do if %%~zA neq 0 (
  echo Unformatted files:
  type "%TEMP%\gofmt_zlanpiko.txt"
  exit /b 1
)
go test ./...
if %errorlevel% neq 0 exit /b %errorlevel%
cmd /c "cd frontend && pnpm install && pnpm typecheck && pnpm test"
if %errorlevel% neq 0 (
  echo test.bat: frontend checks failed.
  exit /b %errorlevel%
)
echo All checks passed (Go + frontend).
