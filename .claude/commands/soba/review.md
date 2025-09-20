---
allowed-tools: Bash, Read, Write, Edit, MultiEdit, Grep, Glob, LS
description: "Review a Pull Request for a soba Issue"
---

# Review PR

PRレビューを実施します。

## Context

- Issue番号: $ARGUMENTS

## Workflow

### 1. Issue確認

```bash
GH_PAGER= gh issue view <issue-number>
GH_PAGER= gh issue view <issue-number> --comments
```

### 2. PR確認

```bash
GH_PAGER= gh pr view <PR-number>
GH_PAGER= gh pr view <PR-number> --json mergeable,mergeStateStatus
```

### 3. コード変更確認

```bash
GH_PAGER= gh pr diff <PR-number>
```

レビュー観点:
- コーディング規約への準拠
- テストの実装状況
- セキュリティ上の懸念
- 不要な差分の有無

### 4. CI確認（必須・完了まで待機）

```bash
gh pr checks <PR-number> --watch  # Timeout 600000
```

⚠️ **重要**: CI完了前にレビュー結果を投稿しないこと

### 5. レビュー結果投稿

`./.tmp/review-result-<issue-number>.md`を作成:

```markdown
## レビュー結果

- Issue: #<issue-number>
- PR: #<PR-number>

### ✅ 判定
- [ ] 承認（LGTM）
- [ ] 修正要求

### 🔄 マージ状態
- [ ] コンフリクトなし
- [ ] コンフリクトあり（要リベース）

### 👍 良い点
- [実装の良い点]

### 🛠 改善提案
- [具体的な改善点]
```

投稿:
```bash
gh pr comment <PR-number> --body "$(cat ./.tmp/review-result-<issue-number>.md)"
```

### 6. ラベル更新

承認の場合:
```bash
gh issue edit <issue-number> --remove-label "soba:reviewing" --add-label "soba:done"
gh pr edit <PR-number> --add-label "soba:lgtm"
```

修正要求の場合:
```bash
gh issue edit <issue-number> --remove-label "soba:reviewing" --add-label "soba:requires-changes"
```
