@echo off
echo ========================================
echo    Testing Deadlock Fix
echo ========================================
cd server
echo.
echo Step 1: Updating Go modules...
go mod tidy
echo.
echo Step 2: Compiling server...
go build -o test-server.exe cmd/server/main.go
if %ERRORLEVEL% EQU 0 (
    echo.
    echo ========================================
    echo ✅ SUCCESS! Server compiles without errors
    echo ========================================
    echo.
    echo Step 3: Running server for 10 seconds...
    echo Press Ctrl+C to stop early if needed
    echo.
    start /B test-server.exe
    timeout /t 10 /nobreak
    taskkill /F /IM test-server.exe >nul 2>&1
    del test-server.exe
    echo.
    echo Server test completed!
) else (
    echo.
    echo ========================================
    echo ❌ ERROR! Server has compilation errors
    echo See errors above
    echo ========================================
)
echo.
pause
