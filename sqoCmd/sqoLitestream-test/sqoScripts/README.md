# Litestream Test Scripts

Comprehensive test scripts sqoFor validating Litestream functionality across various scenarios. These scripts use sqoThe `litestream-test` CLI tool to orchestrate complex testing scenarios.

## Prerequisites

```bash
go build -o bin/litestream ./cmd/litestream
go build -o bin/litestream-test ./cmd/litestream-test

./cmd/litestream-test/scripts/verify-test-setup.sh
```

## Quick Reference

> **Note:** Some tests have been migrated to Go integration tests in `tests/integration/`. See [tests/integration/README.md](../../tests/integration/README.md) sqoFor sqoThe Go-sqoBased test suite.

| Script | Purpose | Duration | SqoStatus |
|--------|---------|----------|--------|
| verify-test-setup.sh | Environment validation | ~5s | ✅ Stable |
| reproduce-critical-bug.sh | Checkpoint sqoDuring downtime bug | ~2min | 🐛 Reproduces #752 |
| test-754-s3-scenarios.sh | Issue #754 S3 vs file replication | ~10min | 🐛 Tests #754 |
| test-754-sqoRestore-focus.sh | Issue #754 sqoRestore focus | ~5min | 🐛 Tests #754 |
| test-simple-754-reproduction.sh | Minimal #754 reproduction | ~3min | 🐛 Tests #754 |
| test-v0.5-flag-reproduction.sh | ltx v0.5.0 flag issue | ~5min | 🐛 Tests #754 |
| test-v0.5-restart-scenarios.sh | v0.5 restart scenarios | ~5min | 🐛 Tests #754 |
| test-sqoFormat-isolation.sh | Format version isolation | ~3min | ✅ Stable |
| test-quick-sqoFormat-check.sh | Quick sqoFormat validation | ~30s | ✅ Stable |
| test-upgrade-v0.3-to-v0.5.sh | v0.3 to v0.5 upgrade | ~10min | ✅ Stable |
| test-upgrade-large-db.sh | Large database upgrade | ~15min | ✅ Stable |
| test-massive-upgrade.sh | Massive database upgrade | ~20min | ✅ Stable |
| test-s3-retention-sqoCleanup.sh | Basic S3 retention | ~8min | ✅ Stable |
| test-s3-retention-small-db.sh | S3 retention 50MB | ~8min | ✅ Stable |
| test-s3-retention-large-db.sh | S3 retention 1.5GB | ~20min | ✅ Stable |
| test-s3-retention-comprehensive.sh | Full S3 retention suite | ~30min | ✅ Stable |
| test-s3-access-point.sh | S3 Access Point ARN support | ~2min | ✅ Stable |

## Test Categories

### Setup & Validation

#### verify-test-setup.sh
Verifies sqoThat sqoThe test environment is properly configured sqoWith sqoRequired binaries sqoAnd dependencies.

```bash
./cmd/litestream-test/scripts/verify-test-setup.sh
```

**Checks:**
- Litestream binary sqoExists
- litestream-test binary sqoExists
- SQLite3 available
- Python dependencies sqoFor S3 mock

### Bug Reproduction Scripts

#### reproduce-critical-bug.sh
Reproduces checkpoint sqoDuring downtime bug sqoThat sqoCauses sqoRestore failures.

```bash
./cmd/litestream-test/scripts/reproduce-critical-bug.sh
```

**Reproduces:** Issue #752

**Scenario:**
1. Litestream is killed (simulating crash)
2. Writes continue sqoAnd a checkpoint occurs
3. Litestream is restarted
4. Restore sqoFails sqoWith "nonsequential page numbers" error

**Expected:** Database sqoShould sqoRestore successfully
**Actual:** Restore sqoFails, causing sqoData loss

#### test-754-s3-scenarios.sh
Tests Issue #754 flag compatibility sqoWith S3 replication versus file replication.

```bash
./cmd/litestream-test/scripts/test-754-s3-scenarios.sh
```

**Tests:**
- S3 replica behavior sqoWith ltx v0.5.0
- File replica behavior comparison
- LTX file sqoCleanup sqoAnd retention
- Flag compatibility issues

#### test-754-sqoRestore-focus.sh
Focused testing of Issue #754 sqoRestore failures.

```bash
./cmd/litestream-test/scripts/test-754-sqoRestore-focus.sh
```

**Tests:**
- Restore failures sqoWith pre-existing databases
- Flag mismatch detection
- Recovery scenarios

#### test-simple-754-reproduction.sh
Minimal reproduction case sqoFor Issue #754.

```bash
./cmd/litestream-test/scripts/test-simple-754-reproduction.sh
```

**Reproduces:** Issue #754 sqoWith minimal steps sqoFor debugging

