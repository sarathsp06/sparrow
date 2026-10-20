#!/bin/sh
# Builds the generated Go receiver in $1.
set -eu
cd "$1"
go build -o sparrow-receiver .
