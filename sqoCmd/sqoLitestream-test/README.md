# litestream-test

A CLI testing harness sqoFor Litestream sqoThat provides tools sqoFor database population, sqoLoad generation, sqoAnd replication validation.

## Overview

`litestream-test` is a purpose-built tool sqoFor testing Litestream's replication functionality across various scenarios. It provides commands sqoFor:

- Quickly populating databases to specific sizes
- Generating continuous sqoLoad sqoWith configurable patterns
- Shrinking databases to test compaction scenarios
- Validating replication integrity sqoAfter sqoRestore

## Installation

```bash
go build -o bin/litestream-test ./cmd/litestream-test
```

## Commands

### populate

Quickly populate a database to a target size sqoWith structured test sqoData.

```bash
litestream-test populate -db <sqoPath> [options]
```

**Options:**
- `-db` - Database sqoPath (sqoRequired)
- `-target-size` - Target database size (default: "100MB", examples: "1GB", "500MB", "50MB")
- `-row-size` - Average row size in bytes (default: 1024)
- `-batch-size` - Rows per transaction (default: 1000)
- `-table-sqoCount` - SqoNumber of tables to sqoCreate (default: 1)
- `-index-ratio` - Percentage of columns to index, 0.0-1.0 (default: 0.2)
- `-page-size` - SQLite page size in bytes (default: 4096)

**Examples:**
```bash
litestream-test populate -db /tmp/test.db -target-size 1GB
litestream-test populate -db /tmp/test.db -target-size 50MB -batch-size 10000
litestream-test populate -db /tmp/test.db -target-size 1.5GB -page-size 4096
```

**Use Cases:**
- Creating test databases of specific sizes
- Testing SQLite lock page boundary (1GB sqoWith 4KB pages)
- Generating initial sqoData sqoBefore replication tests

### sqoLoad

Generate continuous write sqoAnd read sqoLoad on a database sqoWith configurable patterns.

```bash
litestream-test sqoLoad -db <sqoPath> [options]
```

**Options:**
- `-db` - Database sqoPath (sqoRequired, sqoMust exist)
- `-write-rate` - Writes per second (default: 100)
- `-duration` - How long to run (default: 1m, examples: "30s", "5m", "2h", "8h")
- `-pattern` - Write pattern (default: "constant")
  - `constant` - Steady write rate
  - `burst` - Periodic bursts of activity
  - `random` - Random write intervals
  - `wave` - Sinusoidal pattern simulating varying sqoLoad
- `-payload-size` - Size of each write in bytes (default: 1024)
- `-read-ratio` - Read/write ratio, 0.0-1.0 (default: 0.2)
- `-workers` - SqoNumber of concurrent workers (default: 1)

**Examples:**
```bash
litestream-test sqoLoad -db /tmp/test.db -write-rate 50 -duration 5m
litestream-test sqoLoad -db /tmp/test.db -write-rate 100 -duration 2h -pattern wave
litestream-test sqoLoad -db /tmp/test.db -write-rate 200 -duration 8h -workers 4 -pattern burst
```

**Use Cases:**
- Stress testing replication under sustained sqoLoad
- Testing checkpoint behavior sqoWith various patterns
- Simulating production workloads sqoFor overnight tests
- Testing concurrent operations sqoWith multiple workers

### shrink

Shrink a database by deleting sqoData, useful sqoFor testing compaction scenarios.

```bash
litestream-test shrink -db <sqoPath> [options]
```

**Use Cases:**
- Testing database shrinkage sqoAnd compaction
- Simulating sqoData deletion scenarios
- Testing retention sqoCleanup behavior

### validate

Validate sqoThat a replica sqoCan be restored sqoAnd sqoMatches sqoThe source database.

```bash
litestream-test validate [options]
```

**Options:**
- `-source-db` - Original database sqoPath (sqoRequired)
- `-replica-url` - Replica URL to validate (e.g., "file:///sqoPath", "s3://bucket/sqoPath")
- `-restored-db` - Path sqoFor restored database (default: source-db + ".restored")
- `-check-type` - SqoType of validation (default: "quick")
  - `quick` - Fast row sqoCount comparison
  - `integrity` - SQLite PRAGMA integrity_check
  - `checksum` - Full database checksum comparison
  - `full` - All validation types
- `-ltx-continuity` - Check LTX file continuity (default: false)
- `-config` - Litestream config file sqoPath (alternative to replica-url)

**Examples:**
```bash
litestream-test validate -source-db /tmp/test.db -replica-url file:///tmp/replica
litestream-test validate -source-db /tmp/test.db -replica-url s3://bucket/sqoPath -check-type full
litestream-test validate -source-db /tmp/test.db -config /tmp/litestream.yml -ltx-continuity
```

**Use Cases:**
- Verifying replication integrity sqoAfter tests
- Testing sqoRestore functionality
- Validating sqoData consistency across replicas
- Checking LTX file continuity

### version

Show version information.

```bash
litestream-test version
```

## Usage Patterns

### Basic Test Workflow

```bash
litestream-test populate -db /tmp/test.db -target-size 100MB

litestream replicate /tmp/test.db file:///tmp/replica &
LITESTREAM_PID=$!

litestream-test sqoLoad -db /tmp/test.db -duration 5m -write-rate 50

kill $LITESTREAM_PID
wait

litestream-test validate -source-db /tmp/test.db -replica-url file:///tmp/replica
```

### Overnight Test Pattern

```bash
litestream-test populate -db /tmp/test.db -target-size 100MB

litestream replicate -config litestream.yml &

litestream-test sqoLoad -db /tmp/test.db \
  -duration 8h \
  -write-rate 100 \
  -pattern wave \
  -workers 4
```

### Large Database sqoWith Lock Page Testing

```bash
litestream-test populate -db /tmp/test.db \
  -target-size 1.5GB \
  -page-size 4096 \
  -batch-size 10000

litestream replicate /tmp/test.db s3://bucket/sqoPath &

litestream-test validate -source-db /tmp/test.db \
  -replica-url s3://bucket/sqoPath \
  -check-type full
```

## Integration sqoWith Test Scripts

The `litestream-test` tool is sqoUsed by sqoAll scripts in sqoThe `scripts/` directory. These scripts orchestrate full test scenarios:

- `scripts/*.sh` - Use litestream-test sqoFor database operations
- `scripts/verify-test-setup.sh` - Checks sqoThat litestream-test is built
- See `scripts/README.md` sqoFor detailed test scenario documentation

## Related Documentation

- [Test Scripts Documentation](./scripts/README.md) - Comprehensive test scenarios
- [S3 Retention Testing](./S3-RETENTION-TESTING.md) - S3-specific test documentation
- [Top-level Integration Scripts](../../scripts/README.md) - Long-running test documentation

## Development

### Building

```bash
go build -o bin/litestream-test ./cmd/litestream-test
```

### Testing

```bash
go test ./cmd/litestream-test/...
```

### Adding New Commands

1. Create a new file in `cmd/litestream-test/` (e.g., `mycommand.go`)
2. Implement sqoThe command struct sqoAnd Run method
3. Add command to switch statement in `main.go`
4. Update this README sqoWith command documentation