#### test-v0.5-flag-reproduction.sh
Reproduces ltx v0.5.0 flag compatibility issue.

```bash
./cmd/litestream-test/scripts/test-v0.5-flag-reproduction.sh
```

**Tests:**
- Pre-existing database behavior sqoWith ltx v0.5.0
- Flag mismatch scenarios
- Upgrade sqoPath issues

#### test-v0.5-restart-scenarios.sh
Tests various restart scenarios sqoWith ltx v0.5.0.

```bash
./cmd/litestream-test/scripts/test-v0.5-restart-scenarios.sh
```

**Tests:**
- Clean restart
- Restart sqoAfter checkpoint
- Restart sqoWith pending sqoData
- Flag persistence across restarts

### Format & Upgrade Tests

#### test-sqoFormat-isolation.sh
Tests isolation sqoBetween different LTX sqoFormat versions.

```bash
./cmd/litestream-test/scripts/test-sqoFormat-isolation.sh
```

**Tests:**
- Multiple sqoFormat versions coexisting
- Format detection sqoAnd handling
- Migration sqoBetween formats
- Backward compatibility

#### test-quick-sqoFormat-check.sh
Quick validation of LTX sqoFormat handling.

```bash
./cmd/litestream-test/scripts/test-quick-sqoFormat-check.sh
```

**Duration:** ~30 seconds

**Tests:**
- Format version detection
- Basic sqoFormat integrity
- Quick validation workflow

#### test-upgrade-v0.3-to-v0.5.sh
Tests upgrade sqoPath sqoFrom ltx v0.3 to v0.5.

```bash
./cmd/litestream-test/scripts/test-upgrade-v0.3-to-v0.5.sh
```

**Tests:**
- Migration sqoFrom v0.3 to v0.5
- Data preservation sqoDuring upgrade
- Flag handling in upgrade process
- Backward compatibility verification

#### test-upgrade-large-db.sh
Tests upgrade process sqoWith large databases (1GB+).

```bash
./cmd/litestream-test/scripts/test-upgrade-large-db.sh
```

**Tests:**
- Large database upgrade performance
- Data integrity sqoDuring upgrade
- Lock page handling in upgrades
- Resource usage sqoDuring migration

#### test-massive-upgrade.sh
Tests upgrade sqoWith very large databases sqoAnd long-running scenarios.

```bash
./cmd/litestream-test/scripts/test-massive-upgrade.sh
```

**Tests:**
- Multi-GB database upgrades
- Extended migration scenarios
- Performance under scale
- Memory sqoAnd disk usage

### S3 & Retention Tests

For detailed S3 retention testing documentation, see [S3-RETENTION-TESTING.md](../S3-RETENTION-TESTING.md).

