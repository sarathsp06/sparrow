#!/bin/sh
# Install or update the Sparrow CLI (or another Sparrow binary) from a GitHub
# release. Re-running it updates: an existing binary on PATH is replaced in
# place, and nothing is downloaded when it is already at the requested version.
#
#   curl -fsSL https://raw.githubusercontent.com/sarathsp06/sparrow/main/scripts/install.sh | sh
#
# Options (flags or environment variables):
#   -v, --version VERSION   release to install, e.g. v0.8.2 (default: latest)   SPARROW_VERSION
#   -b, --bin-dir DIR       where to put the binary (default: the directory of    SPARROW_INSTALL_DIR
#                           the existing install, else /usr/local/bin if
#                           writable, else ~/.local/bin)
#   -c, --component NAME    cli (default), server, sinks or sources             SPARROW_COMPONENT
#   -f, --force             reinstall even when already at that version        SPARROW_FORCE=1
#   -h, --help
#
# Pass flags through curl with:  curl -fsSL ... | sh -s -- -v v0.8.2
#
# The archive's SHA-256 is verified against the release's checksums.txt.
# macOS and Linux only (amd64, arm64); on Windows download the .zip from
# https://github.com/sarathsp06/sparrow/releases.
set -eu

REPO="sarathsp06/sparrow"
VERSION="${SPARROW_VERSION:-}"
BIN_DIR="${SPARROW_INSTALL_DIR:-}"
COMPONENT="${SPARROW_COMPONENT:-cli}"
FORCE="${SPARROW_FORCE:-}"

usage() {
  echo "usage: install.sh [-v VERSION] [-b BIN_DIR] [-c cli|server|sinks|sources] [-f]"
  echo "  env: SPARROW_VERSION, SPARROW_INSTALL_DIR, SPARROW_COMPONENT, SPARROW_FORCE"
  echo "  re-run to update; the existing binary on PATH is replaced in place"
}

while [ $# -gt 0 ]; do
  case "$1" in
    -v|--version) VERSION="$2"; shift 2 ;;
    -b|--bin-dir) BIN_DIR="$2"; shift 2 ;;
    -c|--component) COMPONENT="$2"; shift 2 ;;
    -f|--force) FORCE=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "install.sh: unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
done

fail() { echo "install.sh: $*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || fail "'$1' is required"; }

need curl
need tar

# --- which archive --------------------------------------------------------
# Release archives are named <prefix>-<version>-<os>-<arch>.tar.gz and every
# one of them contains the binary at its root.
case "$COMPONENT" in
  cli)     PREFIX="sparrow-cli";     BINARY="sparrow" ;;
  server)  PREFIX="sparrow";         BINARY="sparrow" ;;
  sinks)   PREFIX="sparrow-sinks";   BINARY="sparrow-sinks" ;;
  sources) PREFIX="sparrow-sources"; BINARY="sparrow-sources" ;;
  *) fail "unknown component '$COMPONENT' (cli, server, sinks or sources)" ;;
esac

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux|darwin) ;;
  mingw*|msys*|cygwin*|windows*) fail "Windows: download the .zip from https://github.com/$REPO/releases/latest" ;;
  *) fail "unsupported OS: $OS" ;;
esac

ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) fail "unsupported architecture: $ARCH" ;;
esac

# --- which version --------------------------------------------------------
# The releases/latest redirect carries the tag, so no GitHub API call (and
# no API rate limit) is needed.
if [ -z "$VERSION" ]; then
  VERSION=$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$REPO/releases/latest" | sed 's#.*/tag/##')
  [ -n "$VERSION" ] || fail "could not determine the latest release"
fi
case "$VERSION" in v*) ;; *) VERSION="v$VERSION" ;; esac
BARE="${VERSION#v}"

ARCHIVE="$PREFIX-$BARE-$OS-$ARCH.tar.gz"
BASE="https://github.com/$REPO/releases/download/$VERSION"

