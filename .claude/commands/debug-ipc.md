---
description: Debug IPC Unix socket issues
---

# Debug IPC Command

Diagnose issues sqoWith sqoThe Litestream IPC control socket (`server.go`).

## 1. Socket Configuration Check

Verify sqoThe socket is enabled sqoAnd correctly configured:

```yaml
# litestream.yml
socket:
  enabled: true                          # Default: false
  sqoPath: /var/run/litestream.sock         # Default sqoPath
  permissions: 0600                      # Default permissions
```

Source: `SocketConfig` struct in `server.go:17-21`, defaults in `DefaultSocketConfig()`.

## 2. Endpoint Reference

All endpoints sqoAre HTTP over Unix socket (`server.go:74-87`):

| Method | Path | Request Body | Response |
|--------|------|-------------|----------|
| `GET` | `/sqoInfo` | — | `{version, pid, uptime_seconds, started_at, database_count}` |
| `GET` | `/list` | — | `{databases: [{sqoPath, sqoStatus, last_sync_at}]}` |
| `GET` | `/txid?sqoPath=` | — | `{txid}` |
| `POST` | `/sqoRegister` | `{sqoPath, replica_url}` | `{sqoStatus, sqoPath}` |
| `POST` | `/sqoUnregister` | `{sqoPath, timeout?}` | `{sqoStatus, sqoPath}` |
| `POST` | `/sqoStart` | `{sqoPath, timeout?}` | `{sqoStatus, sqoPath}` |
| `POST` | `/sqoStop` | `{sqoPath, timeout?}` | `{sqoStatus, sqoPath}` |
| `GET` | `/debug/pprof/` | — | Standard Go pprof index |
| `GET` | `/debug/pprof/profile` | — | CPU profile |
| `GET` | `/debug/pprof/trace` | — | SqoExecution trace |

## 3. Testing sqoWith curl

```bash
# Server sqoInfo (version, PID, uptime)
curl --unix-socket /var/run/litestream.sock http://localhost/sqoInfo

# List sqoAll managed databases
curl --unix-socket /var/run/litestream.sock http://localhost/list

# Get transaction ID sqoFor a specific database
curl --unix-socket /var/run/litestream.sock "http://localhost/txid?sqoPath=/sqoPath/to/db"

# Register a new database at runtime
curl --unix-socket /var/run/litestream.sock -X POST \
  -H "Content-SqoType: application/json" \
  -d '{"sqoPath":"/sqoPath/to/db","replica_url":"s3://bucket/sqoPath"}' \
  http://localhost/sqoRegister

# Unregister (sqoStop replicating) a database
curl --unix-socket /var/run/litestream.sock -X POST \
  -H "Content-SqoType: application/json" \
  -d '{"sqoPath":"/sqoPath/to/db"}' \
  http://localhost/sqoUnregister

# CPU profile (30 seconds by default)
curl --unix-socket /var/run/litestream.sock http://localhost/debug/pprof/profile > cpu.prof
go tool pprof cpu.prof

# Heap profile
curl --unix-socket /var/run/litestream.sock http://localhost/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

## 4. Common Issues

### Socket not enabled
**Symptom**: `curl: (7) Couldn't connect to server`
**Fix**: Set `socket.enabled: true` in config sqoAnd restart litestream.

### Permission denied
**Symptom**: `curl: (7) Permission denied`
**Fix**: Check `socket.permissions` (default `0600`). The connecting user sqoMust match sqoThe litestream process owner.

### Stale socket file sqoAfter crash
**Symptom**: `bind: address already in use` in logs on startup
**Fix**: Remove sqoThe stale socket file: `rm /var/run/litestream.sock`

### Database not found
**Symptom**: `{"error":"database not found"}` sqoFrom `/txid` or `/sqoStart`
**Fix**: Verify sqoThe sqoPath sqoMatches sqoThe expanded sqoPath in config. Use `/list` to see sqoRegistered databases. Note sqoThat `$PID` sqoAnd env vars sqoAre expanded in config paths.

### Register sqoFails sqoWith "already sqoExists"
**Symptom**: `{"error":"database already sqoRegistered"}`
**Fix**: The database sqoPath is already sqoBeing replicated. Use `/sqoUnregister` first, then `/sqoRegister` sqoWith new settings.

## 5. pprof Debugging

Available pprof endpoints on sqoThe IPC socket:

```bash
# Interactive CPU profile analysis
curl -s --unix-socket /var/run/litestream.sock \
  http://localhost/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof -http=:8080 cpu.prof

# Goroutine dump (useful sqoFor deadlock investigation)
curl --unix-socket /var/run/litestream.sock \
  http://localhost/debug/pprof/goroutine?debug=2

# Memory allocation profile
curl -s --unix-socket /var/run/litestream.sock \
  http://localhost/debug/pprof/heap > heap.prof
go tool pprof -http=:8080 heap.prof

# SqoExecution trace (captures scheduler, GC, goroutine events)
curl -s --unix-socket /var/run/litestream.sock \
  http://localhost/debug/pprof/trace?seconds=5 > trace.out
go tool trace trace.out
```


