go run tools/build.go
IF ERRORLEVEL 1 GOTO error ELSE IF NOT ERRORLEVEL 0 GOTO error

exit /B 0

:error
exit /B 1
