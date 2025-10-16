@echo off
setlocal enabledelayedexpansion

echo ========================================
echo    Agario Clone - Complete Startup
echo ========================================
echo.

REM Проверяем что мы в правильной директории
if not exist "server" (
    echo ❌ Error: server directory not found
    echo Please run this from the agario project root
    pause
    exit /b 1
)

if not exist "client-web" (
    echo ❌ Error: client-web directory not found
    echo Please run this from the agario project root
    pause
    exit /b 1
)

echo ✅ Directories found
echo.

REM Останавливаем старые процессы
echo Cleaning up old processes...
taskkill /F /IM agario-server.exe >nul 2>&1
taskkill /F /IM node.exe >nul 2>&1
timeout /t 2 /nobreak >nul

echo.
echo ========================================
echo    Building Server
echo ========================================
cd server

echo Updating Go modules...
go mod tidy

echo Building server...
go build -o agario-server.exe cmd/server/main.go

if %ERRORLEVEL% NEQ 0 (
    echo.
    echo ❌ Server build failed!
    cd ..
    pause
    exit /b 1
)

echo ✅ Server built successfully
cd ..

echo.
echo ========================================
echo    Starting Server
echo ========================================
start "Agario Server" cmd /c "cd server && agario-server.exe"

echo Waiting for server to start...
timeout /t 3 /nobreak >nul

REM Проверяем что сервер запустился
netstat -ano | findstr :8080 >nul
if %ERRORLEVEL% EQU 0 (
    echo ✅ Server is running on port 8080
) else (
    echo ⚠️ Warning: Server might not be running on port 8080
    echo Check the server window for errors
)

echo.
echo ========================================
echo    Starting Client
echo ========================================
cd client-web

echo Installing dependencies (if needed)...
call npm install

echo Starting development server...
start "Agario Client" cmd /c "npm run dev"

cd ..

echo.
echo ========================================
echo ✅ Startup Complete!
echo ========================================
echo.
echo Server: http://localhost:8080
echo Client: http://localhost:3000 (or check the client window)
echo.
echo WebSocket: ws://localhost:8080/ws
echo.
echo Check the opened windows for any errors.
echo To stop: Close the server and client windows
echo.
echo Press F12 in browser to open DevTools and check:
echo - Console tab for client logs
echo - Network tab for WebSocket connection
echo.
pause
