#!/bin/sh
# Starts the generated Python receiver in $1, in the foreground.
set -eu
cd "$1"
exec .venv/bin/python receiver.py