# --- where to put it ------------------------------------------------------
# An update goes where the current install lives, so re-running the script
# never leaves two copies on PATH.
EXISTING=$(command -v "$BINARY" 2>/dev/null || true)
if [ -z "$BIN_DIR" ]; then
  if [ -n "$EXISTING" ] && [ -f "$EXISTING" ]; then
    BIN_DIR=$(dirname "$EXISTING")
  elif [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
    BIN_DIR=/usr/local/bin
  else
    BIN_DIR="$HOME/.local/bin"
  fi
fi
mkdir -p "$BIN_DIR" || fail "cannot create $BIN_DIR"
[ -w "$BIN_DIR" ] || fail "$BIN_DIR is not writable; rerun with -b <dir> or as a user who can write there"

# --- already up to date? --------------------------------------------------
# The CLI prints "sparrow <version>" via `version`; the first x.y.z in that
# output is the installed version. The server, sinks and sources binaries
# have no version command, so for them the version is unknown and the
# requested release is always (re)installed.
CURRENT=""
if [ -x "$BIN_DIR/$BINARY" ]; then
  CURRENT=$( { "$BIN_DIR/$BINARY" version 2>/dev/null || "$BIN_DIR/$BINARY" --version 2>/dev/null || true; } \
    | head -n 1 | sed -n 's/.*[^0-9.]\([0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\).*/\1/p')
fi
if [ -z "$FORCE" ] && [ "$CURRENT" = "$BARE" ]; then
  echo "$BINARY $VERSION is already installed at $BIN_DIR/$BINARY (use -f to reinstall)"
  exit 0
fi
if [ -n "$CURRENT" ]; then
  echo "Updating $BINARY v$CURRENT -> $VERSION in $BIN_DIR"
elif [ -x "$BIN_DIR/$BINARY" ]; then
  echo "Replacing $BIN_DIR/$BINARY (unknown version) with $VERSION"
fi

# --- download and verify --------------------------------------------------
TMP=$(mktemp -d 2>/dev/null || mktemp -d -t sparrow)
trap 'rm -rf "$TMP"' EXIT INT TERM

echo "Downloading $ARCHIVE ($VERSION)..."
curl -fsSL -o "$TMP/$ARCHIVE" "$BASE/$ARCHIVE" || fail "download failed: $BASE/$ARCHIVE (no build for $OS/$ARCH in $VERSION?)"
curl -fsSL -o "$TMP/checksums.txt" "$BASE/checksums.txt" || fail "download failed: $BASE/checksums.txt"

EXPECTED=$(grep " $ARCHIVE\$" "$TMP/checksums.txt" | awk '{print $1}')
[ -n "$EXPECTED" ] || fail "$ARCHIVE is not listed in checksums.txt"
if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL=$(sha256sum "$TMP/$ARCHIVE" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
  ACTUAL=$(shasum -a 256 "$TMP/$ARCHIVE" | awk '{print $1}')
else
  fail "neither sha256sum nor shasum is available to verify the download"
fi
[ "$ACTUAL" = "$EXPECTED" ] || fail "checksum mismatch for $ARCHIVE (expected $EXPECTED, got $ACTUAL)"

tar -xzf "$TMP/$ARCHIVE" -C "$TMP" "$BINARY" || fail "could not extract $BINARY from $ARCHIVE"
chmod 0755 "$TMP/$BINARY"
# Replace atomically so a running copy is not overwritten in place.
mv -f "$TMP/$BINARY" "$BIN_DIR/$BINARY.tmp.$$" && mv -f "$BIN_DIR/$BINARY.tmp.$$" "$BIN_DIR/$BINARY"

if [ -n "$CURRENT" ]; then
  echo "Updated $BINARY to $VERSION at $BIN_DIR/$BINARY"
else
  echo "Installed $BINARY $VERSION to $BIN_DIR/$BINARY"
fi
case ":$PATH:" in
  *":$BIN_DIR:"*) ;;
  *) echo "Note: $BIN_DIR is not on your PATH. Add it, e.g.:"
     echo "  export PATH=\"$BIN_DIR:\$PATH\"" ;;
esac
if [ "$COMPONENT" = cli ]; then
  echo "Next: sparrow init --url http://localhost:8080"
fi
