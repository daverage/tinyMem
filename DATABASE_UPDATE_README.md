# TinyMem Database Update Instructions

This directory contains scripts to update your existing TinyMem database to the latest schema version (v6) which includes the new `classification` field and other improvements.

## What's New in Schema Version 6

- **Classification Field**: Optional field to improve recall precision (e.g., 'decision', 'constraint', 'glossary', 'invariant')
- **Memory Hygiene**: Better structure for organizing memories
- **Lifecycle Phases**: Formalized memory usage patterns
- **Recall Discipline**: Improved guidelines for efficient memory retrieval

## Update Process

### Prerequisites
- SQLite3 must be installed on your system
- You must run the update from the root of your project where the `.tinyMem/` directory is located

### For Unix-like Systems (Linux/macOS)

1. **Backup First**: The script automatically creates a backup of your database before making changes

2. **Run the Update Script**:
   ```bash
   cd /path/to/your/project  # Navigate to your project root
   /path/to/update_database.sh
   ```

3. **Verify the Update**:
   ```bash
   tinymem stats
   ```

### For Windows Systems

1. **Open PowerShell as Administrator** (recommended) or regular PowerShell

2. **Navigate to your project directory** where `.tinyMem\` is located:
   ```powershell
   cd C:\path\to\your\project
   ```

3. **Run the Update Script**:
   ```powershell
   .\update_database.ps1
   ```

   If you get an execution policy error, you may need to allow script execution:
   ```powershell
   Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
   ```

4. **Verify the Update**:
   ```cmd
   tinymem stats
   ```

## Manual Update (Alternative Method)

If you prefer to run the SQL manually:

1. Navigate to your project directory where `.tinyMem/store.sqlite3` is located
2. Run: `sqlite3 .tinyMem/store.sqlite3 < /path/to/update_database.sql`

## Rollback

If you need to rollback to the previous version:
1. Locate the backup file created during the update (named `store.sqlite3.backup.YYYYMMDD_HHMMSS`)
2. Replace the current database file with the backup:
   ```bash
   cp .tinyMem/store.sqlite3.backup.YYYYMMDD_HHMMSS .tinyMem/store.sqlite3
   ```

## Troubleshooting

### Unix-like Systems (Linux/macOS)
- If you get a "permission denied" error, make sure the script is executable: `chmod +x update_database.sh`
- If the update fails, restore from the backup and contact support
- Make sure you're running the script from the project root where `.tinyMem/` exists

### Windows Systems
- If you get an "execution policy" error, run: `Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser`
- If sqlite3 is not recognized, make sure it's installed and added to your PATH environment variable
- If you get permission errors, try running PowerShell as Administrator
- Make sure you're running the script from the project root where `.tinyMem\` exists

## Notes

- The update is backward compatible - older versions of TinyMem will still work with the updated database
- The new `classification` field is optional and won't affect existing memories
- All existing data is preserved during the update