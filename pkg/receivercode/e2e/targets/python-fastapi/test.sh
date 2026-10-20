#!/bin/sh
# Runs the generated Python receiver's tests in $1.
set -eu
cd "$1"
.venv/bin/python -m pytest -q -p no:cacheprovider
