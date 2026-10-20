#!/bin/sh
# Installs the generated Python receiver's pinned dependencies in $1/.venv.
# LATEST=1 upgrades them to the newest releases (nightly run).
set -eu
cd "$1"
"${PYTHON:-python3}" -m venv .venv
.venv/bin/pip install -q -r requirements.txt
if [ "${LATEST:-0}" = 1 ]; then
  .venv/bin/pip install -q --upgrade fastapi uvicorn pydantic cryptography pytest httpx2
fi
