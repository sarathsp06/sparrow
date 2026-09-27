#!/usr/bin/env bash
# Compile and run the Java SparrowVerify vectors test.
# Uses Docker (eclipse-temurin:21-jdk) since a local JDK may not be present.
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
    mkdir -p /tmp/build/sparrow/verify
    cp client/verify/java/SparrowVerify.java /tmp/build/sparrow/verify/
    cp client/verify/java/VectorsTest.java   /tmp/build/sparrow/verify/
    javac /tmp/build/sparrow/verify/SparrowVerify.java /tmp/build/sparrow/verify/VectorsTest.java
    java -cp /tmp/build sparrow.verify.VectorsTest pkg/signature/testdata/vectors.tsv
  '
