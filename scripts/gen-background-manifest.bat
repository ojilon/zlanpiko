@echo off
REM Regenerate frontend/public/backgrounds/manifest.json (docs/24).
REM Run this after adding or removing background images.
setlocal
pushd "%~dp0\.."
node scripts\gen-background-manifest.mjs
if %errorlevel% neq 0 (
  echo.
  echo node not found? The manifest was not updated.
  popd
  exit /b %errorlevel%
)
popd
