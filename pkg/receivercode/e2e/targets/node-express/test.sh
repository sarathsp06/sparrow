#!/bin/sh
# Runs the generated Node receiver's tests and type check in $1.
set -eu
cd "$1"
npm test
npm run typecheck
