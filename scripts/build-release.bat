@ECHO OFF

IF "%GOPATH%"=="" GOTO NOGO

ECHO Building deej (release)...
go build -o deej-release.exe -ldflags -H=windowsgui .\cmd
ECHO Done.
GOTO DONE

:NOGO
ECHO GOPATH environment variable not set
GOTO DONE

:DONE
