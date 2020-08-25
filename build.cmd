go generate -v
IF ERRORLEVEL 1 GOTO error ELSE IF NOT ERRORLEVEL 0 GOTO error

go build -v -ldflags="-H windowsgui -w -s" -trimpath
IF ERRORLEVEL 1 GOTO error ELSE IF NOT ERRORLEVEL 0 GOTO error

exit /B 0

:error
exit /B 1
