#!/usr/bin/env bash
# Idempotent installer for the 'cockpit' command.
set -euo pipefail

REPO_RAW="https://farishan-labs.github.io/cockpit"
BIN_DIR="${HOME}/.local/bin"

mkdir -p "$BIN_DIR"

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

echo "cockpit: downloading..."
curl -fsSL "${REPO_RAW}/scripts/cockpit" -o "$tmp"

if [ ! -s "$tmp" ]; then
  echo "cockpit: download failed (empty file)." >&2
  exit 1
fi

chmod +x "$tmp"
mv -f "$tmp" "${BIN_DIR}/cockpit"
trap - EXIT

echo "cockpit: installed to ${BIN_DIR}/cockpit"
echo "Try typing \`cockpit\` and press enter."

case ":${PATH}:" in
  *":${BIN_DIR}:"*) ;;
  *)
    echo "cockpit: ${BIN_DIR} is not on PATH. Add it with:"
    echo "  export PATH=\"${BIN_DIR}:\$PATH\""
    ;;
esac