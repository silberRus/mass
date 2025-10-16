@echo off
echo ========================================
echo    Starting Agario Server
echo ========================================
cd server

echo.
echo Cleaning old build...
if exist agario-server.exe del agario-server.exe

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
    echo Starting server...
    echo Server will listen on :8080
    echo Press Ctrl+C to stop
    echo.
    agario-server.exe
) else (
    echo.
    echo ========================================
    echo ❌ Build failed! See errors above.
    echo ========================================
    pause
)
