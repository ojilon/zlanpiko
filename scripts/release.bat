@echo off
REM Release driver: full test suite, then packaging, then the manual
REM publish checklist. This script never tags, pushes or uploads by itself.
REM Usage: release.bat [version]  (default 0.1.0)
setlocal
set VER=0.1.0
if not "%~1"=="" set VER=%~1
call "%~dp0test.bat"
if %errorlevel% neq 0 (
  echo release.bat: tests failed, aborting.
  exit /b 1
)
call "%~dp0package.bat" %VER%
if %errorlevel% neq 0 (
  echo release.bat: packaging failed, aborting.
  exit /b 1
)
echo.
echo Release %VER% is assembled in release\. Manual publish checklist:
echo   1. Verify checksums:      certutil -hashfile release\zlanpiko.exe SHA256
echo   2. Fresh-install rehearsal from release\ on a machine without Go.
echo   3. Smoke test: units/topics/tasks, timeline, export, backup/restore.
echo   4. Update CHANGELOG.md [Unreleased] section if needed.
echo   5. git tag v%VER%
echo   6. git push origin main v%VER%
echo   7. GitHub Release v%VER% with the release\ files + changelog notes.
