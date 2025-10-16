@echo off
echo ========================================
echo    Testing Server Connection
echo ========================================
echo.

echo Step 1: Checking if port 8080 is in use...
netstat -ano | findstr :8080
if %ERRORLEVEL% EQU 0 (
    echo ✅ Port 8080 is in use
) else (
    echo ❌ Port 8080 is NOT in use - server not running?
)

echo.
echo Step 2: Testing HTTP endpoint...
curl -v http://localhost:8080/test 2>&1
if %ERRORLEVEL% EQU 0 (
    echo ✅ HTTP endpoint works
) else (
    echo ❌ HTTP endpoint failed
)

echo.
echo Step 3: Testing health endpoint...
curl http://localhost:8080/health
echo.

echo.
echo ========================================
echo If you see errors above:
echo 1. Make sure server is running
echo 2. Run: start-server-simple.bat
echo 3. Then run this test again
echo ========================================
pause
