#!/bin/bash
# Paint Match AI — Database Setup Script
# Uses RTK for token-optimized output

set -e

DB_PATH="paint_knowledge.db"
MIGRATION="db/migrations/001_initial_schema.sql"

echo "=== Paint Match AI — Database Setup ==="

# 0. Clean start
echo "[0/5] Cleaning database..."
rm -f "$DB_PATH" "$DB_PATH-shm" "$DB_PATH-wal"

# 1. Apply migration
echo "[1/5] Applying migration..."
rtk sqlite3 "$DB_PATH" < "$MIGRATION" 2>&1 || true
rtk sqlite3 "$DB_PATH" "PRAGMA foreign_keys = ON; SELECT 'Migration applied: ' || COUNT(*) || ' tables' FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%';"

# 2. Populate manufacturers and paints
echo "[2/5] Populating manufacturers and paints..."
rtk go run main.go --import-paints 2>&1 | grep -E "\[seed\]" | head -20

# 3. Download manufacturer logos
echo "[3/5] Downloading manufacturer logos..."
rtk go run main.go --download-logos 2>&1 | grep -E "\[asset\]" | head -20

# 4. Generate paint swatches
echo "[4/5] Generating paint swatches..."
rtk go run main.go --generate-swatches 2>&1 | grep -E "\[swatch\]" | head -10

# 5. Verify results
echo "[5/5] Verifying database..."
echo "Manufacturers:"
rtk sqlite3 "$DB_PATH" "SELECT COUNT(*) || ' manufacturers' FROM manufacturers;"
echo "Product Lines:"
rtk sqlite3 "$DB_PATH" "SELECT COUNT(*) || ' product lines' FROM product_lines;"
echo "Paints:"
rtk sqlite3 "$DB_PATH" "SELECT COUNT(*) || ' paints' FROM paints;"
echo "Lookup Tables:"
rtk sqlite3 "$DB_PATH" "SELECT COUNT(*) || ' paint types' FROM paint_types;"
rtk sqlite3 "$DB_PATH" "SELECT COUNT(*) || ' finish types' FROM finish_types;"
rtk sqlite3 "$DB_PATH" "SELECT COUNT(*) || ' coverage types' FROM coverage_types;"
rtk sqlite3 "$DB_PATH" "SELECT COUNT(*) || ' opacity types' FROM opacity_types;"
echo "Swatches:"
rtk sqlite3 "$DB_PATH" "SELECT COUNT(*) || ' swatches' FROM paint_colors WHERE swatch_path IS NOT NULL;"

echo "=== Setup Complete ==="
