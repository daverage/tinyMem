@echo off
REM tinyMem Build & Release Script (Windows)
REM Builds platform binaries and handles full release lifecycle if requested.
REM
REM Usage:
REM   .\build\build.bat                          (Lightweight build ~50-60 MB)
REM   .\build\build.bat [major|minor|patch]      (Full release cycle)
REM
REM Build Options:
REM   Default: Lightweight build with FTS5 only
REM     - Binary size: ~50-60 MB
REM     - Semantic search requires HTTP endpoint (Ollama, etc.)
REM     - All features enabled (CoVe, Ralph, semantic)
REM
REM   Full build with embedded model:
REM     set TINYMEM_EXTRA_BUILD_TAGS=embeddings
REM     .\build\build.bat
REM     - Binary size: ~150-200 MB
REM     - Includes embedded nomic-embed-text-v1.5 model
REM     - Semantic search works offline, no external service needed
REM     - Model file must exist: internal\embedding\models\nomic-embed-text-v1.5.Q4_K_M.gguf
REM
REM   Minimal build (no LLM features):
REM     set TINYMEM_EXTRA_BUILD_TAGS=nollm
REM     .\build\build.bat
REM     - Binary size: ~14 MB
REM     - Disables CoVe and Ralph (which require text generation LLMs)
REM     - Semantic search still available via HTTP endpoint
REM     - Useful for constrained environments or MCP-only deployments
REM
REM   Semantic-only build (embedded search, no LLM features):
REM     set TINYMEM_EXTRA_BUILD_TAGS=embeddings nollm
REM     .\build\build.bat
REM     - Binary size: ~95 MB
REM     - Embedded semantic search works offline
REM     - CoVe and Ralph disabled
REM
REM Examples:
REM   .\build\build.bat                                               # Lightweight build
REM   set TINYMEM_EXTRA_BUILD_TAGS=embeddings && .\build\build.bat    # Full build with embeddings
REM   set TINYMEM_EXTRA_BUILD_TAGS=nollm && .\build\build.bat         # Minimal build
REM   set TINYMEM_EXTRA_BUILD_TAGS=embeddings nollm && .\build\build.bat  # Semantic-only build

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
    git status -s > temp_status.txt
    set /p STATUS=<temp_status.txt
    del temp_status.txt
    if not "!STATUS!"=="" (
        echo ❌ Error: Working directory is not clean. Commit or stash changes before releasing.
        git status -s
        exit /b 1
    )
    
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
    REM Prefer git tag for local builds
    for /f "tokens=*" %%v in ('git describe --tags --dirty --always 2^>nul') do (
        set VERSION=%%v
    )
    if "!VERSION!"=="" (
        REM Read current version from code
        for /f "tokens=3 delims= " %%v in ('findstr /R "var Version = " internal\version\version.go') do (
            set VERSION=%%~v
        )
        set VERSION=!VERSION:"=!
    )
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

REM Clear previous releases
if exist "%OUT_DIR%\*" del /q "%OUT_DIR%\*"

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
REM Cross-compilation (if Zig is present)
REM ------------------------------------------------
where zig >nul 2>nul
if not errorlevel 1 (
    echo → macOS ARM64 (Cross-compiling via zig cc)
    set CGO_ENABLED=1
    set GOOS=darwin
    set GOARCH=arm64
    set CC=zig cc -target aarch64-macos
    go build %TAGS_FLAG% -ldflags "%LDFLAGS%" -o "%OUT_DIR%\tinymem-darwin-arm64" .\cmd\tinymem

    echo → macOS AMD64 (Cross-compiling via zig cc)
    set CGO_ENABLED=1
    set GOOS=darwin
    set GOARCH=amd64
    set CC=zig cc -target x86_64-macos
    go build %TAGS_FLAG% -ldflags "%LDFLAGS%" -o "%OUT_DIR%\tinymem-darwin-amd64" .\cmd\tinymem

    echo → Linux AMD64 (Cross-compiling via zig cc)
    set CGO_ENABLED=1
    set GOOS=linux
    set GOARCH=amd64
    set CC=zig cc -target x86_64-linux-musl
    go build %TAGS_FLAG% -ldflags "%LDFLAGS%" -o "%OUT_DIR%\tinymem-linux-amd64" .\cmd\tinymem

    echo → Linux ARM64 (Cross-compiling via zig cc)
    set CGO_ENABLED=1
    set GOOS=linux
    set GOARCH=arm64
    set CC=zig cc -target aarch64-linux-musl
    go build %TAGS_FLAG% -ldflags "%LDFLAGS%" -o "%OUT_DIR%\tinymem-linux-arm64" .\cmd\tinymem
    
    set CC=
) else (
    echo Skipping cross-compilation (zig not found). To enable: winget install zig.zig
)

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

    echo.
    echo ✅ Release !VERSION! processed successfully!
) else (
    echo.
    echo Build complete. Artifacts in %OUT_DIR%
)

exit /b 0
