
set GOOS=windows
set GOARCH=amd64

where rsrc > nul 2>&1
if errorlevel 1 (
    go get -v github.com/akavel/rsrc
)

rem rsrc -manifest main.manifest -arch amd64 -ico ico/favicon.ico
go generate
IF ERRORLEVEL 1 GOTO error ELSE IF NOT ERRORLEVEL 0 GOTO error

go build -ldflags="-H windowsgui"
IF ERRORLEVEL 1 GOTO error ELSE IF NOT ERRORLEVEL 0 GOTO error


exit /B 0


:error
exit /B 1
