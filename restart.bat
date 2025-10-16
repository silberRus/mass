@echo off
echo ========================================
echo    Restarting Agario (Kill + Build + Run)
echo ========================================

echo Stopping old processes...
taskkill /F /IM agario-server.exe >nul 2>&1
taskkill /F /IM node.exe >nul 2>&1
timeout /t 2 /nobreak >nul

echo.
echo Building server...
cd server
del agario-server.exe >nul 2>&1
go mod tidy
go build -o agario-server.exe cmd/server/main.go

if %ERRORLEVEL% NEQ 0 (
    echo ❌ Build failed!
    cd ..
    pause
    exit /b 1
)

echo ✅ Build successful!
cd ..

echo.
echo Starting server...
start "Agario Server" cmd /c "cd server && agario-server.exe"
timeout /t 2 /nobreak >nul

echo Starting client...
start "Agario Client" cmd /c "cd client-web && npm run dev"

echo.
echo ========================================
echo ✅ Restarted!
echo ========================================
echo Server: http://localhost:8080
echo Client: Check the client window for URL
echo.
echo Open browser, press F12, check Console tab!
echo.
pause
