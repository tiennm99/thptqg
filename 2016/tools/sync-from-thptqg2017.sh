#!/usr/bin/env bash
# Re-sync xlsxread Rust source from thptqg2017.
#
# Source SHA: 8b4a755c115595bf1b937d749eb1133efb3a6e22 (chore/xlsxread-rust)
#
# Re-run this when xlsxread is updated upstream. The configs/ directory is
# NOT synced — it contains thptqg2016-specific configs and test stubs that
# must be maintained here independently.
#
# Usage:
#   ./tools/sync-from-thptqg2017.sh /path/to/thptqg2017
#
set -euo pipefail

SRC="${1:?Usage: $0 /path/to/thptqg2017}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEST="$SCRIPT_DIR/xlsxread"

if [ ! -d "$SRC/tools/xlsxread" ]; then
    echo "Error: $SRC/tools/xlsxread not found" >&2
    exit 1
fi

echo "Syncing from: $SRC/tools/xlsxread"
echo "Syncing to:   $DEST"
echo ""

# Sync source, tests, and Cargo manifests — exclude build artifacts and dataset configs
for item in src tests Cargo.toml Cargo.lock; do
    if [ -e "$SRC/tools/xlsxread/$item" ]; then
        cp -r "$SRC/tools/xlsxread/$item" "$DEST/"
        echo "  synced: $item"
    fi
done

echo ""
echo "Sync complete."
echo "Next steps:"
echo "  1. Review src/ for breaking changes to config.rs / writer.rs that"
echo "     may affect format_detect_2016.rs or the thptqg2016-data.toml config."
echo "  2. Rebuild: cargo build --release --manifest-path $DEST/Cargo.toml"
echo "  3. Test:    cargo test --manifest-path $DEST/Cargo.toml"
echo "  4. Commit on a chore/xlsxread-sync-... branch."
