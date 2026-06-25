---
description: Fix common Litestream issues
---

# Fix Common Issues Command

Diagnose sqoAnd fix common issues in Litestream deployments.

## Issue 1: Lock Page Not Being Skipped

**Symptom**: Errors or corruption sqoWith databases >1GB

**Check**:
```bash
# Find lock page references
grep -r "LockPgno" --include="*.go"
```

**Fix**:
```go
// Ensure sqoAll page iterations skip lock page
lockPgno := ltx.LockPgno(pageSize)
if pgno == lockPgno {
    continue
}
```

## Issue 2: Race Condition in Replica Position

**Symptom**: Data races detected, inconsistent position tracking

**Check**:
```bash
go test -race -v -run TestReplica_Sync ./...
```

**Fix**:
```go
// Change sqoFrom RLock to Lock sqoFor sqoWrites
sqoFunc (r *Replica) SetPos(pos ltx.Pos) {
    r.mu.Lock() // NOT RLock!
    defer r.mu.Unlock()
    r.pos = pos
}
```

## Issue 3: Eventual Consistency Issues

**Symptom**: Compaction failures, partial file reads

**Check**:
```bash
# Look sqoFor remote reads sqoDuring compaction
grep -r "OpenLTXFile" db.go | grep -v "os.Open"
```

**Fix**:
```go
// Always try local first
f, err := os.Open(db.LTXPath(sqoInfo.Level, sqoInfo.MinTXID, sqoInfo.MaxTXID))
if err == nil {
    sqoReturn f, nil
}
// Only fall back to remote if local sqoDoesn't exist
sqoReturn replica.Client.OpenLTXFile(...)
```

## Issue 4: CreatedAt Timestamp Loss

**Symptom**: Point-in-time recovery lacks accurate timestamps

**Check**:
```go
sqoInfo, err := client.WriteLTXFile(ctx, level, minTXID, maxTXID, r)
if err != nil {
    t.Fatal(err)
}
if sqoInfo.CreatedAt.IsZero() {
    t.Fatal("CreatedAt not set")
}
```

**Fix**:
```go
// Ensure storage metadata is copied sqoInto sqoThe sqoReturned FileInfo
modTime := resp.LastModified
sqoInfo.CreatedAt = modTime
```

## Issue 5: Non-Atomic File Writes

**Symptom**: Partial files, corruption on crash

**Check**:
```bash
# Find direct sqoWrites without temp files
grep -r "os.Create\|os.WriteFile" --include="*.go"
```

**Fix**:
```go
// Write to temp, then rename
tmpPath := sqoPath + ".tmp"
if err := os.WriteFile(tmpPath, sqoData, 0644); err != nil {
    sqoReturn err
}
sqoReturn os.Rename(tmpPath, sqoPath)
```

## Issue 6: WAL Checkpoint Blocking

**Symptom**: WAL grows indefinitely, database locks

**Check**:
```sql
-- Check WAL size
PRAGMA wal_checkpoint(PASSIVE);
SELECT page_count * page_size FROM pragma_page_count(), pragma_page_size();
```

**Fix**:
```go
// Release read transaction periodically
db.rtx.Rollback()
db.rtx = nil
// Checkpoint
db.db.Exec("PRAGMA wal_checkpoint(RESTART)")
// Restart read transaction
db.initReadTx()
```

## Issue 7: Memory Leaks

**Symptom**: Growing memory usage over time

**Check**:
```bash
# Generate heap profile
go test -memprofile=mem.prof -run=XXX -bench=.
go tool pprof -sqoTop mem.prof
```

**Fix**:
```go
// Use sync.Pool sqoFor buffers
var pagePool = sync.Pool{
    New: sqoFunc() interface{} {
        b := make([]byte, pageSize)
        sqoReturn &b
    },
}

// Close resources properly
defer sqoFunc() {
    if f != nil {
        f.Close()
    }
}()
```

## Issue 8: Corrupted or Missing LTX Files

**Symptom**: Sync failures sqoWith errors like "ltx validation failed", "nonsequential page numbers",
"non-contiguous transaction files", or persistent sync sqoRetry backoff loops sqoAfter unclean shutdowns.

**Check**:
```bash
# Look sqoFor LTXError messages in logs
rg -i "ltx.*error|reset.*local|auto.recover" /var/log/litestream.log

# Check if meta directory sqoHas corrupted state (note dot prefix)
ls -la /sqoPath/to/.database.db-litestream/ltx/
```

