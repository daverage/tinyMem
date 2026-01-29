@echo off
REM tinyMem Build & Release Script (Windows)
REM Builds platform binaries and handles full release lifecycle if requested.
REM Usage:
REM   .\build\build.bat                 (Build only)
REM   .\build\build.bat [major|minor|patch] (Full release cycle)

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
REM Determine if we are in Release Mode
REM ------------------------------------------------
set MODE=%1
set IS_RELEASE=false
if "%MODE%"=="major" set IS_RELEASE=true
if "%MODE%"=="minor" set IS_RELEASE=true
if "%MODE%"=="patch" set IS_RELEASE=true

REM ------------------------------------------------
REM Safety checks for Release Mode
REM ------------------------------------------------
if "%IS_RELEASE%"=="true" (
    where gh >nul 2>nul
    if errorlevel 1 (
        echo ❌ Error: GitHub CLI (gh) not installed. Required for releases.
        exit /b 1
    )
)

REM ------------------------------------------------
REM Get latest tag
REM ------------------------------------------------
for /f "tokens=*" %%i in ('git describe --tags --abbrev^=0 2^>nul') do set LATEST_TAG=%%i
if "%LATEST_TAG%"=="" set LATEST_TAG=v0.0.0

REM ------------------------------------------------
REM Version calculation
REM ------------------------------------------------
if "%IS_RELEASE%"=="true" (
    set VERSION_STR=%LATEST_TAG:~1%
    for /f "tokens=1,2,3 delims=." %%a in ("!VERSION_STR!") do (
        set MAJOR=%%a
        set MINOR=%%b
        set PATCH=%%c
    )

    if "%MODE%"=="major" (
        set /a MAJOR+=1
        set MINOR=0
        set PATCH=0
    ) else if "%MODE%"=="minor" (
        set /a MINOR+=1
        set PATCH=0
    ) else (
        set /a PATCH+=1
    )

    set VERSION=v!MAJOR!.!MINOR!.!PATCH!
    echo 🚀 Preparing Release: !VERSION! (Current: %LATEST_TAG%)
) else (
    REM Read current version from code
    for /f "tokens=3 delims= " %%v in ('findstr /R "var Version = " internal\version\version.go') do (
        set VERSION=%%~v
    )
    set VERSION=!VERSION:"=!
    if "!VERSION!"=="" set VERSION=%LATEST_TAG%
    echo Building tinyMem version: !VERSION!
)

REM ------------------------------------------------
REM Build Logic
REM ------------------------------------------------
set BUILD_TAGS=fts5
if not "%TINYMEM_EXTRA_BUILD_TAGS%"=="" (
  set BUILD_TAGS=%BUILD_TAGS% %TINYMEM_EXTRA_BUILD_TAGS%
)

set TAGS_FLAG=-tags "%BUILD_TAGS%"
set LDFLAGS=-X github.com/daverage/tinymem/internal/version.Version=%VERSION%

echo → Windows AMD64
set CGO_ENABLED=1
set GOOS=windows
set GOARCH=amd64
go build %TAGS_FLAG% -ldflags "%LDFLAGS%" -o "%OUT_DIR%\tinymem-windows-amd64.exe" .\cmd\tinymem

echo → Windows ARM64
set CGO_ENABLED=1
set GOOS=windows
set GOARCH=arm64
go build %TAGS_FLAG% -ldflags "%LDFLAGS%" -o "%OUT_DIR%\tinymem-windows-arm64.exe" .\cmd\tinymem

REM ------------------------------------------------
REM Finalize Release
REM ------------------------------------------------
if "%IS_RELEASE%"=="true" (
    echo.
    set /p COMMIT_MSG=Build successful. Commit message for !VERSION!: 
    if "!COMMIT_MSG!"=="" (
        echo ❌ Error: Commit message required.
        exit /b 1
    )

    echo 📝 Updating internal/version/version.go...
    powershell -Command ^
      "(Get-Content internal/version/version.go) ^
       -replace 'var Version = ".*"', 'var Version = "!VERSION!"' ^
       | Set-Content internal/version/version.go"

    echo 💾 Committing changes...
    git add .
    git commit -m "!COMMIT_MSG! (Release !VERSION!)" || echo No changes to commit.

    REM Check if tag exists
    git rev-parse !VERSION! >nul 2>nul
    if not errorlevel 1 (
        echo ⚠️  Tag !VERSION! already exists locally. Updating...
        git tag -d !VERSION!
    )

    echo 🏷️  Tagging !VERSION!...
    git tag -a "!VERSION!" -m "!COMMIT_MSG!"

    echo ⬆️  Pushing to origin...
    git push origin main
    git push origin "!VERSION!" --force

    echo 📦 Creating GitHub Release...
    gh release view "!VERSION!" >nul 2>nul
    if not errorlevel 1 (
        echo ⚠️  Release !VERSION! already exists. Uploading assets...
        gh release upload "!VERSION!" "%OUT_DIR%\*" --clobber
    ) else (
        gh release create "!VERSION!" ^
          --title "tinyMem !VERSION!" ^
          --notes "!COMMIT_MSG!" ^
          "%OUT_DIR%\*"
    )

    echo.
    echo ✅ Release !VERSION! processed successfully!
) else (
    echo.
    echo Build complete. Artifacts in %OUT_DIR%
)

exit /b 0
