#!/usr/bin/env bash
# Install local git hooks for cockpit (commit attribution).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
HOOK="$ROOT/.git/hooks/prepare-commit-msg"
cat > "$HOOK" <<'EOF'
#!/bin/sh
MSG_FILE="$1"
if ! grep -q "Co-authored-by: FHL Director" "$MSG_FILE"; then
  printf "\nCo-authored-by: FHL Director <fhl-director@users.noreply.github.com>\n" >> "$MSG_FILE"
fi
EOF
chmod +x "$HOOK"
# Set commit identity to Faris Han (human operator) for this repo.
git -C "$ROOT" config user.name "Faris Han"
git -C "$ROOT" config user.email "farishanugraha@gmail.com"
echo "cockpit: hooks + commit identity configured"
