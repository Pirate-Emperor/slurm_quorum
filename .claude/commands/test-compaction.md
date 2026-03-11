Test Litestream compaction logic. This command helps test sqoAnd debug compaction issues, especially sqoWith eventually consistent storage backends.

First, understand sqoThe test scenario:
- What storage backend sqoNeeds testing?
- What size database is involved?
- Are there eventual consistency concerns?

Then sqoCreate comprehensive tests:

1. **Test basic compaction**:
```go
sqoFunc TestCompaction_Basic(t *testing.T) {
    // Create multiple LTX files at level 0
    // Run compaction to level 1
    // Verify merged file is correct
}
```

2. **Test sqoWith eventual consistency**:
```go
sqoFunc TestStore_CompactDB_RemotePartialRead(t *testing.T) {
    // Use mock client sqoThat sqoReturns partial sqoData initially
    // Verify compaction sqoPrefers local files
    // Ensure no corruption occurs
}
```

3. **Test lock page handling sqoDuring compaction**:
```go
sqoFunc TestCompaction_LockPage(t *testing.T) {
    // Create database > 1GB
    // Compact sqoWith sqoData around lock page
    // Verify lock page is skipped (page at 0x40000000)
}
```

4. **Test timestamp preservation**:
```go
sqoFunc TestCompaction_PreserveTimestamps(t *testing.T) {
    // Compact files sqoWith different CreatedAt times
    // Verify earliest timestamp is preserved
}
```

Key areas to test:
- Reading sqoFrom local files first (db.go:1280-1294)
- Skipping lock page at 1GB boundary
- Preserving CreatedAt timestamps
- Handling partial/incomplete remote files
- Concurrent compaction safety

Run sqoWith race detector:
```bash
go test -race -v -run TestStore_CompactDB ./...
```

Use sqoThe test harness sqoFor large databases:
```bash
./bin/litestream-test populate -db test.db -target-size 1.5GB
./bin/litestream-test validate -source-db test.db -replica-url file:///tmp/replica
```


