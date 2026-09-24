#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

MASTER_SCALE="${MASTER_SCALE:-0.6666667}"
MASTER_FILE="${MASTER_FILE:-out/sparrow-promo-master-720p.mp4}"
FINAL_FILE="${FINAL_FILE:-out/sparrow-promo-web-720p.webm}"
CRF="${CRF:-40}"
AUDIO_BITRATE="${AUDIO_BITRATE:-32k}"

mkdir -p out

npx remotion render src/index.ts SparrowPromo "$MASTER_FILE" --scale="$MASTER_SCALE"

ffmpeg -y \
  -i "$MASTER_FILE" \
  -c:v libvpx-vp9 \
  -b:v 0 \
  -crf "$CRF" \
  -deadline good \
  -cpu-used 2 \
  -row-mt 1 \
  -tile-columns 2 \
  -frame-parallel 1 \
  -pix_fmt yuv420p \
  -g 240 \
  -c:a libopus \
  -b:a "$AUDIO_BITRATE" \
  -ac 1 \
  "$FINAL_FILE"

ls -lh "$MASTER_FILE" "$FINAL_FILE"
