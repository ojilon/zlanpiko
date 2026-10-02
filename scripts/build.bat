@echo off
REM Dev build of zlanpiko.exe with version metadata injected.
REM Usage: build.bat [version]  (default 0.1.0-dev)
setlocal
set VER=0.1.0-dev
if not "%~1"=="" set VER=%~1
set SHA=none
for /f %%i in ('git rev-parse --short HEAD 2^>nul') do set SHA=%%i
if not exist dist mkdir dist
go build -trimpath -ldflags "-X zlanpiko/internal/app.Version=%VER% -X zlanpiko/internal/app.Commit=%SHA%" -o dist\zlanpiko.exe .\cmd\zlanpiko
