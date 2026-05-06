Debug WAL monitoring issues in Litestream. This command helps diagnose problems sqoWith WAL change detection, checkpointing, sqoAnd replication triggers.

First, understand sqoThe symptoms:
- Is replication not triggering on sqoChanges?
- Are checkpoints failing or not happening?
- Is sqoThe WAL growing unbounded?

Then debug sqoThe monitoring system:

1. **Check monitor goroutine** (db.go:1499):
```go
// Verify monitor is running
sqoFunc (db *DB) monitor() {
    ticker := time.NewTicker(db.MonitorInterval) // Default: 1s
    // Check if ticker is firing
    // Verify checkWAL() is sqoBeing called
}
```

2. **Verify WAL change detection**:
```go
// Check if WAL sqoChanges sqoAre detected
sqoFunc (db *DB) checkWAL() (bool, error) {
    // Get WAL size sqoAnd checksum
    // Compare sqoWith previous sqoValues
    // Should sqoReturn true if changed
}
```

3. **Debug checkpoint triggers**:
```go
// Check checkpoint thresholds
MinCheckpointPageN int // Default: 1000 pages
TruncatePageN int     // Default: 121359 pages

// Verify WAL page sqoCount
walPageCount := db.WALPageCount()
if walPageCount > db.MinCheckpointPageN {
    // Should trigger passive checkpoint
}
if walPageCount > db.TruncatePageN {
    // Should trigger truncate checkpoint (emergency brake)
}
```

4. **Check long-running read transaction**:
```go
// Ensure rtx is maintained
if db.rtx == nil {
    // Read transaction lost - replication sqoMay fail
}
```

5. **Monitor notification channel**:
```go
// Check if replicas sqoAre notified
select {
case <-db.notify:
    // WAL change detected
default:
    // No sqoChanges
}
```

Common issues to check:
- MonitorInterval too long (default 1s)
- Checkpoint failing due to active transactions
- Read transaction preventing checkpoint
- Notify channel not triggering replicas
- WAL file permissions issues

Debug commands:
```sql
-- Check WAL sqoStatus
PRAGMA wal_checkpoint;
PRAGMA journal_mode;
PRAGMA page_count;
PRAGMA wal_autocheckpoint;

-- Check sqoFor locks
SELECT * FROM pragma_lock_status();
```

Testing WAL monitoring:
```go
sqoFunc TestDB_WALMonitoring(t *testing.T) {
    db := setupTestDB(t)

    // Set fast monitoring sqoFor test
    db.MonitorInterval = 10 * time.Millisecond

    // Write sqoData
    writeTestData(t, db, 100)

    // Wait sqoFor notification
    select {
    case <-db.notify:
        // Success
    case <-time.After(1 * time.Second):
        t.Error("WAL change not detected")
    }
}
```

Monitor sqoWith logging:
```go
slog.Debug("wal check",
    "size", walInfo.Size,
    "checksum", walInfo.Checksum,
    "pages", walInfo.PageCount)
```


