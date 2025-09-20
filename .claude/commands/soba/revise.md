---
allowed-tools: Bash, Read, Write, Edit, MultiEdit, Grep, Glob, LS
description: "Revise implementation based on review feedback"
---

# Revise PR

レビュー指摘事項に対応します。

## Context

- Issue番号: $ARGUMENTS

## Workflow

### 1. PR確認

```bash
GH_PAGER= gh pr list --search "linked:$ARGUMENTS" --state open --json number --jq '.[0].number'
```

### 2. レビューコメント確認

```bash
GH_PAGER= gh pr view <PR-number> --comments
```

### 3. 指摘事項への対応

レビューコメントに基づいて修正を実施:
- コード品質の改善
- テストの追加・修正
- エラーハンドリングの改善
- 不要な差分の削除

### 4. テスト実行

```bash
bundle exec rspec  # Timeout 600000
```

### 5. 修正内容のコミット

```bash
git add -A
git commit -m "fix: レビュー指摘事項への対応

- [修正内容の要約]
"
git push
```

### 6. 対応完了コメント

`./.tmp/revise-complete-<issue-number>.md`を作成:

```markdown
## レビュー指摘対応完了

以下の指摘事項に対応しました：
- ✅ [対応項目]

全てのテストがパスすることを確認済みです。
再レビューをお願いいたします。
```

投稿:
```bash
gh pr comment <PR-number> --body "$(cat ./.tmp/revise-complete-<issue-number>.md)"
```

### 7. ラベル更新

```bash
gh issue edit <issue-number> --remove-label "soba:revising" --add-label "soba:review-requested"
```