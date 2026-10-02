@echo off
REM Run the full test suite plus static checks.
setlocal
go vet ./...
if %errorlevel% neq 0 exit /b %errorlevel%
gofmt -l internal cmd > "%TEMP%\gofmt_zlanpiko.txt" 2>nul
for %%A in ("%TEMP%\gofmt_zlanpiko.txt") do if %%~zA neq 0 (
  echo Unformatted files:
  type "%TEMP%\gofmt_zlanpiko.txt"
  exit /b 1
)
go test ./...
