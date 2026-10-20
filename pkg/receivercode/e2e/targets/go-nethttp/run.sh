#!/bin/sh
# Starts the receiver built by setup.sh in $1, in the foreground.
set -eu
cd "$1"
exec ./sparrow-receiver
