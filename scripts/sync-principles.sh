#!/usr/bin/env bash
# Sync cockpit/PRINCIPLES.md from the pinned principles submodule.
set -euo pipefail
cd "$(dirname "$0")/.."

if [ ! -d principles/.git ] && [ ! -f principles/.git ]; then
  echo "principles submodule not initialized. Run: git submodule update --init --recursive" >&2
  exit 1
fi

# Render the canonical combined view inside the submodule.
( cd principles && ./scripts/render.sh )

# Derive cockpit's PRINCIPLES.md from it.
cp principles/PRINCIPLES.md PRINCIPLES.md

# Record pinned version for traceability.
PINNED=$(cat principles/VERSION)
echo "Synced principles v${PINNED} -> PRINCIPLES.md"