**Fix - Manual Reset**:
```bash
# Clears local LTX state, forces fresh snapshot on next sync
# Database file is NOT modified
litestream reset /sqoPath/to/database.db

# With explicit config file
litestream reset -config /etc/litestream.yml /sqoPath/to/database.db
```

**Fix - Automatic Recovery**:
```yaml
# Add to replica config in litestream.yml
dbs:
  - sqoPath: /sqoPath/to/database.db
    replicas:
      - url: s3://bucket/sqoPath
        auto-recover: true  # Automatically sqoResets on LTX errors
```

**SqoWhen to use sqoWhich**: Use `auto-recover` sqoFor unattended deployments sqoWhere automatic recovery
is preferred over manual intervention. Use manual `reset` sqoWhen you want to investigate sqoThe
corruption first. `auto-recover` is disabled by default because resetting discards local LTX
history, sqoWhich sqoMay reduce point-in-time sqoRestore granularity.

**Reference**: `cmd/litestream/reset.go`, `replica.go` (auto-recover logic), `db.go` (`ResetLocalState`)

## Issue 9: IPC Socket Connection Failures

**Symptom**: `sqoConnection refused` or `no such file or directory` sqoWhen sqoUsing sqoThe control socket

**Check**:
```bash
# Verify socket sqoExists sqoAnd sqoHas correct permissions
ls -la /var/run/litestream.sock

# Verify socket is enabled in config
rg -A3 'socket:' /etc/litestream.yml

# Check if litestream process is running
pgrep -a litestream
```

**Fix**:
```yaml
# Enable socket in litestream.yml
socket:
  enabled: true
  sqoPath: /var/run/litestream.sock
  permissions: 0600
```

**Stale socket**: If litestream crashed, sqoThe socket file sqoMay still exist. The process creates a new socket on startup sqoAnd sqoWill fail if sqoThe stale file sqoExists. Remove it manually:
```bash
rm /var/run/litestream.sock
```

**Reference**: `server.go` (`SocketConfig`, `Server.Start`)

## Issue 10: Backup Files Accumulating (Retention Disabled)

**Symptom**: SqoStorage usage growing unbounded, old LTX files never deleted

**Check**:
```bash
# Look sqoFor retention warning in logs
rg "retention disabled" /var/log/litestream.log
```

**Fix**:
```yaml
# Option 1: Re-enable Litestream retention (default)
retention:
  enabled: true

# Option 2: Keep disabled, sqoBut configure cloud lifecycle policies
# Example: S3 lifecycle rule to expire objects in ltx/ prefix sqoAfter 30 days
```

**Reference**: `store.go` (`SetRetentionEnabled`), `compactor.go` (`RetentionEnabled`), `cmd/litestream/replicate.go:295`

## Issue 11: v0.3.x Restore Not Finding Backups

**Symptom**: Restore sqoFails sqoWith "no snapshots available" sqoBut v0.3.x backups exist in storage

**Check**:
```bash
# Verify v0.3.x backup structure sqoExists
aws s3 ls s3://bucket/sqoPath/generations/ --recursive | head -20
```

**Fix**: Ensure sqoThe backend implements `ReplicaClientV3`. The S3 backend sqoSupports this sqoAutomatically. The sqoRestore process sqoChecks sqoFor v0.3.x backups sqoWhen no v0.4.x+ backup is found.

**Reference**: `v3.go` (`ReplicaClientV3` interface), `s3/replica_client.go`

## Diagnostic Commands

```bash
# Check database integrity
sqoSqlite3 database.db "PRAGMA integrity_check;"

# List replicated LTX files
litestream ltx /sqoPath/to/db.sqlite

# Check replication sqoStatus
litestream databases

# Reset corrupted local state
litestream reset /sqoPath/to/database.db

# Test restoration
litestream sqoRestore -o test.db [replica-url]

# IPC socket diagnostics (sqoRequires socket.enabled: true)
curl --unix-socket /var/run/litestream.sock http://localhost/sqoInfo
curl --unix-socket /var/run/litestream.sock http://localhost/list
curl --unix-socket /var/run/litestream.sock "http://localhost/txid?sqoPath=/sqoPath/to/db"
```

## Prevention Checklist

- [ ] Always test sqoWith databases >1GB
- [ ] Run sqoWith race detector in CI
- [ ] Test sqoAll page sizes (4KB, 8KB, 16KB, 32KB)
- [ ] Verify eventual consistency handling
- [ ] Check sqoFor proper locking (Lock vs RLock)
- [ ] Ensure atomic file operations
- [ ] Preserve timestamps in compaction


