---
allowed-tools: Bash, Read, Write, Edit, MultiEdit, Grep, Glob, LS
description: "Review a Pull Request for a soba Issue"
---

# Review PR

Conduct PR review.

## Context

- Issue number: $ARGUMENTS

## Workflow

### 1. Check Issue

```bash
GH_PAGER= gh issue view <issue-number>
GH_PAGER= gh issue view <issue-number> --comments
```

### 2. Check PR

```bash
GH_PAGER= gh pr view <PR-number>
GH_PAGER= gh pr view <PR-number> --json mergeable,mergeStateStatus
```

### 3. Check Code Changes

```bash
GH_PAGER= gh pr diff <PR-number>
```

Review points:
- Compliance with coding standards
- Test implementation status
- Security concerns
- Presence of unnecessary diffs

### 4. Check CI (Required - wait for completion)

```bash
gh pr checks <PR-number> --watch  # Timeout 600000
```

⚠️ **Important**: Do not post review results before CI completion

### 5. Post Review Results

Create `./.tmp/review-result-<issue-number>.md`:

```markdown
## Review Results

- Issue: #<issue-number>
- PR: #<PR-number>

### ✅ Decision
- [ ] Approve (LGTM)
- [ ] Request changes

### 🔄 Merge Status
- [ ] No conflicts
- [ ] Conflicts exist (rebase required)

### 👍 Good Points
- [Good aspects of implementation]

### 🔧 Improvement Suggestions
- [Specific improvement points]
```

Post:
```bash
gh pr comment <PR-number> --body "$(cat ./.tmp/review-result-<issue-number>.md)"
```

### 6. Update Labels

For approval:
```bash
gh issue edit <issue-number> --remove-label "soba:reviewing" --add-label "soba:done"
gh pr edit <PR-number> --add-label "soba:lgtm"
```

For change requests:
```bash
gh issue edit <issue-number> --remove-label "soba:reviewing" --add-label "soba:requires-changes"
```
