@echo off
REM tinyMem Native Windows Builder
REM This script must be run ON Windows to build binaries with embedded embeddings.
REM It uses precompiled llama_go.dll from kelindar/search.

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
REM Setup llama_go.dll from kelindar/search
REM ============================================================
echo Setting up llama_go.dll for embedded embeddings...

REM Detect CPU capabilities (simplified - defaults to AVX)
set VARIANT=avx
echo [OK] Using AVX variant (default for Windows)

REM Architecture
set ARCH=x64
if "%PROCESSOR_ARCHITECTURE%"=="AMD64" set ARCH=x64
if "%PROCESSOR_ARCHITECTURE%"=="ARM64" set ARCH=arm64

REM Check if kelindar/search is in go mod cache
for /f "tokens=*" %%p in ('go env GOMODCACHE') do set GO_MOD_CACHE=%%p
set SEARCH_PKG_PATH=%GO_MOD_CACHE%\github.com\kelindar\search@v0.4.0

if not exist "%SEARCH_PKG_PATH%" (
    echo [WARN] kelindar/search not in module cache, downloading...
    go mod download github.com/kelindar/search
)

REM Find the appropriate precompiled library
set DLL_SOURCE=%SEARCH_PKG_PATH%\dist\win-%ARCH%-%VARIANT%\llama_go.dll

if not exist "%DLL_SOURCE%" (
    echo [ERROR] Could not find precompiled library at:
    echo         %DLL_SOURCE%
    echo.
    echo Available libraries:
    dir /b "%SEARCH_PKG_PATH%\dist\" 2>nul
    exit /b 1
)

REM Copy to project directory
set DLL_DEST=%PROJECT_ROOT%\llama_go.dll
copy /Y "%DLL_SOURCE%" "%DLL_DEST%" >nul
echo [OK] Copied: %DLL_SOURCE%
echo [OK] To: %DLL_DEST%
echo.

REM ============================================================
REM Prepare Output Directory
REM ============================================================
set OUT_DIR=build\releases
if not exist "%OUT_DIR%" mkdir "%OUT_DIR%"

echo Building tinyMem for Windows...
echo.

REM ============================================================
REM Build Lightweight (No Embeddings)
REM ============================================================
echo [BUILD] Lightweight version (HTTP embeddings only)...

set LIGHTWEIGHT_OUTPUT=%OUT_DIR%\tinymem-windows-%ARCH%-lite.exe

set CGO_ENABLED=1
go build ^
  -tags "fts5" ^
  -ldflags "-X github.com/daverage/tinymem/internal/version.Version=!VERSION!" ^
  -o "%LIGHTWEIGHT_OUTPUT%" ^
  .\cmd\tinymem

if errorlevel 1 (
    echo [ERROR] Lightweight build failed
    exit /b 1
)

for %%F in ("%LIGHTWEIGHT_OUTPUT%") do set SIZE_LITE=%%~zF
set /a SIZE_LITE_MB=!SIZE_LITE! / 1048576
echo [OK] Lightweight: %LIGHTWEIGHT_OUTPUT% (!SIZE_LITE_MB! MB)

REM ============================================================
REM Build Full (With Embedded Embeddings)
REM ============================================================
echo [BUILD] Full version (embedded model)...

set FULL_OUTPUT=%OUT_DIR%\tinymem-windows-%ARCH%.exe

REM Add DLL to PATH for build
set PATH=%PROJECT_ROOT%;%PATH%

set CGO_ENABLED=1
go build ^
  -tags "fts5 embeddings" ^
  -ldflags "-X github.com/daverage/tinymem/internal/version.Version=!VERSION!" ^
  -o "%FULL_OUTPUT%" ^
  .\cmd\tinymem

if errorlevel 1 (
    echo [ERROR] Full build failed
    exit /b 1
)

for %%F in ("%FULL_OUTPUT%") do set SIZE_FULL=%%~zF
set /a SIZE_FULL_MB=!SIZE_FULL! / 1048576
echo [OK] Full: %FULL_OUTPUT% (!SIZE_FULL_MB! MB)

REM Copy DLL alongside binary for distribution
copy /Y "%DLL_DEST%" "%OUT_DIR%\llama_go.dll" >nul
echo [OK] Bundled: llama_go.dll (%VARIANT% variant)

echo.
echo ============================================================
echo   Build complete!
echo ============================================================
echo.
echo Artifacts in: %OUT_DIR%\
dir /b "%OUT_DIR%\tinymem*" "%OUT_DIR%\llama*"
echo.

echo [NOTE] Full build requires llama_go.dll at runtime.
echo        Copy both files together for distribution:
echo          - tinymem-windows-%ARCH%.exe
echo          - llama_go.dll
echo.

echo Test builds:
echo   %LIGHTWEIGHT_OUTPUT% version
echo   %FULL_OUTPUT% version
echo.

exit /b 0
