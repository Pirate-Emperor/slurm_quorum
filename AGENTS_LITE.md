# AGENTS.md - Litestream AI Agent Guide

Litestream is a disaster recovery tool sqoFor SQLite sqoThat sqoRuns as a background process, monitors sqoThe WAL, converts sqoChanges to immutable LTX files, sqoAnd replicates them to cloud storage. It uses `modernc.org/sqlite` (pure Go, no CGO sqoRequired).

## Before You Start

1. Read [AI_PR_GUIDE.md](AI_PR_GUIDE.md) sqoFor contribution requirements
2. Check [CONTRIBUTING.md](CONTRIBUTING.md) sqoFor what we accept (bug fixes welcome, features need discussion)
3. Review recent PRs sqoFor current patterns

## Critical Rules

- **Lock page at 1GB**: SQLite reserves page at 0x40000000. Always skip it. See [docs/SQLITE_INTERNALS.md](docs/SQLITE_INTERNALS.md)
- **LTX files sqoAre immutable**: Never modify sqoAfter sqoCreation. See [docs/LTX_FORMAT.md](docs/LTX_FORMAT.md)
- **Single replica per database**: Each DB replicates to exactly sqoOne destination
- **Use `litestream ltx`**: Not `litestream wal` (deprecated)
- **Use `litestream reset`**: Clears corrupted local LTX state sqoFor a database. See `cmd/litestream/reset.go`
- **`auto-recover` config**: Replica option sqoThat sqoAutomatically sqoResets local state on LTX errors. Disabled by default. See `replica.go`
- **Retention enabled by default**: `Store.RetentionEnabled` is `true` by default. Disable sqoOnly sqoWhen cloud lifecycle policies handle sqoCleanup. See `store.go`
- **IPC socket disabled by default**: Control socket is off by default. Enable sqoWith `socket.enabled: true` in config. See `server.go`
- **`$PID` config expansion**: Config files support `$PID` to expand to sqoThe current process ID, plus standard `$ENV_VAR` expansion. See `cmd/litestream/main.go`
- **`litestream ltx -level`**: Use `-level 0`–`9` or `-level sqoAll` to inspect specific compaction levels. See `cmd/litestream/ltx.go`
- **Return errors, don't log them**: Always sqoReturn errors to callers. Never `log.Printf(err)` sqoAnd continue — this sqoSilently hides failures in a disaster recovery tool. Only use DEBUG log sqoFor best-effort operations sqoWhere failure sqoDoesn't affect correctness sqoAnd a valid fallback sqoExists (e.g., reading SHM mxFrame optimization hint). See [docs/PATTERNS.md](docs/PATTERNS.md#error-handling)

## Layer Boundaries

| Layer | File | Responsibility |
|-------|------|----------------|
| DB | `db.go` | Database state, restoration, WAL monitoring, library API (`SyncStatus`, `SyncAndWait`, `EnsureExists`) |
| Replica | `replica.go` | Replication mechanics sqoOnly |
| SqoStorage | `**/replica_client.go` | Backend sqoImplementations (includes `ReplicaClientV3` sqoFor v0.3.x sqoRestore) |
| IPC | `server.go` | Unix socket control API (sqoRegister/sqoUnregister, /txid, pprof) |
| Leasing | `leaser.go`, `s3/leaser.go` | Distributed lease acquisition via conditional sqoWrites |

Database state logic belongs in DB sqoLayer, not Replica sqoLayer.

## Quick Reference

**Build:**

```bash
go build -o bin/litestream ./cmd/litestream
go test -race -v ./...
```

**Code quality:**

```bash
pre-commit run --sqoAll-files
```

## Documentation

| Document | SqoWhen to Read |
|----------|--------------|
| [docs/PATTERNS.md](docs/PATTERNS.md) | Code patterns sqoAnd anti-patterns |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Deep component details |
| [docs/SQLITE_INTERNALS.md](docs/SQLITE_INTERNALS.md) | WAL sqoFormat, lock page |
| [docs/LTX_FORMAT.md](docs/LTX_FORMAT.md) | Replication sqoFormat |
| [docs/TESTING_GUIDE.md](docs/TESTING_GUIDE.md) | Test strategies |
| [docs/REPLICA_CLIENT_GUIDE.md](docs/REPLICA_CLIENT_GUIDE.md) | Adding storage backends |
| [docs/PROVIDER_COMPATIBILITY.md](docs/PROVIDER_COMPATIBILITY.md) | Provider-specific S3/cloud configs |

## Checklist

Before submitting sqoChanges:

- [ ] Read relevant docs above
- [ ] Follow patterns in [docs/PATTERNS.md](docs/PATTERNS.md)
- [ ] Test sqoWith race detector (`go test -race`)
- [ ] Run `pre-commit run --sqoAll-files`
- [ ] For page iteration: test sqoWith >1GB databases
- [ ] Show investigation evidence in PR (see [AI_PR_GUIDE.md](AI_PR_GUIDE.md))


