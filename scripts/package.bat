@echo off
REM Release packaging: clean dist/release, versioned builds of all three
REM executables (CLI+TUI, installer, frameless GUI), release assembly,
REM SHA-256 checksums.
REM Usage: package.bat [version]  (default 0.1.0)
setlocal
set VER=0.1.0
if not "%~1"=="" set VER=%~1
set SHA=none
for /f %%i in ('git rev-parse --short HEAD 2^>nul') do set SHA=%%i
for /f %%d in ('powershell -NoProfile -NonInteractive -Command "(Get-Date).ToString('yyyy-MM-dd')"') do set BUILD_DATE=%%d
if exist dist rmdir /s /q dist
if exist release rmdir /s /q release
mkdir dist
mkdir release
set FLAGS=-trimpath -ldflags "-X zlanpiko/internal/app.Version=%VER% -X zlanpiko/internal/app.Commit=%SHA% -X zlanpiko/internal/app.BuildDate=%BUILD_DATE%"
go build %FLAGS% -o dist\zlanpiko.exe .\cmd\zlanpiko
if %errorlevel% neq 0 exit /b %errorlevel%
go build %FLAGS% -o dist\zlanpiko-installer.exe .\cmd\installer
if %errorlevel% neq 0 exit /b %errorlevel%
call "%~dp0build-gui.bat" %VER%
if %errorlevel% neq 0 (
  echo package.bat: GUI build failed, aborting.
  exit /b 1
)
copy /y dist\zlanpiko.exe release\ >nul
copy /y dist\zlanpiko-installer.exe release\ >nul
copy /y dist\zlanpiko-gui.exe release\ >nul
copy /y assets\README.txt release\ >nul
copy /y CHANGELOG.md release\CHANGELOG.txt >nul
powershell -NoProfile -NonInteractive -Command "$files = @('release\zlanpiko.exe','release\zlanpiko-installer.exe','release\zlanpiko-gui.exe','release\README.txt','release\CHANGELOG.txt'); $out = foreach ($f in $files) { $h = (Get-FileHash -Algorithm SHA256 -LiteralPath $f).Hash.ToLower(); \"$h  $(Split-Path $f -Leaf)\" }; $out | Out-File -Encoding ascii release\checksums.txt"
if %errorlevel% neq 0 exit /b %errorlevel%
echo Release %VER% assembled in release\:
dir /b release
