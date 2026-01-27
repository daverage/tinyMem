# TinyMem Database Update Script for Windows (PowerShell)
# This script updates an existing TinyMem database to the latest schema version (v6)
# It handles both fresh installations and incremental updates from older versions

Write-Host "TinyMem Database Update Script" -ForegroundColor Green
Write-Host "===============================" -ForegroundColor Green

# Check if sqlite3 is available
try {
    $sqliteCheck = Get-Command sqlite3 -ErrorAction Stop
    Write-Host "SQLite3 found: $($sqliteCheck.Path)" -ForegroundColor Green
} catch {
    Write-Host "Error: sqlite3 is required but not found in PATH." -ForegroundColor Red
    Write-Host "Please install SQLite3 and add it to your PATH, then try again." -ForegroundColor Red
    exit 1
}

# Find the TinyMem database file
$DbPath = ".tinyMem\store.sqlite3"

if (!(Test-Path $DbPath)) {
    Write-Host "Error: TinyMem database not found at $DbPath" -ForegroundColor Red
    Write-Host "Please run this script from the root of your project where .tinyMem\ is located." -ForegroundColor Red
    exit 1
}

Write-Host "Found TinyMem database at: $DbPath" -ForegroundColor Green

# Get current schema version
$CurrentVersion = sqlite3.exe $DbPath "PRAGMA user_version;"
Write-Host "Current schema version: $CurrentVersion" -ForegroundColor Yellow

# Backup the database before making changes
$Timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$BackupPath = "${DbPath}.backup.${Timestamp}"
Write-Host "Creating backup at: $BackupPath" -ForegroundColor Cyan
Copy-Item $DbPath $BackupPath

Write-Host "Applying database updates..." -ForegroundColor Cyan

# Get the directory of this script to locate the SQL file
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$SqlFilePath = Join-Path $ScriptDir "update_database.sql"

if (!(Test-Path $SqlFilePath)) {
    Write-Host "Error: update_database.sql not found at $SqlFilePath" -ForegroundColor Red
    exit 1
}

# Apply the update script
Get-Content $SqlFilePath | sqlite3.exe $DbPath

$NewVersion = sqlite3.exe $DbPath "PRAGMA user_version;"
Write-Host "New schema version: $NewVersion" -ForegroundColor Yellow

if ($NewVersion -ge 6) {
    Write-Host ""
    Write-Host "✅ Database update completed successfully!" -ForegroundColor Green
    Write-Host "Your TinyMem database is now at schema version $NewVersion" -ForegroundColor Green
    Write-Host ""
    Write-Host "💡 Backup of the original database saved at: $BackupPath" -ForegroundColor Cyan
    Write-Host "   Keep this backup until you've verified everything works correctly." -ForegroundColor Cyan
} else {
    Write-Host ""
    Write-Host "❌ Database update may not have completed successfully." -ForegroundColor Red
    Write-Host "Current schema version is $NewVersion, expected at least 6." -ForegroundColor Red
    Write-Host "Please check the database and restore from backup if needed." -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "To verify the update worked correctly, you can run:" -ForegroundColor White
Write-Host "  sqlite3.exe $DbPath `"SELECT * FROM memories LIMIT 5;`"" -ForegroundColor Gray
Write-Host "  tinymem stats" -ForegroundColor Gray