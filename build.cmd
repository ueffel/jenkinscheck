
set GOOS=windows
set GOARCH=amd64

where goversioninfo > nul 2>&1
if errorlevel 1 (
    go get -v github.com/josephspurrier/goversioninfo/cmd/goversioninfo
)

go generate
IF ERRORLEVEL 1 GOTO error ELSE IF NOT ERRORLEVEL 0 GOTO error

go build -v -ldflags="-H windowsgui"
IF ERRORLEVEL 1 GOTO error ELSE IF NOT ERRORLEVEL 0 GOTO error


exit /B 0


:error
exit /B 1
