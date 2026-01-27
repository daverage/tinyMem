#!/bin/bash

# TinyMem Database Update Script
# This script updates an existing TinyMem database to the latest schema version (v6)
# It handles both fresh installations and incremental updates from older versions

set -e  # Exit on any error

echo "TinyMem Database Update Script"
echo "==============================="

# Check if sqlite3 is available
if ! command -v sqlite3 &> /dev/null; then
    echo "Error: sqlite3 is required but not installed."
    echo "Please install sqlite3 and try again."
    exit 1
fi

# Find the TinyMem database file
DB_PATH=".tinyMem/store.sqlite3"

if [ ! -f "$DB_PATH" ]; then
    echo "Error: TinyMem database not found at $DB_PATH"
    echo "Please run this script from the root of your project where .tinyMem/ is located."
    exit 1
fi

echo "Found TinyMem database at: $DB_PATH"

# Get current schema version
CURRENT_VERSION=$(sqlite3 "$DB_PATH" "PRAGMA user_version;")
echo "Current schema version: $CURRENT_VERSION"

# Backup the database before making changes
BACKUP_PATH="${DB_PATH}.backup.$(date +%Y%m%d_%H%M%S)"
echo "Creating backup at: $BACKUP_PATH"
cp "$DB_PATH" "$BACKUP_PATH"

echo "Applying database updates..."

# Apply the update script
sqlite3 "$DB_PATH" < update_database.sql

NEW_VERSION=$(sqlite3 "$DB_PATH" "PRAGMA user_version;")
echo "New schema version: $NEW_VERSION"

if [ "$NEW_VERSION" -ge 6 ]; then
    echo ""
    echo "✅ Database update completed successfully!"
    echo "Your TinyMem database is now at schema version $NEW_VERSION"
    echo ""
    echo "💡 Backup of the original database saved at: $BACKUP_PATH"
    echo "   Keep this backup until you've verified everything works correctly."
else
    echo ""
    echo "❌ Database update may not have completed successfully."
    echo "Current schema version is $NEW_VERSION, expected at least 6."
    echo "Please check the database and restore from backup if needed."
    exit 1
fi

echo ""
echo "To verify the update worked correctly, you can run:"
echo "  sqlite3 $DB_PATH \"SELECT * FROM memories LIMIT 5;\""
echo "  tinymem stats"