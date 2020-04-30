@ECHO OFF

IF "%GOPATH%"=="" GOTO NOGO

ECHO Building deej (development)...
go build -o deej-dev.exe .\cmd
ECHO Done.
GOTO DONE

:NOGO
ECHO GOPATH environment variable not set
GOTO DONE

:DONE
