@echo off
REM tinyMem Release Automation Script for Windows
REM Usage: .\build\release.bat [major|minor|patch] (default: patch)

setlocal enabledelayedexpansion

REM 1. Ensure working directory is clean
git status -s > temp_status.txt
set /p STATUS=<temp_status.txt
del temp_status.txt
if not "%STATUS%"=="" (
    echo ❌ Error: Working directory is not clean. Please commit or stash changes first.
    git status -s
    exit /b 1
)

REM 2. Get latest tag
for /f "tokens=*" %%i in ('git describe --tags --abbrev^=0 2^>nul') do set LATEST_TAG=%%i
if "%LATEST_TAG%"=="" set LATEST_TAG=v0.0.0
echo Current version: %LATEST_TAG%

REM 3. Calculate new version
set VERSION=%LATEST_TAG%:~1%
for /f "tokens=1,2,3 delims=." %%a in ("%VERSION%") do (
    set MAJOR=%%a
    set MINOR=%%b
    set PATCH=%%c
)

set MODE=%1
if "%MODE%"=="" set MODE=patch

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

set NEW_TAG=v%MAJOR%.%MINOR%.%PATCH%

REM 4. Confirm with user
echo.
echo ------------------------------------------------
echo 🚀 Ready to release: %NEW_TAG%
echo ------------------------------------------------
echo This will:
echo   1. Update internal/version/version.go to %NEW_TAG%
echo   2. Run .\build\build.bat to generate binaries
echo   3. Commit all changes and built binaries
echo   4. Create git tag %NEW_TAG%
echo   5. Push 'main' and tags to origin
echo.
set /p CONTINUE=Continue? (y/N): 
if /i not "%CONTINUE%"=="y" (
    echo Aborted.
    exit /b 1
)

REM 5. Update version.go
echo 📝 Updating version.go to %NEW_TAG%...
REM Use powershell to perform the replacement since Windows doesn't have sed
powershell -Command "(Get-Content internal/version/version.go) -replace 'var Version = ".*"', 'var Version = "%NEW_TAG%"' | Set-Content internal/version/version.go"

REM 6. Build
echo 🔨 Building binaries...
call .\build\build.bat

REM 7. Commit changes
echo 💾 Preparing commit...
git add .
set /p COMMIT_MSG=Enter commit message (default: Release %NEW_TAG%): 
if "%COMMIT_MSG%"=="" set COMMIT_MSG=Release %NEW_TAG%

git commit -m "%COMMIT_MSG%"

REM 8. Create Tag
echo 🏷️  Tagging %NEW_TAG%...
git tag -a "%NEW_TAG%" -m "Release %NEW_TAG%"

REM 9. Push
echo ⬆️  Pushing to origin...
git push origin main
git push origin "%NEW_TAG%"

echo.
echo ✅ Release %NEW_TAG% completed successfully!
endlocal
