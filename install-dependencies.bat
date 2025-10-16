@echo off
echo ========================================
echo    Installing Dependencies
echo ========================================
echo.

echo [1/2] Installing Go dependencies...
cd server
go mod tidy
echo Go dependencies installed!
cd ..
echo.

echo [2/2] Installing Node dependencies...
cd client-web
call npm install
echo Node dependencies installed!
cd ..
echo.

echo ========================================
echo Installation complete!
echo You can now run start-all.bat
echo ========================================
pause
