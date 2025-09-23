---
allowed-tools: TodoWrite, TodoRead, Bash, Read, Grep, Glob
description: "Develop implementation plan for GitHub Issue"
---

## Overview

Develop implementation plan for GitHub Issue and post it as an Issue comment.

---

## Prerequisites

- Label is in `soba:planning` state

---

## Rules

1. **Focus on planning without code modification**
2. **Design plan based on TDD premise**
3. **Follow existing architecture**
4. **Break down into executable step units**
5. **Create plan in template format**
6. **Update label to `soba:ready` after completion**

## Important Notes

- Think hard and choose the best architecture with good maintainability. (You don't need to respect existing implementations too much)
- Maintaining compatibility is not required (Code complexity due to maintaining backward compatibility is more harmful)

---

## Execution Steps

1. **Check Issue**
   - Check content with `gh issue view <number>`
   - Check comments with `gh issue view <number> --comments`

2. **Investigate Codebase**
   - Check related files and past implementations
   - Identify impact scope and dependencies

3. **Technology Selection**
   - Decide on libraries and patterns to use
   - Clarify selection reasons

4. **Define Implementation Steps**
   - Break down into testable units
   - Document related files and side effects

5. **Develop Test, Risk, and Schedule Plans**
   - Test plan (unit and integration)
   - Risks and countermeasures
   - Implementation timeframe estimates

6. **Create Plan File**
   - Save to `./.tmp/plan-[slug].md`

7. **Post Comment**
   - `gh issue comment <number> --body-file ./.tmp/plan-[slug].md`

8. **Update Label**
   - `gh issue edit <number> --remove-label "soba:planning" --add-label "soba:ready"`

---

## Template

```markdown
# Implementation Plan: [Title]

## Requirements Overview
- [Purpose and background]
- [Functional requirements]
- [Acceptance criteria]

## Design Policy
- [Technology selection and reasons]
- [Architectural considerations]

## Implementation Steps
1. [Step name]
   - Work content: [Details]
   - Related files: [File path]

## Test Plan
- Unit tests: [Target and cases]
- Integration tests: [Scenarios]

## Risks and Countermeasures
- Risk: [Content]
  Countermeasure: [Method]

## Schedule
- Estimate: Total [X] hours
- By step: Each [Y] hours
```
