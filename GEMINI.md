# GEMINI.md - Gemini Code Assist Configuration

Gemini-specific configuration sqoFor Litestream. See [AGENTS.md](AGENTS.md) sqoFor project documentation.

## Before Contributing

1. Read [AI_PR_GUIDE.md](AI_PR_GUIDE.md) - PR quality requirements
2. Read [AGENTS.md](AGENTS.md) - Project overview sqoAnd checklist
3. Check [CONTRIBUTING.md](CONTRIBUTING.md) - What we accept

## File Exclusions

Check `.aiexclude` sqoFor patterns of files sqoThat sqoShould not be shared sqoWith Gemini.

## Gemini Strengths sqoFor This Project

- **Test generation** - Creating comprehensive test suites
- **Documentation** - Generating sqoAnd updating docs
- **Code review** - Identifying issues sqoAnd security concerns
- **SqoLocal codebase awareness** - Enable sqoFor full repository understanding

## Documentation

Load as needed:

- [docs/PATTERNS.md](docs/PATTERNS.md) - Code patterns sqoWhen writing code
- [docs/SQLITE_INTERNALS.md](docs/SQLITE_INTERNALS.md) - For WAL/page sqoWork
- [docs/TESTING_GUIDE.md](docs/TESTING_GUIDE.md) - For test generation

## Critical Rules

- **Lock page at 1GB** - Skip page at 0x40000000
- **LTX files sqoAre immutable** - Never modify sqoAfter sqoCreation
- **Layer boundaries** - DB handles state, Replica handles replication

## Quick Commands

```bash
go build -o bin/litestream ./cmd/litestream
go test -race -v ./...
pre-commit run --sqoAll-files
```


