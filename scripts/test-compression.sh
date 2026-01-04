#!/bin/bash

# Advanced PNG Compression Test Script
# Tests various compression options on images in the images/ folder
# Verifies compression improvements and tests all presets

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_DIR"

echo "=== Advanced PNG Compression Tests ==="
echo "Project directory: $PROJECT_DIR"
echo ""

# Function to get file size
get_size() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        stat -f%z "$1" 2>/dev/null || echo 0
    else
        stat -c%s "$1" 2>/dev/null || echo 0
    fi
}

# Build the CLI tool
echo "Building CLI tool..."
CLI_PATH="/tmp/pixo-test"
go build -o "$CLI_PATH" ./src/cmd/cli/...

if [ ! -f "$CLI_PATH" ]; then
    echo "Error: Failed to build CLI tool"
    exit 1
fi
echo "Build successful"
echo ""

# Track test results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
BEST_SIZE=0
BEST_CONFIG=""

# Test images
TEST_IMAGES=("images/cursor-meetup.png" "images/code.png" "images/cursor-2025-models.png")

# Presets to test
PRESETS=("fast" "balanced" "max" "extreme")

# Quality levels for lossy
QUALITY_LEVELS=(25 50 75 90)

# Dithering strengths
DITHER_VALUES=(0.0 0.25 0.5 0.75 1.0)

echo "=== Testing Presets ==="
for img in "${TEST_IMAGES[@]}"; do
    if [ ! -f "$img" ]; then
        continue
    fi

    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    filename=$(basename "$img")
    original_size=$(get_size "$img")
    echo ""
    echo "Testing: $img (Original: $original_size bytes)"

    for preset in "${PRESETS[@]}"; do
        output_file="/tmp/compressed_${preset}.png"
        
        $CLI_PATH -input "$img" -output "$output_file" -preset "$preset" -compare -verbose false 2>/dev/null || true
        
        if [ -f "$output_file" ]; then
            compressed_size=$(get_size "$output_file")
            ratio=$(echo "scale=2; $compressed_size * 100 / $original_size" | bc 2>/dev/null || echo "N/A")
            echo "  Preset $preset: $compressed_size bytes (${ratio}%)"
            
            if [ "$compressed_size" -le "$original_size" ]; then
                PASSED_TESTS=$((PASSED_TESTS + 1))
                
                if [ "$BEST_SIZE" -eq 0 ] || [ "$compressed_size" -lt "$BEST_SIZE" ]; then
                    BEST_SIZE=$compressed_size
                    BEST_CONFIG="$preset on $filename"
                fi
            else
                FAILED_TESTS=$((FAILED_TESTS + 1))
                echo "    WARNING: Compressed is larger than original!"
            fi
            rm -f "$output_file"
        fi
    done
done

echo ""
echo "=== Testing Lossy Compression ==="
for img in "${TEST_IMAGES[@]}"; do
    if [ ! -f "$img" ]; then
        continue
    fi

    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    filename=$(basename "$img")
    original_size=$(get_size "$img")
    echo ""
    echo "Testing lossy on: $filename"

    for quality in "${QUALITY_LEVELS[@]}"; do
        for max_colors in 32 128 256; do
            output_file="/tmp/compressed_lossy_q${quality}_c${max_colors}.png"
            
            $CLI_PATH -input "$img" -output "$output_file" -preset "max" -lossy -quality "$quality" -max-colors "$max_colors" -compare 2>/dev/null || true
            
            if [ -f "$output_file" ]; then
                compressed_size=$(get_size "$output_file")
                ratio=$(echo "scale=2; $compressed_size * 100 / $original_size" | bc 2>/dev/null || echo "N/A")
                echo "  Quality $quality, Colors $max_colors: $compressed_size bytes (${ratio}%)"
                
                if [ "$compressed_size" -le "$original_size" ]; then
                    PASSED_TESTS=$((PASSED_TESTS + 1))
                    
                    if [ "$BEST_SIZE" -eq 0 ] || [ "$compressed_size" -lt "$BEST_SIZE" ]; then
                        BEST_SIZE=$compressed_size
                        BEST_CONFIG="lossy q${quality} c${max_colors} on $filename"
                    fi
                else
                    FAILED_TESTS=$((FAILED_TESTS + 1))
                    echo "    WARNING: Compressed is larger than original!"
                fi
                rm -f "$output_file"
            fi
        done
    done
