@echo off
REM tinyMem Build Script for Windows
REM Builds all platform binaries and places them in the releases folder

echo Building tinyMem binaries...

REM Create releases directory if it doesn't exist
if not exist releases mkdir releases

REM Build Windows AMD64
echo Building Windows AMD64...
set GOOS=windows
set GOARCH=amd64
go build -tags fts5 -o releases/tinymem-windows-amd64.exe ./cmd/tinymem
echo.✓ Built releases/tinymem-windows-amd64.exe

REM Build Windows ARM64
echo Building Windows ARM64...
set GOOS=windows
set GOARCH=arm64
go build -tags fts5 -o releases/tinymem-windows-arm64.exe ./cmd/tinymem
echo.✓ Built releases/tinymem-windows-arm64.exe

REM Build Linux AMD64
echo Building Linux AMD64...
set GOOS=linux
set GOARCH=amd64
go build -tags fts5 -o releases/tinymem-linux-amd64 ./cmd/tinymem
echo.✓ Built releases/tinymem-linux-amd64

REM Build Linux ARM64
echo Building Linux ARM64...
set GOOS=linux
set GOARCH=arm64
go build -tags fts5 -o releases/tinymem-linux-arm64 ./cmd/tinymem
echo.✓ Built releases/tinymem-linux-arm64

REM Build macOS AMD64
echo Building macOS AMD64...
set GOOS=darwin
set GOARCH=amd64
go build -tags fts5 -o releases/tinymem-darwin-amd64 ./cmd/tinymem
echo.✓ Built releases/tinymem-darwin-amd64

REM Build macOS ARM64
echo Building macOS ARM64...
set GOOS=darwin
set GOARCH=arm64
go build -tags fts5 -o releases/tinymem-darwin-arm64 ./cmd/tinymem
echo.✓ Built releases/tinymem-darwin-arm64

echo.
echo Build completed successfully!
echo.
echo Binaries created in releases\:
dir releases\
echo.
echo Build script completed.