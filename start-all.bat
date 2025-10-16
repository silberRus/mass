@echo off
echo ========================================
echo    Starting Agario Clone
echo ========================================
echo.

echo [1/2] Starting Server...
start "Agario Server" cmd /k "cd server && go mod tidy && go run cmd/server/main.go"
timeout /t 3 /nobreak > nul

echo [2/2] Starting Client...
start "Agario Client" cmd /k "cd client-web && npm install && npm run dev"

echo.
echo ========================================
echo Both server and client are starting...
echo Server: http://localhost:8080
echo Client: http://localhost:3000
echo ========================================
echo.
echo Press any key to exit...
pause > nul