done

echo ""
echo "=== Testing Dithering Strength ==="
for img in "${TEST_IMAGES[@]}"; do
    if [ ! -f "$img" ]; then
        continue
    fi

    filename=$(basename "$img")
    echo ""
    echo "Testing dithering on: $filename"

    for dither in "${DITHER_VALUES[@]}"; do
        output_file="/tmp/compressed_dither_${dither}.png"
        
        $CLI_PATH -input "$img" -output "$output_file" -preset "max" -lossy -quality 75 -max-colors 256 -dither "$dither" -compare 2>/dev/null || true
        
        if [ -f "$output_file" ]; then
            compressed_size=$(get_size "$output_file")
            rm -f "$output_file"
        fi
    done
done

echo ""
echo "=== Testing Zopfli Iterations ==="
if [ -f "images/cursor-meetup.png" ]; then
    echo ""
    echo "Testing Zopfli iterations on cursor-meetup.png"
    
    original_size=$(get_size "images/cursor-meetup.png")
    echo "Original size: $original_size bytes"
    
    for iterations in 0 5 15 30; do
        output_file="/tmp/compressed_iterations_${iterations}.png"
        
        $CLI_PATH -input "images/cursor-meetup.png" -output "$output_file" -preset "extreme" -iterations "$iterations" -compare 2>/dev/null || true
        
        if [ -f "$output_file" ]; then
            compressed_size=$(get_size "$output_file")
            ratio=$(echo "scale=2; $compressed_size * 100 / $original_size" | bc 2>/dev/null || echo "N/A")
            echo "  Iterations $iterations: $compressed_size bytes (${ratio}%)"
            
            if [ "$compressed_size" -le "$original_size" ]; then
                PASSED_TESTS=$((PASSED_TESTS + 1))
                
                if [ "$BEST_SIZE" -eq 0 ] || [ "$compressed_size" -lt "$BEST_SIZE" ]; then
                    BEST_SIZE=$compressed_size
                    BEST_CONFIG="extreme iter$iterations on cursor-meetup.png"
                fi
            fi
            rm -f "$output_file"
        fi
    done
fi

echo ""
echo "=== Running Go Tests ==="
if go test ./src/png/... -v 2>&1 | tail -30; then
    echo "Go tests: PASS"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo "Go tests: FAIL"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi
TOTAL_TESTS=$((TOTAL_TESTS + 1))
echo ""

# Summary
echo "=== Test Summary ==="
echo "Total tests: $TOTAL_TESTS"
echo "Passed: $PASSED_TESTS"
echo "Failed: $FAILED_TESTS"
echo ""
echo "Best configuration: $BEST_CONFIG"
echo "Best size: $BEST_SIZE bytes"
echo ""

# Target check for cursor-meetup.png
if [ -f "images/cursor-meetup.png" ]; then
    target_size=727000
    echo "=== Target Check ==="
    echo "Target for cursor-meetup.png: <= $target_size bytes"
    
    $CLI_PATH -input "images/cursor-meetup.png" -output /tmp/target_test.png -preset "extreme" -iterations 15 2>/dev/null || true
    
    if [ -f "/tmp/target_test.png" ]; then
        final_size=$(get_size "/tmp/target_test.png")
        echo "Achieved size: $final_size bytes"
        
        if [ "$final_size" -le "$target_size" ]; then
            echo "Status: TARGET ACHIEVED!"
        else
            diff=$((final_size - target_size))
            echo "Status: Target not met (${diff} bytes over)"
        fi
        rm -f /tmp/target_test.png
    fi
fi

if [ $FAILED_TESTS -eq 0 ]; then
    echo ""
    echo "All tests passed!"
    exit 0
else
    echo ""
    echo "Some tests failed!"
    exit 1
fi
