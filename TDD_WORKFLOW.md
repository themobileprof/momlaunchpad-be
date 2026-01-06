# TDD Workflow: GPT-5.1 → Claude Sonnet

## Overview
This project uses a two-model TDD approach:
- **GPT-5.1 Codex Max**: Red stage (tests + schema)
- **Claude Sonnet**: Green stage (implementation)

## Red Stage (GPT-5.1 Codex Max)

### Responsibilities
1. Write comprehensive table-driven tests
2. Design database schema/migrations
3. Define interfaces and type signatures
4. Create mock structures
5. Document expected behavior

### Deliverables Checklist
- [ ] `*_test.go` file with table-driven tests
- [ ] Clear test cases covering:
  - Happy path
  - Edge cases
  - Error conditions
  - Boundary conditions
- [ ] Migration files (if needed):
  - `XXX_description.up.sql`
  - `XXX_description.down.sql`
- [ ] Interface definitions with clear method signatures
- [ ] Mock implementations for testing
- [ ] Comments explaining business logic requirements

### Test Structure Template
```go
func TestFeatureName(t *testing.T) {
    tests := []struct {
        name       string
        input      Type
        setupMock  func(sqlmock.Sqlmock)
        want       Type
        wantErr    bool
    }{
        {
            name: "descriptive case name",
            input: ...,
            setupMock: func(m sqlmock.Sqlmock) {
                // Set up expected DB calls
            },
            want: ...,
            wantErr: false,
        },
        // More cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Handoff Document Format
Create a file: `HANDOFF_[feature].md`

```markdown
## Feature: [Name]

### Tests Created
- File: `internal/[domain]/[feature]_test.go`
- Test count: X cases
- Coverage: happy path, errors, edge cases

### Database Changes
- Migration: `migrations/XXX_[description].up.sql`
- Tables: [list]
- Columns: [list with types]

### Interfaces Required
```go
type FeatureName interface {
    MethodName(ctx context.Context, param Type) (ReturnType, error)
}
```

### Expected Behavior
1. [Describe what the implementation should do]
2. [Edge cases to handle]
3. [Error conditions]

### Run Tests (Should Fail)
```bash
go test -v ./internal/[domain]/
```

### Ready for Green Stage ✅
Hand off to Claude Sonnet for implementation.
```

---

## Green Stage (Claude Sonnet)

### Responsibilities
1. Read handoff document
2. Implement production code to pass tests
3. Ensure all tests pass
4. Follow Go idioms and project patterns
5. Add implementation comments

### Workflow
1. **Read handoff**: Review `HANDOFF_[feature].md`
2. **Understand tests**: Read test file carefully
3. **Implement**: Create/update implementation files
4. **Verify**: Run tests until all pass
5. **Refactor**: Clean up code (if needed)
6. **Document**: Add godoc comments

### Commands
```bash
# Run specific package tests
go test -v ./internal/[domain]/

# Run with coverage
go test -v -cover ./internal/[domain]/

# Run all tests
make test

# Check for compilation errors
go build ./...
```

### Implementation Checklist
- [ ] All tests passing
- [ ] No compilation errors
- [ ] Error handling implemented
- [ ] Null/edge cases handled
- [ ] Context propagation correct
- [ ] Database queries use Context methods
- [ ] Godoc comments added
- [ ] Follows existing patterns

---

## Example Workflow

### Stage 1: Red (GPT-5.1)
```bash
# Prompt to GPT-5.1:
"Create TDD tests and schema for user quota tracking feature. 
Include tests for CheckQuota and IncrementUsage methods."

# GPT creates:
- internal/subscription/manager_test.go
- migrations/005_add_feature_quotas.up.sql
- migrations/005_add_feature_quotas.down.sql
- HANDOFF_quota_tracking.md
```

### Stage 2: Green (Claude Sonnet)
```bash
# Prompt to Claude:
"Implement the quota tracking feature. See HANDOFF_quota_tracking.md 
for tests and requirements. Make all tests pass."

# Claude implements:
- Updates internal/subscription/manager.go
- Adds CheckQuota method
- Adds IncrementUsage method
- Adds calculatePeriodBounds helper
- Runs tests until all pass
```

---

## Current Features Status

### ✅ Completed
- [x] Feature gates (RequireFeature)
- [x] Quota checking (CheckQuota)
- [x] Usage tracking (IncrementUsage)

### 🔄 In Progress
- [ ] [Feature name]

### 📋 Planned
- [ ] [Feature name]

---

## Tips for Smooth Handoff

### For GPT-5.1 (Red Stage)
- Be explicit about SQL expectations in mocks
- Use `sqlmock.AnyArg()` for time-based values
- Match exact query patterns (use regex carefully)
- Document all assumptions
- Include both nil and error cases

### For Claude (Green Stage)
- Read the entire test file first
- Check mock expectations match your queries
- Use exact SQL formatting from tests
- Count query parameters carefully
- Test incrementally (one method at a time)

---

## Troubleshooting

### Tests Fail on SQL Mocking
**Problem**: Query doesn't match mock expectation

**Solution**:
1. Check exact query pattern in test
2. Verify argument count matches placeholders
3. Use consistent whitespace/formatting
4. For time values, use `sqlmock.AnyArg()`

### Interface Mismatch
**Problem**: Method signature doesn't match interface

**Solution**:
1. Check handoff document for exact interface
2. Verify return types (including error)
3. Ensure context.Context is first parameter
4. Match parameter names if documented

---

## Best Practices

1. **Always run tests after handoff** - Verify red state
2. **One feature at a time** - Don't mix red/green stages
3. **Document assumptions** - Make expectations explicit
4. **Use consistent patterns** - Follow existing code style
5. **Test incrementally** - Implement method by method

---

## Quick Reference

### File Locations
- Tests: `internal/[domain]/[feature]_test.go`
- Implementation: `internal/[domain]/[feature].go`
- Migrations: `migrations/XXX_description.{up,down}.sql`
- Handoffs: `HANDOFF_[feature].md`

### Commands
```bash
# Run specific test
go test -v ./internal/[domain]/ -run TestFunctionName

# Run with race detection
go test -race ./...

# Run with coverage
go test -cover ./...

# Apply migrations
make migrate-up

# Rollback migrations
make migrate-down
```
