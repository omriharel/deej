@ECHO OFF

ECHO Building deej (development)...

REM set repo root in relation to script path to avoid cwd dependency
SET "DEEJ_ROOT=%~dp0..\..\..\.."

REM shove git commit, version tag into env
for /f "delims=" %%a in ('git rev-list -1 --abbrev-commit HEAD') do @set GIT_COMMIT=%%a
for /f "delims=" %%a in ('git describe --tags --always') do @set VERSION_TAG=%%a
set BUILD_TYPE=dev
ECHO Embedding build-time parameters:
ECHO - gitCommit %GIT_COMMIT%
ECHO - versionTag %VERSION_TAG%
ECHO - buildType %BUILD_TYPE%

REM go build -o "C:\Users\Fippi\git\deej\build\deej-dev.exe" -ldflags "-X main.gitCommit=47831d0 -X main.versionTag=v0.9.9-11-g47831d0 -X main.buildType=dev "C:\Users\Fippi\git\deej\cmd"
go build -o "%DEEJ_ROOT%\deej-dev.exe" -ldflags "-X main.gitCommit=%GIT_COMMIT% -X main.versionTag=%VERSION_TAG% -X main.buildType=%BUILD_TYPE%" "%DEEJ_ROOT%\pkg\deej\cmd"
if %ERRORLEVEL% NEQ 0 GOTO BUILDERROR
ECHO Done.
GOTO DONE

:BUILDERROR
ECHO Failed to build deej in development mode! See above output for details.
EXIT /B 1

:DONE
