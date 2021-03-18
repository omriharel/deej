@ECHO OFF

ECHO Building deej (all)...

REM set windows scripts dir root in relation to script path to avoid cwd dependency
SET "WIN_SCRIPTS_ROOT=%~dp0"

CALL "%WIN_SCRIPTS_ROOT%build-dev.bat"
CALL "%WIN_SCRIPTS_ROOT%build-release.bat"
