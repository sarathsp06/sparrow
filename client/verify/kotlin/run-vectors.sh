#!/usr/bin/env bash
# Compile and run the Kotlin SparrowVerify vectors test.
# Uses Docker: eclipse-temurin:21-jdk with kotlinc downloaded inside.
# Works from any cwd.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

docker run --rm \
  -v "$REPO_ROOT:/repo:ro" \
  -w /repo \
  eclipse-temurin:21-jdk \
  bash -c '
    set -euo pipefail

    # Install kotlinc (compiler only, no IDE)
    KOTLIN_VERSION=2.1.10
    cd /tmp
    curl -sLO "https://github.com/JetBrains/kotlin/releases/download/v${KOTLIN_VERSION}/kotlin-compiler-${KOTLIN_VERSION}.zip"
    apt-get update -qq && apt-get install -y -qq unzip >/dev/null 2>&1
    unzip -q "kotlin-compiler-${KOTLIN_VERSION}.zip"
    export PATH="/tmp/kotlinc/bin:$PATH"

    cd /repo
    mkdir -p /tmp/build
    kotlinc \
      client/verify/kotlin/SparrowVerify.kt \
      client/verify/kotlin/VectorsTest.kt \
      -d /tmp/build/vectors.jar \
      -nowarn 2>/dev/null

    kotlin -cp /tmp/build/vectors.jar sparrow.verify.VectorsTestKt pkg/signature/testdata/vectors.tsv
  '
