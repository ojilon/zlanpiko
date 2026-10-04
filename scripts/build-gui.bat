@echo off
REM GUI build: frontend deps + typecheck + production build, then the
REM frameless Wails executable with version metadata injected.
REM Usage: build-gui.bat [version]  (default 0.1.0-dev)
REM Requires: Go toolchain, Wails CLI v2, Node + pnpm.
REM NOTE: pnpm runs under cmd /c because PowerShell 5.1 blocks its .ps1
REM shim; wails.exe needs no wrapper. TypeScript for ad-hoc use lives in
REM D:\Dev\pnpm\global (pnpm add -g typescript); the build itself uses the
REM pinned local toolchain in frontend (see pnpm-lock.yaml).
setlocal
set VER=0.1.0-dev
if not "%~1"=="" set VER=%~1
set SHA=none
for /f %%i in ('git rev-parse --short HEAD 2^>nul') do set SHA=%%i
for /f %%d in ('powershell -NoProfile -NonInteractive -Command "(Get-Date).ToString('yyyy-MM-dd')"') do set BUILD_DATE=%%d
if not exist dist mkdir dist
cmd /c "cd frontend && pnpm install && pnpm typecheck && pnpm build"
if %errorlevel% neq 0 (
  echo build-gui.bat: frontend step failed.
  exit /b %errorlevel%
)
wails build -skipbindings -o zlanpiko-gui.exe -ldflags "-X zlanpiko/internal/app.Version=%VER% -X zlanpiko/internal/app.Commit=%SHA% -X zlanpiko/internal/app.BuildDate=%BUILD_DATE%"
if %errorlevel% neq 0 (
  echo build-gui.bat: wails build failed.
  exit /b %errorlevel%
)
copy /y build\bin\zlanpiko-gui.exe dist\zlanpiko-gui.exe >nul
if %errorlevel% neq 0 exit /b %errorlevel%
echo GUI %VER% built: dist\zlanpiko-gui.exe
