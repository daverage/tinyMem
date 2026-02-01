@echo off
REM tinyMem Native Windows Builder
REM This script builds tinyMem binaries on Windows.

setlocal enabledelayedexpansion

echo ============================================================
echo   tinyMem Native Windows Builder
echo ============================================================
echo.

REM ============================================================
REM Platform Check
REM ============================================================
if not "%OS%"=="Windows_NT" (
    echo [ERROR] This script must be run on Windows.
    echo         For macOS, use build-macos.sh
    echo         For Linux, use build-linux.sh
    exit /b 1
)

echo [OK] Platform: Windows
echo.

REM ============================================================
REM Resolve Project Root
REM ============================================================
set SCRIPT_DIR=%~dp0
set PROJECT_ROOT=%SCRIPT_DIR%..
cd /d "%PROJECT_ROOT%"

REM ============================================================
REM Version Detection
REM ============================================================
for /f "tokens=*" %%v in ('git describe --tags --dirty --always 2^>nul') do (
    set VERSION=%%v
)
if "!VERSION!"=="" (
    for /f "tokens=3 delims= " %%v in ('findstr /R "var Version = " internal\version\version.go') do (
        set VERSION=%%~v
    )
    set VERSION=!VERSION:"=!
)
if "!VERSION!"=="" set VERSION=dev

echo [OK] Version: !VERSION!
echo.

REM ============================================================
REM Check Build Dependencies
REM ============================================================
echo Checking build dependencies...

where go >nul 2>nul
if errorlevel 1 (
    echo [ERROR] Go not found. Please install Go 1.21 or later.
    exit /b 1
)
for /f "tokens=3" %%v in ('go version') do echo [OK] Go %%v

where git >nul 2>nul
if errorlevel 1 (
    echo [ERROR] Git not found. Please install Git.
    exit /b 1
)
for /f "tokens=3" %%v in ('git --version') do echo [OK] Git %%v

echo.

REM ============================================================
REM Prepare Output Directory
REM ============================================================
set OUT_DIR=build\releases
if not exist "%OUT_DIR%" mkdir "%OUT_DIR%"

echo Building tinyMem for Windows...
echo.

REM ============================================================
REM Build with FTS5 Lexical Recall
REM ============================================================
echo [BUILD] Building tinyMem (FTS5 + CoVe + Evidence + Mode Enforcement)...

set ARCH=x64
if "%PROCESSOR_ARCHITECTURE%"=="AMD64" set ARCH=x64
if "%PROCESSOR_ARCHITECTURE%"=="ARM64" set ARCH=arm64

set OUTPUT=%OUT_DIR%\tinymem-windows-%ARCH%.exe

set CGO_ENABLED=1
go build ^
  -tags "fts5" ^
  -ldflags "-X github.com/daverage/tinymem/internal/version.Version=!VERSION!" ^
  -o "%OUTPUT%" ^
  .\cmd\tinymem

if errorlevel 1 (
    echo [ERROR] Build failed
    exit /b 1
)

for %%F in ("%OUTPUT%") do set SIZE=%%~zF
set /a SIZE_MB=!SIZE! / 1048576
echo [OK] Built: %OUTPUT% (!SIZE_MB! MB)

echo.
echo ============================================================
echo   Build complete!
echo ============================================================
echo.
echo Artifacts in: %OUT_DIR%\
dir /b "%OUT_DIR%\tinymem*"
echo.

echo Test build:
echo   %OUTPUT% version
echo.

exit /b 0
