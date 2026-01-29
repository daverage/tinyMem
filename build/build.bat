@echo off
REM tinyMem Build Script (Windows)
REM Builds platform binaries into build\releases (never tracked by git)

setlocal enabledelayedexpansion

REM ------------------------------------------------
REM Resolve project root
REM ------------------------------------------------
set SCRIPT_DIR=%~dp0
set PROJECT_ROOT=%SCRIPT_DIR%..
cd /d "%PROJECT_ROOT%"

set OUT_DIR=build\releases
if not exist "%OUT_DIR%" mkdir "%OUT_DIR%"

REM ------------------------------------------------
REM Determine version (prefer code, fallback to git)
REM ------------------------------------------------
set VERSION=

for /f "tokens=3 delims= " %%v in (
  'findstr /R "var Version = " internal\version\version.go'
) do (
  set VERSION=%%~v
)

REM Strip quotes
set VERSION=%VERSION:"=%

if "%VERSION%"=="" (
  for /f %%v in ('git describe --tags --always --dirty 2^>nul') do set VERSION=%%v
)

if "%VERSION%"=="" set VERSION=dev

echo Building tinyMem version: %VERSION%

REM ------------------------------------------------
REM Build tags
REM ------------------------------------------------
set BUILD_TAGS=fts5
if not "%TINYMEM_EXTRA_BUILD_TAGS%"=="" (
  set BUILD_TAGS=%BUILD_TAGS% %TINYMEM_EXTRA_BUILD_TAGS%
)

set TAGS_FLAG=-tags "%BUILD_TAGS%"
set LDFLAGS=-X github.com/andrzejmarczewski/tinyMem/internal/version.Version=%VERSION%

REM ------------------------------------------------
REM Build helper
REM ------------------------------------------------
call :build_target "Windows AMD64" windows amd64 "%OUT_DIR%\tinymem-windows-amd64.exe"
call :build_target "Windows ARM64" windows arm64 "%OUT_DIR%\tinymem-windows-arm64.exe"

echo.
echo Build complete. Artifacts:
dir "%OUT_DIR%"
exit /b 0

REM =================================================
REM Functions
REM =================================================
:build_target
set LABEL=%~1
set GOOS=%~2
set GOARCH=%~3
set OUTPUT=%~4

echo → %LABEL%
set CGO_ENABLED=1
set GOOS=%GOOS%
set GOARCH=%GOARCH%

go build %TAGS_FLAG% -ldflags "%LDFLAGS%" -o "%OUTPUT%" .\cmd\tinymem
if errorlevel 1 exit /b 1

echo ✓ Built %OUTPUT%
exit /b 0
