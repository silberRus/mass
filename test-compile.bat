@echo off
echo ========================================
echo    Testing Server Compilation
echo ========================================
cd server
echo.
echo Checking Go modules...
go mod tidy
echo.
echo Trying to compile...
go build -o test-server.exe cmd/server/main.go
if %ERRORLEVEL% EQU 0 (
    echo.
    echo ========================================
    echo SUCCESS! Server compiles without errors
    echo ========================================
    del test-server.exe
) else (
    echo.
    echo ========================================
    echo ERROR! Server has compilation errors
    echo See errors above
    echo ========================================
)
echo.
pause
