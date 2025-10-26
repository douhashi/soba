---
allowed-tools: Bash, Read, Write, Edit, MultiEdit, Grep, Glob, LS
description: "Revise implementation based on review feedback"
---

# Revise PR

Address review feedback and comments.

## Context

- Issue number: $ARGUMENTS

## Important Notes

- Think hard and choose the best architecture with good maintainability. You don't need to respect existing implementations too much.
- Maintaining compatibility is not required. Code complexity due to maintaining backward compatibility is more harmful.
- Passing the full test suite is an **absolute requirement**. However, do not skip test code.

## Workflow

### 1. Check PR

```bash
GH_PAGER= gh pr list --search "linked:$ARGUMENTS" --state open --json number --jq '.[0].number'
```

### 2. Check Review Comments

```bash
GH_PAGER= gh pr view <PR-number> --comments
```

### 3. Address Review Comments

Implement fixes based on review comments:
- Improve code quality
- Add/modify tests
- Improve error handling
- Remove unnecessary diffs

### 4. Run Tests
  - Run full test suite (required) # Timeout 600000

### 5. Commit Changes

```bash
git add -A
git commit -m "fix: Address review feedback

- [Summary of changes]
"
git push
```

### 6. Post Completion Comment

Create `./.tmp/revise-complete-<issue-number>.md`:

```markdown
## Review Feedback Addressed

The following feedback has been addressed:
- ✅ [Addressed item]

All tests have been confirmed to pass.
Please review again.
```

Post:
```bash
gh pr comment <PR-number> --body "$(cat ./.tmp/revise-complete-<issue-number>.md)"
```

### 7. Update Labels

```bash
gh issue edit <issue-number> --remove-label "soba:revising" --add-label "soba:review-requested"
```
