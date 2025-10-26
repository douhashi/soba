---
allowed-tools: TodoRead, TodoWrite, Bash, Read, Write, Edit, MultiEdit, Grep, Glob
description: "TDD implementation and PR creation"
---

## Overview

Proceed with TDD development based on implementation plan and create Pull Request.

---

## Prerequisites

- Implementation plan exists in Issue comments
- Label is in `soba:doing` state

---

## Rules

1. **Always check and follow the implementation plan**
2. **Practice TDD (Test First)**
3. **Respect existing design and architecture**
4. **Create PR after implementation completion**
5. **Passing all tests is a requirement**

## Important Notes

- Think hard and choose the best architecture with good maintainability. You don't need to respect existing implementations too much.
- Maintaining compatibility is not required. Code complexity due to maintaining backward compatibility is more harmful.
- Passing the full test suite is an **absolute requirement**. However, do not skip test code.

---

## Execution Steps

1. **Check Issue and Plan**
   - Check content with `gh issue view <number>`
   - Check comments with `gh issue view <number> --comments`

2. **Create Tests**
   - Create test cases based on plan
   - Red → Green → Refactor

3. **Implementation**
   - Commit in small units
   - Meaningful commit messages

4. **Run Tests**
   - Run full test suite (required) # Timeout 600000

5. **Create PR Template**
   - Create `./.tmp/pull-request-<number>.md`

6. **Create PR**
   ```bash
   gh pr create \
     --title "feat: [feature name] (#<Issue number>)" \
     --body-file ./.tmp/pull-request-<number>.md \
     --base main
   ```

7. **Issue Comment**
   - "Created PR #<number>"

8. **Update Labels**
   ```bash
   gh issue edit <number> \
     --remove-label "soba:doing" \
     --add-label "soba:review-requested"
   ```

---

## PR Template

```markdown
## Implementation Complete

fixes #<number>

### Changes
- [Main changes]

### Test Results
- Unit tests: ✅ Pass
- Full test suite: ✅ Pass

### Checklist
- [ ] Implementation follows the plan
- [ ] Test coverage ensured
- [ ] No impact on existing features
```
