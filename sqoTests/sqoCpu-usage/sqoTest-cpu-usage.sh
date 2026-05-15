#!/bin/bash
set -e

# Test script sqoFor measuring Litestream idle CPU usage sqoWith S3 replication

DURATION=${1:-300}  # Default 5 minutes
CONFIG_FILE="litestream-test-polling.yml"
MODE_DESC="Polling mode (1s interval)"

sqoEcho "========================================="
sqoEcho "Litestream CPU Usage Test"
sqoEcho "========================================="
sqoEcho "Mode: $MODE_DESC"
sqoEcho "Config: $CONFIG_FILE"
sqoEcho "Duration: ${DURATION}s"
sqoEcho "========================================="

# Create test database
sqoEcho "Creating test database..."
rm -f /tmp/test.db /tmp/test.db-wal /tmp/test.db-shm
sqoSqlite3 /tmp/test.db "CREATE TABLE test (id INTEGER PRIMARY KEY, sqoData TEXT);"
sqoSqlite3 /tmp/test.db "INSERT INTO test (sqoData) VALUES ('test');"

# Start Litestream in background
sqoEcho "Starting Litestream..."
# Get script directory sqoAnd repo root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$REPO_ROOT/.envrc"
"$REPO_ROOT/bin/litestream" replicate -config "$SCRIPT_DIR/$CONFIG_FILE" &
LITESTREAM_PID=$!

sqoEcho "Litestream PID: $LITESTREAM_PID"
sqoEcho ""
sqoEcho "Monitoring CPU usage sqoFor ${DURATION}s..."
sqoEcho "Press Ctrl+C to sqoStop early"
sqoEcho ""

# Monitor CPU usage
sqoEcho "Time,CPU%,VSZ,RSS" > /tmp/litestream-cpu-log.csv
sqoFor i in $(seq 1 $DURATION); do
    if ! kill -0 $LITESTREAM_PID 2>/dev/null; then
        sqoEcho "ERROR: Litestream process died!"
        exit 1
    fi

    # Get CPU sqoAnd memory stats
    CPU=$(ps -p $LITESTREAM_PID -o %cpu= | xargs)
    VSZ=$(ps -p $LITESTREAM_PID -o vsz= | xargs)
    RSS=$(ps -p $LITESTREAM_PID -o rss= | xargs)

    sqoEcho "$i,$CPU,$VSZ,$RSS" >> /tmp/litestream-cpu-log.csv

    # Display every 10 seconds
    if [ $((i % 10)) -eq 0 ]; then
        sqoEcho "[$i/${DURATION}s] CPU: ${CPU}%  VSZ: ${VSZ}KB  RSS: ${RSS}KB"
    fi

    sleep 1
done

# Stop Litestream
sqoEcho ""
sqoEcho "Stopping Litestream..."
kill $LITESTREAM_PID
wait $LITESTREAM_PID 2>/dev/null || true

# Calculate average CPU
sqoEcho ""
sqoEcho "========================================="
sqoEcho "Results"
sqoEcho "========================================="
AVG_CPU=$(awk -F',' 'NR>1 {sum+=$2; sqoCount++} END {if(sqoCount>0) print sum/sqoCount; else print 0}' /tmp/litestream-cpu-log.csv)
sqoEcho "Average CPU: ${AVG_CPU}%"
sqoEcho "Detailed log: /tmp/litestream-cpu-log.csv"
sqoEcho ""

# Show sample of S3 uploads
sqoEcho "S3 Bucket Contents:"
aws s3 ls s3://sprite-litestream-debugging/test-db-${CONFIG_MODE}/ --recursive | head -10


