@ECHO OFF

IF "%GOPATH%"=="" GOTO NOGO
IF NOT EXIST %GOPATH%\bin\rsrc.exe GOTO INSTALL
:POSTINSTALL
ECHO Creating pkg/deej/cmd/rsrc.syso
%GOPATH%\bin\rsrc -manifest pkg\deej\assets\deej.manifest  -ico pkg\deej\assets\logo.ico -o pkg\deej\cmd\rsrc_windows.syso
GOTO DONE

:INSTALL
ECHO Installing rsrc...
go get  github.com/akavel/rsrc
IF ERRORLEVEL 1 GOTO GETFAIL
GOTO POSTINSTALL

:GETFAIL
ECHO Failure running go get  github.com/akavel/rsrc.  Ensure that go and git are in PATH
GOTO DONE

:NOGO
ECHO GOPATH environment variable not set
GOTO DONE

:DONE
