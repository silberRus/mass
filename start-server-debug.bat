@echo off
echo ========================================
echo    Starting Agario Server with Logs
echo ========================================
cd server

echo.
echo Updating Go modules...
go mod tidy

echo.
echo Building server...
go build -o agario-server.exe cmd/server/main.go

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ========================================
    echo ✅ Build successful!
    echo ========================================
    echo.
    echo Starting server with detailed logs...
    echo Logs will be shown here and saved to server.log
    echo.
    echo Press Ctrl+C to stop the server
    echo.
    agario-server.exe 2>&1 | more
) else (
    echo.
    echo ========================================
    echo ❌ Build failed!
    echo ========================================
)

pause
