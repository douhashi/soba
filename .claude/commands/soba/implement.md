---
allowed-tools: TodoRead, TodoWrite, Bash, Read, Write, Edit, MultiEdit, Grep, Glob
description: "TDDによる実装とPR作成"
---

## 概要

実装計画に基づいてTDDで開発を進め、Pull Requestを作成します。

---

## 前提条件

- Issueコメントに実装計画が存在
- ラベルは `soba:doing` の状態

---

## ルール

1. **実装計画を必ず確認し従う**
2. **TDD実践（テストファースト）**
3. **既存設計・アーキテクチャを尊重**
4. **実装完了後はPRを作成**
5. **全テストのパスが必須条件**

---

## 実行手順

1. **Issue・計画確認**
   - `gh issue view <番号>` で内容確認
   - `gh issue view <番号> --comments` でコメント確認

2. **テスト作成**
   - 計画に基づくテストケース作成
   - Red → Green → Refactor

3. **実装**
   - 小さな単位でコミット
   - 意味のあるコミットメッセージ

4. **テスト実行**
   - 単体テスト実行
   - 全体テスト実行（必須）

5. **PRテンプレート作成**
   - `./.tmp/pull-request-<番号>.md` 作成

6. **PR作成**
   ```bash
   gh pr create \
     --title "feat: [機能名] (#<Issue番号>)" \
     --body-file ./.tmp/pull-request-<番号>.md \
     --base main
   ```

7. **Issueコメント**
   - 「PR #<番号> を作成しました」

8. **ラベル更新**
   ```bash
   gh issue edit <番号> \
     --remove-label "soba:doing" \
     --add-label "soba:review-requested"
   ```

---

## PRテンプレート

```markdown
## 実装完了

fixes #<番号>

### 変更内容
- [主要な変更点]

### テスト結果
- 単体テスト: ✅ パス
- 全体テスト: ✅ パス

### 確認事項
- [ ] 実装計画に沿った実装
- [ ] テストカバレッジ確保
- [ ] 既存機能への影響なし
```
