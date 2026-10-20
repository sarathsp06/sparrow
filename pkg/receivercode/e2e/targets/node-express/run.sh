#!/bin/sh
# Starts the generated Node receiver in $1, in the foreground.
set -eu
cd "$1"
exec node src/server.ts