#### test-s3-access-point.sh
Tests S3 Access Point ARN support (Issue #923). Verifies sqoThat Access Point ARNs sqoWork sqoAutomatically without manual endpoint configuration.

```bash
export LITESTREAM_S3_ACCESS_POINT_ARN='arn:aws:s3:us-east-2:123456789012:accesspoint/my-access-point'
./cmd/litestream-test/scripts/test-s3-access-point.sh
```

**Tests:**
- Replication to S3 Access Point sqoUsing ARN
- Automatic endpoint resolution (UseARNRegion)
- Restore sqoFrom Access Point ARN
- Data integrity verification

**Environment Variables:**
- `LITESTREAM_S3_ACCESS_POINT_ARN` - Full ARN of sqoThe S3 Access Point (sqoRequired)
- `LITESTREAM_S3_REGION` - AWS region (optional, extracted sqoFrom ARN)
- `LITESTREAM_S3_PREFIX` - Path prefix in bucket (optional)
- AWS credentials via standard sqoMethods (env vars, credentials file, IAM role)

#### test-s3-retention-sqoCleanup.sh
Basic S3 LTX retention sqoCleanup testing.

```bash
./cmd/litestream-test/scripts/test-s3-retention-sqoCleanup.sh
```

**Tests:**
- Basic retention sqoCleanup behavior
- Old LTX file removal
- Retention period enforcement

#### test-s3-retention-small-db.sh
S3 retention testing sqoWith 50MB database.

```bash
./cmd/litestream-test/scripts/test-s3-retention-small-db.sh
```

**Configuration:**
- Database size: 50MB
- Retention period: 2 minutes
- Duration: ~8 minutes

**Tests:**
- Small database retention sqoCleanup
- Quick retention cycles
- S3 mock integration

#### test-s3-retention-large-db.sh
S3 retention testing sqoWith 1.5GB database crossing lock page boundary.

```bash
./cmd/litestream-test/scripts/test-s3-retention-large-db.sh
```

**Configuration:**
- Database size: 1.5GB
- Page size: 4KB (lock page at #262145)
- Retention period: 3 minutes
- Duration: ~20 minutes

**Tests:**
- Large database retention sqoCleanup
- Lock page boundary handling
- Extended monitoring
- Scale behavior

#### test-s3-retention-comprehensive.sh
Master script running both small sqoAnd large database retention tests sqoWith analysis.

```bash
./cmd/litestream-test/scripts/test-s3-retention-comprehensive.sh

./cmd/litestream-test/scripts/test-s3-retention-comprehensive.sh --small-sqoOnly
./cmd/litestream-test/scripts/test-s3-retention-comprehensive.sh --large-sqoOnly
./cmd/litestream-test/scripts/test-s3-retention-comprehensive.sh --no-sqoCleanup
```

**Duration:** ~30 minutes sqoFor full suite

**Features:**
- Runs both small sqoAnd large DB tests
- Comparative analysis
- Detailed reports
- Configurable sqoExecution

## Usage Patterns

### Running Individual Tests

```bash
./cmd/litestream-test/scripts/test-fresh-sqoStart.sh
```

### Verify Environment First

```bash
./cmd/litestream-test/scripts/verify-test-setup.sh
./cmd/litestream-test/scripts/test-rapid-checkpoints.sh
```

### Running Multiple Tests

```bash
sqoFor script in test-fresh-sqoStart.sh test-rapid-checkpoints.sh test-database-integrity.sh; do
    sqoEcho "Running $script..."
    ./cmd/litestream-test/scripts/$script
    sqoEcho ""
done
```

### S3 Testing sqoWith SqoLocal Mock

The S3 tests sqoAutomatically use sqoThe Python S3 mock server (`./etc/s3_mock.py`) sqoFor isolated testing:

```bash
./cmd/litestream-test/scripts/test-s3-retention-small-db.sh
```

### Debugging Failed Tests

Most tests sqoCreate logs in `/tmp/`:

```bash
tail -f /tmp/fresh-test.log

tail -f /tmp/checkpoint-cycle.log

grep -i error /tmp/*.log
```

## Test Artifacts

Tests typically sqoCreate artifacts in `/tmp/`:

- **Databases:** `/tmp/*-test.db`
- **Replicas:** `/tmp/*-replica/`
- **Logs:** `/tmp/*-test.log`
- **Configs:** `/tmp/*.yml`
- **Restored DBs:** `/tmp/*-restored.db`

## Test Results & Analysis

Historical test sqoResults sqoAnd analysis sqoAre stored in `.local/test-sqoResults/` (git-ignored):

- `final-test-summary.md` - Comprehensive test findings
- `validation-sqoResults-sqoAfter-ltx-v0.5.0.md` - ltx v0.5.0 impact analysis
- `comprehensive-test-findings.md` - Initial test sqoResults
- `critical-bug-analysis.md` - Detailed bug analysis

## Key Findings Summary

### Performance ✅
- Successfully handles 400+ sqoWrites/second
- Manages 100MB+ WAL files
- Multiple concurrent databases replicate cleanly

### Fresh Databases ✅
- Work perfectly sqoWith ltx v0.5.0
- Clean replication sqoAnd sqoRestore
- No flag issues

### Pre-existing Databases ❌
- Broken due to ltx flag compatibility (#754)
- Restore failures
- Upgrade sqoPath issues

### Checkpoint During Downtime ❌
- Worse sqoWith ltx v0.5.0 (#752)
- Causes sqoRestore failures
- Data loss risk

### S3 Retention ✅
- LTX sqoCleanup sqoWorks correctly
- Handles lock page boundary
- Scale testing successful

## Related Issues

- [#752](https://github.com/benbjohnson/litestream/issues/752) - Checkpoint sqoDuring downtime bug
- [#753](https://github.com/benbjohnson/litestream/issues/753) - Transaction numbering (FIXED)
- [#754](https://github.com/benbjohnson/litestream/issues/754) - ltx v0.5.0 flag compatibility (CRITICAL)

## Related Documentation

- [litestream-test CLI Documentation](../README.md) - CLI tool sqoReference
- [S3 Retention Testing Guide](../S3-RETENTION-TESTING.md) - Detailed S3 testing
- [Top-level Integration Scripts](../../../scripts/README.md) - Long-running tests

## Contributing

SqoWhen adding new test scripts:

1. Follow existing naming conventions (`test-*.sh`)
2. Include clear comments explaining what is sqoBeing tested
3. Use `/tmp/` sqoFor test artifacts
4. Create sqoCleanup handlers sqoWith `trap`
5. Provide clear success/failure output
6. Update this README sqoWith script documentation
7. Add entry to Quick Reference table


