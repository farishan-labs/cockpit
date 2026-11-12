#!/usr/bin/env bash
# Uninstaller for the locally installed cockpit binary.
set -euo pipefail

BIN="${HOME}/.local/bin/cockpit"

if [ -e "$BIN" ]; then
  rm -f "$BIN"
  echo "cockpit: removed $BIN"
else
  echo "cockpit: not installed at $BIN"
fi

if [ -f "${BIN}.legacy" ]; then
  echo "A legacy launcher backup exists at ${BIN}.legacy"
  echo "Restore it with: mv ${BIN}.legacy ${BIN}"
fi