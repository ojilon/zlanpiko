zlanpiko — local-first academic manager for Windows
================================================

CONTENTS
--------
zlanpiko.exe             The application (terminal UI + command line).
zlanpiko-installer.exe   Setup: install location, data directory,
                         Start-Menu shortcut and PATH entry.
README.txt               This file.
CHANGELOG.txt            What changed in each version.
checksums.txt            SHA-256 hashes of the files above.

REQUIREMENTS
------------
Windows 10 or 11. No administrator rights, no internet connection and no
additional software are required.

INSTALL
-------
1. Copy all files from this package into one folder (anywhere you like).
2. Run zlanpiko-installer.exe and follow the prompts:
   - installation directory (default: %LocalAppData%\Zlanpiko)
   - academic data directory (default: %USERPROFILE%\AcademicData)
3. Restart your terminal so the PATH entry takes effect.
4. Start with:  zlanpiko

The installer is safe to run again later: it replaces program files,
preserves your academic data and applies any database updates.

FIRST RUN WITHOUT THE INSTALLER
-------------------------------
Running zlanpiko.exe directly also works: it asks for a data directory
on first launch and stores everything there.

YOUR DATA
---------
Academic records live in the data directory you chose (SQLite database
plus real folders per unit, topic and task). The program directory only
holds the executables. Back up with:  zlanpiko backup create --full

UPDATING
--------
Run the newer zlanpiko-installer.exe over the existing installation.
Your data and settings are preserved.

HELP
----
zlanpiko help            command overview
zlanpiko units list      course units
... and inside the app press ? for keys, or type /help.
