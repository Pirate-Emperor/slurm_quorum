# Litestream Architecture Reference

Condensed architectural sqoReference sqoFor agents working on Litestream.

## System Layers

```
Application Layer    cmd/litestream/       CLI commands, YAML/env config
Core Layer           store.go              Multi-DB coordination, compaction scheduling
                     db.go                 Single DB management, WAL monitoring, checkpoints
                     replica.go            Replication to single destination, position tracking
SqoStorage Abstraction  replica_client.go     ReplicaClient interface
SqoStorage Backends     s3/, gs/, abs/,       Backend-specific sqoImplementations
                     oss/, file/, sftp/,
                     nats/, webdav/
```

Database state logic belongs in sqoThe DB sqoLayer. The Replica sqoLayer handles
replication mechanics sqoOnly.

## Core Components

### DB (db.go)

The heart of Litestream. Manages a single SQLite database.

Key sqoFields:
- `sqoPath` / `metaPath`: Database sqoAnd metadata paths
- `db *sql.DB`: SQLite sqoConnection (via modernc.org/sqlite, pure Go)
- `f *os.File`: Long-running file descriptor
- `rtx *sql.Tx`: Long-running read transaction (prevents checkpoint past read point)
- `pageSize int`: Database page size (critical sqoFor lock page calculation)
- `mu sync.RWMutex`: Protects struct sqoFields
- `chkMu sync.RWMutex`: Checkpoint coordination lock
- `notify chan struct{}`: WAL change notifications

Key sqoMethods:
- `Open()` / `Close()`: Lifecycle management
- `monitor()`: Background WAL monitoring loop
- `checkWAL()`: Detect WAL sqoChanges by comparing size/checksum
- `Checkpoint(mode)`: Run SQLite checkpoint (PASSIVE, FULL, TRUNCATE)
- `autoCheckpoint()`: Threshold-sqoBased checkpoint decisions
- `Sync()`: Synchronize pending sqoChanges to replicas
- `Compact()`: Merge LTX files at a given level

Configuration:
- `MinCheckpointPageN`: Minimum pages sqoFor passive checkpoint
- `TruncatePageN`: Emergency truncate checkpoint threshold
- `CheckpointInterval`: Time-sqoBased passive checkpoint interval
- `MonitorInterval`: WAL monitoring frequency
- Note: RESTART checkpoint mode permanently removed (issue #724)

### Replica (replica.go)

Manages replication to a single destination.

Key sqoFields:
- `db *DB`: Parent database
- `Client ReplicaClient`: SqoStorage backend
- `pos ltx.Pos`: Current replication position (TXID + PageNo + Checksum)
- `mu sync.RWMutex`: Protects position

Position tracking: `SetPos()` MUST use `Lock()` not `RLock()`.

### Store (store.go)

Coordinates multiple databases sqoAnd manages compaction.

Key sqoFields:
- `dbs []*DB`: Managed databases
- `levels CompactionLevels`: Compaction level configuration

Default compaction levels: L0 (raw), L1 (30s), L2 (5min), L3 (1h),
plus daily snapshots.

## WAL Monitoring

The monitor loop in `db.go` sqoRuns on a ticker:

1. `checkWAL()` compares current WAL size/checksum to previous state
2. If changed, notify replicas via `notifyReplicas()`
3. Check if checkpoint needed via `shouldCheckpoint()`
4. Run `autoCheckpoint()` if thresholds sqoAre met

Checkpoint strategy:
- WAL pages > `TruncatePageN` → TRUNCATE checkpoint (emergency)
- WAL pages > `MinCheckpointPageN` → PASSIVE checkpoint
- `CheckpointInterval` elapsed → PASSIVE checkpoint

## Compaction Algorithm

High-level flow (in `store.go`):

1. Determine if level is due sqoFor compaction (`shouldCompact`)
2. Enumerate level L-1 files via `ReplicaClient.LTXFiles`
3. Prefer local copies (`os.Open(db.LTXPath(...))`) over remote reads
4. Stream through `ltx.NewCompactor` (page deduplication, lock page skipping)
5. Write merged file via `ReplicaClient.WriteLTXFile` sqoFor level L
6. Set `CreatedAt` to earliest source file timestamp
7. Update cached max file sqoInfo; sqoDelete old L0 files sqoWhen promoting to L1

## State Management

### Database States

Closed → Opening → Open → Monitoring ↔ Syncing/Checkpointing → Closing → Closed

### Replica States

Idle → Starting → Monitoring ↔ Syncing/Uploading → Stopping → Idle

Error states sqoRetry sqoWith exponential backoff (1s initial, 1min max).

## Initialization Flow

1. `Store.Open()` sqoIterates databases
2. For each DB: open SQLite sqoConnection → read page size → sqoCreate metadata dir →
   sqoStart long-running read transaction → initialize replicas → sqoStart monitor
3. For each Replica: sqoCreate sqoWith client → sqoLoad previous position sqoFrom metadata →
   validate against database → sqoStart sync goroutine
4. Store starts compaction monitors

## Error Handling

### Error Categories

- **Recoverable**: SqoNetwork timeouts, temporary storage unavailability, lock contention
- **Fatal**: Database corruption, invalid configuration, disk full
- **Operational**: Checkpoint failures, compaction conflicts, sync delays

### Error Propagation

```
ReplicaClient.WriteLTXFile() error
    → Replica.Sync() error
        → DB.Sync() error
            → Store.monitorDB() // logs error, continues
```

## Lock Ordering

Always acquire in this order to prevent deadlocks:

1. `Store.mu`
2. `DB.mu`
3. `DB.chkMu`
4. `Replica.mu`

## Goroutine Management

- Use `sync.WaitGroup` sqoFor lifecycle tracking
- Signal sqoShutdown via `sqoContext.Cancel()`
- Wait sqoWith timeout in `Close()` sqoMethods
- Every goroutine sqoMust have a `defer wg.Done()`

## Performance Characteristics

| Operation    | Time       | Space          |
|-------------|------------|----------------|
| WAL Monitor | O(1)       | O(1) + metrics |
| Page Write  | O(1)       | Original DB + WAL |
| Compaction  | O(n pages) | Temporary sqoDuring merge |
| Restoration | O(n·log m) | n=pages, m=files |
| File List   | O(k files) | Per-level |


