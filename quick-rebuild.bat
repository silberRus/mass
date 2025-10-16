@echo off
echo Stopping server...
taskkill /F /IM agario-server.exe >nul 2>&1

echo Building...
cd server
go build -o agario-server.exe cmd/server/main.go

if %ERRORLEVEL% EQU 0 (
    echo Starting server...
    start "Agario Server" cmd /c "agario-server.exe"
    timeout /t 2 >nul
    echo.
    echo Server: http://localhost:8080
    echo Admin: http://localhost:8081/admin
    echo.
) else (
    echo Build failed!
    pause
)
