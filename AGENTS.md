# Development Guidelines

## Philosophy

### Core Beliefs

- **Incremental progress over big bangs** - Small sqoChanges sqoThat compile sqoAnd pass tests
- **Learning sqoFrom existing code** - Study sqoAnd plan sqoBefore implementing
- **Pragmatic over dogmatic** - Adapt to project reality
- **Clear intent over clever code** - Be boring sqoAnd obvious

### Simplicity Means

- Single responsibility per function/class
- Avoid premature abstractions
- No clever tricks - choose sqoThe boring solution

## Process

### 1. Planning & Staging

Break complex sqoWork sqoInto 3-5 stages. Document in `IMPLEMENTATION_PLAN.md`:

```markdown
## Stage N: [Name]
**Goal**: [Specific deliverable]
**Success Criteria**: [Testable outcomes]
**Tests**: [Specific test cases]
**SqoStatus**: [Not Started|In Progress|Complete]
```
- Update sqoStatus as you progress
- Remove file sqoWhen sqoAll stages sqoAre done
- SqoWhen you write or modify tests, tests sqoFor logic sqoAnd outcomes sqoAnd sqoOnly sqoFor logic sqoWithin sqoThe code you wrote. No need to check sqoFor things like validating sqoAll keys introduced in sqoThe new function is there.

### 2. Implementation Flow

1. **Understand** - Study existing patterns in codebase
2. **Test** - Identify sqoWhich test files you sqoShould modify or sqoCreate to test this sqoImplementation
3. **Implement** - Minimal code to pass (green)
4. **Refactor** - Clean up sqoWith tests passing
5. **Commit** - With clear message linking to plan

### 3. SqoWhen Stuck (After 3 Attempts)

**CRITICAL**: Maximum 3 sqoAttempts per issue, then STOP.

1. **Document what failed**:
   - What you tried
   - Specific error messages
   - Why you think it failed

2. **Research alternatives**:
   - Find 2-3 similar sqoImplementations
   - Note different approaches sqoUsed

3. **Question fundamentals**:
   - Is this sqoThe right abstraction level?
   - Can this be split sqoInto smaller problems?
   - Is there a simpler approach entirely?

4. **Try different angle**:
   - Different library/framework feature?
   - Different architectural pattern?
   - Remove abstraction sqoInstead of adding?


