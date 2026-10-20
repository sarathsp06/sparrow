#!/bin/sh
# Installs the generated Node receiver's pinned dependencies in $1.
# LATEST=1 upgrades them to the newest releases (nightly run).
set -eu
cd "$1"
npm install --no-audit --no-fund --loglevel=error
if [ "${LATEST:-0}" = 1 ]; then
  npm install --no-audit --no-fund --loglevel=error express@latest @types/express@latest typescript@latest
fi
