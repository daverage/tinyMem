@echo off
REM tinyMem Release Automation Script for Windows
REM Usage: .\build\release.bat [major|minor|patch] (default: patch)

setlocal enabledelayedexpansion

set DIST_DIR=dist

REM ------------------------------------------------
REM 1. Ensure working directory is clean
REM ------------------------------------------------
git status -s > temp_status.txt
set /p STATUS=<temp_status.txt
del temp_status.txt

if not "%STATUS%"=="" (
    echo ❌ Error: Working directory is not clean.
    git status -s
    exit /b 1
)

REM ------------------------------------------------
REM 2. Get latest tag
REM ------------------------------------------------
for /f "tokens=*" %%i in ('git describe --tags --abbrev^=0 2^>nul') do set LATEST_TAG=%%i
if "%LATEST_TAG%"=="" set LATEST_TAG=v0.0.0
echo Current version: %LATEST_TAG%

REM ------------------------------------------------
REM 3. Calculate new version
REM ------------------------------------------------
set VERSION=%LATEST_TAG:~1%
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

REM ------------------------------------------------
REM 4. Confirm
REM ------------------------------------------------
echo.
echo ------------------------------------------------
echo 🚀 Ready to release: %NEW_TAG%
echo ------------------------------------------------
echo This will:
echo   1. Update internal/version/version.go
echo   2. Build binaries
echo   3. Commit changes
echo   4. Create git tag %NEW_TAG%
echo   5. Push main + tag
echo   6. Create GitHub Release with binaries
echo.
set /p CONTINUE=Continue? (y/N):
if /i not "%CONTINUE%"=="y" (
    echo Aborted.
    exit /b 1
)

REM ------------------------------------------------
REM 5. Update version.go
REM ------------------------------------------------
echo 📝 Updating version.go...
powershell -Command ^
  "(Get-Content internal/version/version.go) ^
   -replace 'var Version = \".*\"', 'var Version = \"%NEW_TAG%\"' ^
   | Set-Content internal/version/version.go"

REM ------------------------------------------------
REM 6. Build
REM ------------------------------------------------
echo 🔨 Building binaries...
if exist "%DIST_DIR%" rmdir /s /q "%DIST_DIR%"
call .\build\build.bat

if not exist "%DIST_DIR%" (
    echo ❌ Build did not produce %DIST_DIR%\
    exit /b 1
)

REM ------------------------------------------------
REM 7. Commit
REM ------------------------------------------------
echo 💾 Committing release...
git add .
git commit -m "Release %NEW_TAG%"

REM ------------------------------------------------
REM 8. Tag
REM ------------------------------------------------
echo 🏷️  Tagging %NEW_TAG%...
git tag -a "%NEW_TAG%" -m "Release %NEW_TAG%"

REM ------------------------------------------------
REM 9. Push
REM ------------------------------------------------
echo ⬆️  Pushing to origin...
git push origin main
git push origin "%NEW_TAG%"

REM ------------------------------------------------
REM 10. Create GitHub Release + upload binaries
REM ------------------------------------------------
echo 📦 Creating GitHub Release...

gh release create "%NEW_TAG%" ^
  --title "tinyMem %NEW_TAG%" ^
  --notes "Release %NEW_TAG%" ^
  "%DIST_DIR%\*"

echo.
echo ✅ Release %NEW_TAG% published successfully!
endlocal
