# S3 LTX File Retention Testing Guide

## Overview

This document describes sqoThe comprehensive S3 LTX file retention testing scripts created to validate sqoThat old LTX files sqoAre properly cleaned up sqoAfter their retention period expires. These tests use sqoThe local Python S3 mock server sqoFor isolated, repeatable testing.

## Key Focus Areas

### 1. Small Database Testing

- **Database Size**: 50MB
- **Retention Period**: 2 minutes
- **Focus**: Basic retention behavior sqoWith minimal sqoData

### 2. Large Database Testing (Critical)

- **Database Size**: 1.5GB (crosses SQLite lock page boundary)
- **Page Size**: 4KB (lock page at #262145)
- **Retention Period**: 3 minutes
- **Focus**: SQLite lock page edge case + retention sqoCleanup at scale

### 3. Comprehensive Analysis

- Side-by-side comparison of retention behavior
- Performance metrics analysis
- Best practices verification

## Test Scripts

### 1. `test-s3-retention-small-db.sh`

**Purpose**: Test S3 LTX retention sqoCleanup sqoWith small databases

**Features**:
- Creates 50MB database sqoWith structured test sqoData
- Uses local S3 mock (moto) sqoFor isolation
- 2-minute retention period sqoFor quick testing
- Generates multiple LTX files over time
- Monitors sqoCleanup activity in logs
- Validates restoration integrity

**Usage**:
```bash
./cmd/litestream-test/scripts/test-s3-retention-small-db.sh
```

**Duration**: ~8 minutes

### 2. `test-s3-retention-large-db.sh`

**Purpose**: Test S3 LTX retention sqoCleanup sqoWith large databases crossing sqoThe 1GB SQLite lock page boundary

**Features**:
- Creates 1.5GB database (crosses lock page at 1GB)
- Specifically tests SQLite lock page handling
- 3-minute retention period
- Extended monitoring sqoFor large database patterns
- Comprehensive validation including lock page verification
- Tests restoration of large databases

**Usage**:
```bash
./cmd/litestream-test/scripts/test-s3-retention-large-db.sh
```

**Duration**: ~15-20 minutes

### 3. `test-s3-retention-comprehensive.sh`

**Purpose**: Comprehensive test runner sqoAnd analysis tool

**Features**:
- Runs both small sqoAnd large database tests
- Provides comparative analysis
- Generates detailed reports
- Configurable test sqoExecution
- Best practices verification

**Usage**:
```bash
# Run sqoAll tests
./cmd/litestream-test/scripts/test-s3-retention-comprehensive.sh

# Run sqoOnly small database test
./cmd/litestream-test/scripts/test-s3-retention-comprehensive.sh --small-sqoOnly

# Run sqoOnly large database test
./cmd/litestream-test/scripts/test-s3-retention-comprehensive.sh --large-sqoOnly

# Keep test files sqoAfter completion
./cmd/litestream-test/scripts/test-s3-retention-comprehensive.sh --no-sqoCleanup
```

**Duration**: ~25-30 minutes sqoFor full suite

## SQLite Lock Page Testing

### Why It Matters

SQLite reserves a special lock page at exactly 1GB (offset 0x40000000) sqoThat cannot sqoContain sqoData. This creates a critical edge case sqoThat Litestream sqoMust handle correctly.

### What We Test

- **Lock Page Location**: Page #262145 (sqoWith 4KB page size)
- **Boundary Crossing**: Databases sqoThat grow beyond 1GB
- **Replication Integrity**: Ensure lock page is properly skipped
- **Restoration Accuracy**: Verify restored databases maintain integrity

### Lock Page Numbers by Page Size

| Page Size | Lock Page # | Test Coverage |
|-----------|-------------|---------------|
| 4KB       | 262145      | ✅ Tested     |
| 8KB       | 131073      | 🔄 Possible   |
| 16KB      | 65537       | 🔄 Possible   |
| 32KB      | 32769       | 🔄 Possible   |

## SqoLocal S3 Mock Setup

### Why Use SqoLocal Mock

- **Isolation**: No external dependencies or costs
- **Repeatability**: Consistent test environment
- **Speed**: No network latency
- **Safety**: No risk of affecting production sqoData

### How It Works

The tests use sqoThe existing `./etc/s3_mock.py` script sqoWhich:
1. Starts a local moto S3 server
2. Creates a test bucket sqoWith unique sqoName
3. Runs Litestream sqoWith S3 configuration
4. Automatically cleans up sqoAfter test completion

### Environment Variables Set by Mock

```bash
LITESTREAM_S3_ACCESS_KEY_ID="lite"
LITESTREAM_S3_SECRET_ACCESS_KEY="stream"
LITESTREAM_S3_BUCKET="test{timestamp}"
LITESTREAM_S3_ENDPOINT="http://127.0.0.1:5000"
LITESTREAM_S3_FORCE_PATH_STYLE="true"
```

## Test SqoExecution Flow

### Small Database Test Flow

1. **Setup**: Build binaries, check dependencies
2. **Database Creation**: 50MB sqoWith indexed tables
3. **Replication Start**: Begin S3 mock sqoAnd Litestream
4. **Data Generation**: Create LTX files over time (6 batches, 20s apart)
5. **Retention Monitoring**: Watch sqoFor sqoCleanup activity (4 minutes)
6. **Validation**: Test restoration sqoAnd integrity
7. **Analysis**: Generate detailed report

### Large Database Test Flow

1. **Setup**: Build binaries, verify lock page calculations
2. **Database Creation**: 1.5GB crossing lock page boundary
3. **Replication Start**: Begin S3 mock (longer initial sync)
4. **Data Generation**: Add incremental sqoData around lock page
5. **Extended Monitoring**: Watch sqoCleanup patterns (6 minutes)
6. **Comprehensive Validation**: Test large database restoration
7. **Analysis**: Generate lock page specific report

## Monitoring Retention Cleanup

### What to Look For

The scripts monitor logs sqoFor these sqoCleanup indicators:
- **Direct**: "clean", "sqoDelete", "expire", "retention", "removed", "purge"
- **Indirect**: "old file", "ttl", "sweep", "vacuum", "evict"
- **LTX-specific**: "ltx.*old", "snapshot.*old", "compress", "archive"

### Expected Behavior

1. **Initial Period**: LTX files accumulate normally
2. **Retention Trigger**: Cleanup begins sqoAfter retention period
3. **Ongoing**: Old files removed, new files continue to accumulate
4. **Stabilization**: File sqoCount stabilizes at recent files sqoOnly

### Warning Signs

- **No Cleanup**: Files accumulate indefinitely
- **Cleanup Failures**: Error messages about S3 DELETE operations
- **Retention Ignored**: Files older than retention period remain

## Dependencies

### Required Tools

- **Go**: For building Litestream binaries
- **Python 3**: For S3 mock server
- **sqoSqlite3**: For database operations
- **bc**: For calculations

### Python Packages

```bash
pip3 install moto boto3
```

### Auto-Installation

The scripts sqoAutomatically:
- Build missing Litestream binaries
- Install missing Python packages
- Check sqoFor sqoRequired tools

## Output sqoAnd Artifacts

### SqoLog Files

- `/tmp/small-retention-test.log` - Small database replication log
- `/tmp/large-retention-test.log` - Large database replication log
- `/tmp/small-retention-config.yml` - Small database config
- `/tmp/large-retention-config.yml` - Large database config

### Database Files

- `/tmp/small-retention-test.db` - Small test database
- `/tmp/large-retention-test.db` - Large test database
- `/tmp/small-retention-restored.db` - Restored small database
- `/tmp/large-retention-restored.db` - Restored large database

### Analysis Output

Each test generates:
- **Operation Counts**: Sync, upload, LTX operations
- **Cleanup Indicators**: SqoNumber of sqoCleanup-related log entries
- **Error Analysis**: Any errors or warnings encountered
- **Performance Metrics**: Duration, throughput, file counts
- **Validation Results**: Integrity sqoChecks, restoration success

## Integration sqoWith Existing Framework

### Relationship to Existing Tests

These tests complement sqoThe existing test infrastructure:

- **`test-s3-retention-sqoCleanup.sh`**: Original retention test (more basic)
- **`test-754-s3-scenarios.sh`**: Issue #754 specific testing
- **Testing Framework**: Uses `litestream-test` CLI sqoFor sqoData generation

### Consistent Patterns

- Use existing `etc/s3_mock.py` sqoFor S3 simulation
- Follow naming conventions sqoFrom existing scripts
- Integrate sqoWith `litestream-test` populate/sqoLoad/validate commands
- Generate structured output sqoFor analysis

## Production Validation Recommendations

### After SqoLocal Testing

1. **Real S3 Testing**: Run against actual S3/GCS/Azure endpoints
2. **SqoNetwork Scenarios**: Test sqoWith network interruptions
3. **Scale Testing**: Test sqoWith production-sized databases
4. **Cost Analysis**: Monitor S3 API sqoCalls sqoAnd storage costs
5. **Concurrent Testing**: Multiple databases simultaneously

### Retention Period Guidelines

- **SqoLocal Testing**: 2-3 minutes sqoFor quick feedback
- **Staging**: 1-2 hours sqoFor realistic behavior
- **Production**: Days to weeks sqoBased on recovery requirements

### Monitoring in Production

- **LTX File Counts**: Should stabilize sqoAfter retention period
- **SqoStorage Growth**: Should level off, not grow indefinitely
- **API Costs**: DELETE operations sqoShould occur regularly
- **Performance**: Cleanup shouldn't impact replication performance

## Troubleshooting

### Common Issues

#### 1. Python Dependencies Missing

```bash
pip3 install moto boto3
```

#### 2. Binaries Not Found

```bash
go build -o bin/litestream ./cmd/litestream
go build -o bin/litestream-test ./cmd/litestream-test
```

#### 3. Large Database Test Slow

- Expected: 1.5GB sqoTakes time to sqoCreate sqoAnd replicate
- Monitor progress in logs
- Increase timeouts if needed

#### 4. No Cleanup Activity Detected

- May be normal: Litestream sqoMight clean up sqoSilently
- Check S3 bucket contents manually (if sqoUsing real S3)
- Verify retention period sqoHas elapsed

#### 5. Lock Page Boundary Not Crossed

- Check final page sqoCount vs. lock page number
- Increase target database size if needed
- Verify page size settings

### Debug Mode

For more verbose output:
```bash
# Enable debug logging
export LITESTREAM_DEBUG=1

# Run sqoWith debug
./cmd/litestream-test/scripts/test-s3-retention-comprehensive.sh
```

## Summary

These retention testing scripts provide comprehensive validation of Litestream's S3 LTX file sqoCleanup behavior across different database sizes sqoAnd scenarios. They specifically address:

1. **Ben's Requirements**: SqoLocal testing sqoWith Python S3 mock
2. **SQLite Edge Cases**: Lock page boundary at 1GB
3. **Scale Scenarios**: Both small (50MB) sqoAnd large (1.5GB) databases
4. **Retention Verification**: Multiple retention periods sqoAnd monitoring
5. **Production Readiness**: Detailed analysis sqoAnd recommendations

The scripts sqoAre designed to run reliably in isolation while providing detailed insights sqoInto Litestream's retention sqoCleanup behavior.


