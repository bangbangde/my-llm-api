@echo off
echo Building the application...
go build -o llm-gateway.exe

echo Starting the server...
start /B llm-gateway.exe

echo Waiting for server to start...
timeout /t 3 /nobreak >nul

echo Running integration tests...
go test -run TestIntegration -v

echo Done!
pause
