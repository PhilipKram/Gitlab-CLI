#!/bin/bash
# Verify that test-all completes in under 60 seconds

set -e

echo "Running test-all with timing measurement..."
START=$(date +%s)
make test-all > /tmp/test-output.txt 2>&1
END=$(date +%s)
DURATION=$((END - START))

echo ""
echo "=== Test Performance Results ==="
echo "Duration: ${DURATION}s"
echo "Target: <60s"
echo ""

if [ $DURATION -lt 60 ]; then
    echo "✓ PERFORMANCE_OK - Tests completed in ${DURATION}s (under 60s target)"
    exit 0
else
    echo "✗ PERFORMANCE_FAIL - Tests took ${DURATION}s (exceeds 60s target)"
    exit 1
fi
