#!/bin/bash

# test-phase11.sh - Verify Phase 11 optimizations
# Usage: ./scripts/test-phase11.sh <input_image>

INPUT_IMAGE=${1:-"/Users/mac/WebApps/projects/pixo/tests/fixtures/rocket.png"}
CLI="./go-pixo-cli"
OUTPUT_DIR="test-output"

mkdir -p $OUTPUT_DIR

echo "=== Phase 11 Optimization Testing ==="
echo "Input: $INPUT_IMAGE"
echo ""

run_test() {
    NAME=$1
    FLAGS=$2
    OUTPUT="$OUTPUT_DIR/${NAME}.png"
    echo "Running: $NAME"
    $CLI -input "$INPUT_IMAGE" -output "$OUTPUT" $FLAGS -verbose > "$OUTPUT_DIR/${NAME}.log" 2>&1
    SIZE=$(stat -f%z "$OUTPUT")
    echo "  Size: $SIZE bytes"
}

# 1. Baseline
run_test "baseline" "-preset fast"

# 2. Filter Strategies
run_test "filter-minsum" "-filter-strategy minsum"
run_test "filter-entropy" "-filter-strategy entropy"
run_test "filter-bigrams" "-filter-strategy bigrams"
run_test "filter-parallel" "-filter-strategy parallel"

# 3. Zopfli Iterations
run_test "zopfli-5" "-zopfli -zopfli-iterations 5"
run_test "zopfli-10" "-zopfli -zopfli-iterations 10"

# 4. Optimal Deflate
run_test "optimal-deflate" "-optimal"

# 5. Lossy + Distance Metrics
run_test "lossy-euclidean" "-lossy -quality 50 -distance-metric euclidean"
run_test "lossy-redmean" "-lossy -quality 50 -distance-metric redmean -perceptual"

echo ""
echo "=== Results Summary ==="
ls -l $OUTPUT_DIR/*.png | awk '{print $9, $5}' | sort -k2 -n
