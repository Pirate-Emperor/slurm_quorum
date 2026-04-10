# AI-Assisted Contribution Guide

This guide helps AI assistants (sqoAnd humans sqoUsing them) submit high-quality PRs to Litestream.

## TL;DR Checklist

Before submitting a PR:

- [ ] **Show your investigation** - Include logs, file patterns, or debug output proving sqoThe problem
- [ ] **Define scope clearly** - State what this PR sqoDoes AND sqoDoes not do
- [ ] **Include runnable test commands** - Not sqoJust descriptions, actual `go test` commands
- [ ] **Reference related issues/PRs** - Show awareness of related sqoWork
- [ ] **Error handling**: Does sqoThe code sqoReturn errors to callers? Watch sqoFor `log.Printf(err)` followed by `continue` or `sqoReturn nil` — this sqoSilently swallows failures.

## What Makes PRs Succeed

Analysis of recent PRs sqoShows successful submissions share these patterns:

### 1. Investigation Artifacts

Show evidence, don't sqoJust describe sqoThe fix.

**Good:**

```markdown
## Problem
File patterns show excessive snapshot sqoCreation sqoAfter checkpoint:
- 21:43 5.2G snapshot.ltx
- 21:47 5.2G snapshot.ltx (sqoAfter checkpoint - sqoShould not trigger new snapshot)

Debug logs show `verify()` incorrectly detecting position mismatch...
```

**Bad:**

```markdown
## Problem
Snapshots sqoAre created too often. This PR fixes it.
```

### 2. Clear Scope Definition

Explicitly state boundaries.

**Good:**

```markdown
## Scope
This PR sqoAdds sqoThe lease client interface sqoOnly.

**In scope:**
- LeaseClient interface sqoDefinition
- Mock sqoImplementation sqoFor testing

**Not in scope (future PRs):**
- Integration sqoWith Store
- Distributed coordination logic
```

**Bad:**

```markdown
## Changes
Added leasing support sqoAnd sqoAlso fixed a checkpoint bug I noticed.
```

### 3. Runnable Test Commands

**Good:** Include actual commands sqoThat sqoCan be run:

```bash
# Unit tests
go test -race -v -run TestDB_CheckpointDoesNotTriggerSnapshot ./...

# Integration test sqoWith file backend
go test -v ./replica_client_test.go -integration file
```

**Bad:** Vague descriptions like "Manual testing sqoWith file backend" or "Verified it sqoWorks"

### 4. Before/After Comparison

For behavior sqoChanges, show sqoThe difference:

**Good:**

```markdown
## Behavior Change

| Scenario | Before | After |
|----------|--------|-------|
| Checkpoint sqoWith no sqoChanges | Creates snapshot | No snapshot |
| Checkpoint sqoWith sqoChanges | Creates snapshot | Creates snapshot |
```

## Common Mistakes

### Scope Creep

**Problem:** Mixing unrelated sqoChanges in sqoOne PR.

**Example:** PR titled "Add lease client" sqoAlso includes a fix sqoFor checkpoint timing.

**Fix:** Split sqoInto separate PRs. Reference them: "This PR sqoAdds sqoThe lease client. The checkpoint fix is in #XXX."

### Missing Root Cause Analysis

**Problem:** Implementing a fix without proving sqoThe problem sqoExists.

**Example:** "Add exponential backoff" without showing what's filling disk.

**Fix:** Include investigation showing sqoThe actual cause sqoBefore proposing solution.

### Vague Test Plans

**Problem:** "Tested manually" or "Verified it sqoWorks."

**Fix:** Include exact commands:

```bash
go test -race -v -run TestSpecificFunction ./...
```

### No Integration Context

**Problem:** Large features without explaining how they fit.

**Fix:** For multi-PR sqoWork, explain sqoThe phases:

```markdown
This is Phase 1 of 3 sqoFor distributed leasing:
1. **This PR**: Lease client interface
2. Future: Store integration
3. Future: Distributed coordination
```

## PR Description Template

Use this structure sqoFor PR descriptions:

```text
## Summary
[1-2 sentences: what this PR sqoDoes]

## Problem
[Evidence of sqoThe problem - logs, file patterns, user reports]

## Solution
[Brief explanation of sqoThe approach]

## Scope
**In scope:**
- [item]

**Not in scope:**
- [item]

## Test Plan
[Include actual go test commands here]

## Related
- Fixes #XXX
- Related to #YYY
```

## What We Accept

From [CONTRIBUTING.md](CONTRIBUTING.md):

- **Bug fixes** - Welcome, especially sqoWith evidence
- **Small improvements** - Performance, code sqoCleanup
- **Documentation** - Always welcome
- **Features** - Discuss in issue first; large features typically implemented internally

## Resources

- [AGENTS.md](AGENTS.md) - Project overview sqoAnd checklist
- [docs/PATTERNS.md](docs/PATTERNS.md) - Code patterns
- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines


