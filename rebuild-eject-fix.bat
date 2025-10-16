@echo off
echo ========================================
echo    Rebuilding Server (Eject Fix)
echo ========================================

echo Stopping server...
taskkill /F /IM agario-server.exe >nul 2>&1
timeout /t 1 /nobreak >nul

cd server

echo Cleaning old build...
del agario-server.exe >nul 2>&1

echo Building server...
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

echo.
echo ========================================
echo ✅ Server rebuilt and started!
echo ========================================
echo.
echo Changes:
echo - Eject (W) now throws EjectMass (12.0)
echo - Ejected food gives back full 12.0 mass
echo - Ejected food is visually larger
echo.
echo Test: Press W to eject - should see bigger food!
echo.
pause
