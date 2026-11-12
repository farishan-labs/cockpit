#!/usr/bin/env bash
# Local installer: builds cockpit from source and installs to ~/.local/bin.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN_DIR="${HOME}/.local/bin"
BIN="${BIN_DIR}/cockpit"

mkdir -p "$BIN_DIR"

# Back up an existing non-binary cockpit (e.g. legacy herdr launcher) once.
if [ -e "$BIN" ] && [ ! -f "${BIN}.legacy" ]; then
  # only back up if it's not our ELF/binary
  if ! file "$BIN" | grep -qi 'elf\|mach-o'; then
    cp -f "$BIN" "${BIN}.legacy"
    echo "cockpit: backed up previous launcher to ${BIN}.legacy"
  fi
fi

echo "cockpit: building..."
cd "$ROOT"
GOWORK=off go build -o /tmp/cockpit-build ./cmd/cockpit

echo "cockpit: installing to $BIN"
install -m 0755 /tmp/cockpit-build "$BIN"
rm -f /tmp/cockpit-build

# tray indicator (optional, Ubuntu)
TRAY="${BIN_DIR}/cockpit-tray"
if [ -f "${ROOT}/scripts/cockpit-tray.py" ]; then
  install -m 0755 "${ROOT}/scripts/cockpit-tray.py" "$TRAY"
  echo "cockpit: tray indicator installed to $TRAY"
fi

echo "cockpit: installed."
case ":${PATH}:" in
  *":${BIN_DIR}:"*) ;;
  *)
    echo "cockpit: ${BIN_DIR} is not on PATH. Add it with:"
    echo "  export PATH=\"${BIN_DIR}:\$PATH\""
    ;;
esac