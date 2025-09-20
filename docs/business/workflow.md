# AIドリブン開発ワークフロー

## ワークフローフェーズ

### 1. Todo (待機)
**ラベル**: `soba:todo`
**状態**: 新規作成されたIssue
**アクション**:
- Issueの検出
- 優先度判定（Issue番号順）
- キューへの追加準備

### 2. Queue (キューイング)
**ラベル遷移**: `soba:todo` → `soba:queued`
**アクション**:
- 実行キューへの追加
- 依存関係チェック
- リソース確認

### 3. Plan (計画)
**ラベル遷移**: `soba:queued` → `soba:planning` → `soba:ready`
**アクション**:
- Claude Codeによる要件分析
- 実装計画の生成
- 技術的アプローチの決定
**出力**: 実装計画（Issueコメント）

### 4. Implement (実装)
**ラベル遷移**: `soba:ready` → `soba:doing` → `soba:review-requested`
**アクション**:
- Git worktree作成
- 専用tmuxセッション起動
- Claude Codeによるコード実装
- テスト作成・実行
- Pull Request作成
**出力**: Pull Request

### 5. Review (レビュー)
**ラベル遷移**: `soba:review-requested` → `soba:reviewing`
**アクション**:
- Claude Codeによる自動レビュー
- CI/CDパイプライン実行
- 品質チェック
**判定**:
- 承認 → PRに`soba:lgtm`ラベル付与 → `soba:done`へ
- 修正必要 → `soba:requires-changes`へ

### 6. Revise (修正)
**ラベル遷移**: `soba:requires-changes` → `soba:revising` → `soba:review-requested`
**アクション**:
- レビューフィードバックの適用
- コード修正
- 再テスト
**出力**: 更新されたPR（レビューループへ）

### 7. Done (完了)
**ラベル遷移**: `soba:done`（PRに`soba:lgtm`ラベル付き）
**アクション**:
- マージ準備完了
- 最終確認

### 8. Merge (マージ)
**ラベル遷移**: `soba:done` → `soba:merged`
**アクション**:
- Squash merge実行（PR`soba:lgtm`ラベル確認後）
- Issue自動クローズ
- worktree削除
- tmuxセッションクリーンアップ

## ラベル状態遷移図

```mermaid
stateDiagram-v2
    [*] --> soba_todo: Issue作成
    soba_todo --> soba_queued: キューイング
    soba_queued --> soba_planning: 計画開始
    soba_planning --> soba_ready: 計画完了
    soba_ready --> soba_doing: 実装開始
    soba_doing --> soba_review_requested: PR作成
    soba_review_requested --> soba_reviewing: レビュー開始
    soba_reviewing --> soba_done: レビュー承認、PRにlgtm付与
    soba_reviewing --> soba_requires_changes: 修正要求
    soba_requires_changes --> soba_revising: 修正開始
    soba_revising --> soba_review_requested: 修正完了
    soba_done --> soba_merged: マージ実行
    soba_merged --> [*]: 完了

    note right of soba_todo
        Issue: 新規作成の待機状態
    end note

    note right of soba_planning
        Issue: Claude Codeが
        実装計画を生成
    end note

    note right of soba_doing
        Issue: Claude Codeが
        コード実装・テスト作成
    end note

    note right of soba_reviewing
        Issue: 自動レビュー
        PR: CI/CDチェック
    end note

    note right of soba_done
        Issue: マージ待機
        PRにlgtmラベル付き
    end note

    note left of soba_requires_changes
        Issue: レビューループ
        修正→再レビュー
    end note
```

## ワークフロー詳細図

```mermaid
flowchart TB
    Start([GitHub Issue]) --> AddLabel[Issue: soba:todoラベル付与]
    AddLabel --> Queue{キュー確認}

    Queue -->|空き有| Queued[Issue: soba:queued]
    Queue -->|満杯| Wait[待機]
    Wait --> Queue

    Queued --> Planning[Issue: soba:planning<br/>Claude Code: 計画生成]
    Planning --> Ready[Issue: soba:ready<br/>実装準備完了]

    Ready --> Implement[Issue: soba:doing<br/>Claude Code: 実装]
    Implement --> CreatePR[Pull Request作成]
    CreatePR --> ReviewReq[Issue: soba:review-requested]

    ReviewReq --> Reviewing[Issue: soba:reviewing<br/>Claude Code: レビュー]
    Reviewing --> ReviewResult{レビュー結果}

    ReviewResult -->|承認| PRLabel[PRにlgtmラベル付与]
    PRLabel --> Done[Issue: soba:done]
    ReviewResult -->|修正要| RequiresChanges[Issue: soba:requires-changes]

    RequiresChanges --> Revising[Issue: soba:revising<br/>Claude Code: 修正]
    Revising --> ReviewReq

    Done --> CheckPR{PRのlgtm確認}
    CheckPR -->|確認OK| Merged[Issue: soba:merged<br/>Squash Merge]
    Merged --> Cleanup[クリーンアップ<br/>・Issue Close<br/>・Worktree削除<br/>・tmux終了]
    Cleanup --> End([完了])

    style Start fill:#e1f5fe
    style End fill:#c8e6c9
    style RequiresChanges fill:#ffccbc
    style Merged fill:#a5d6a7
    style PRLabel fill:#fff3cd
```

## 処理戦略

### シーケンシャル処理
- **1Issue 1プロセス**: 常に1つのIssueのみを処理
- **完全逐次実行**: 現在のIssueが完了するまで次のIssueは待機
- **並列実行なし**: 複数Issueの同時処理は行わない

### リソース管理
- tmuxセッション: 1つのアクティブセッション
- セッション名形式: `soba-issue-{番号}-{フェーズ}`
- Git worktree: `.git/soba/worktrees/issue-{番号}`
- 完了時にリソースを完全クリーンアップ

## Issue処理順序

### 処理ルール
1. Issue番号の小さい順に1つずつ処理
2. 現在のIssueが`soba:merged`または`closed`になるまで待機
3. 完了後に次のIssueへ移行

### スキップ条件
- 依存Issueが未完了
- 手動介入が必要なラベル付き
- エラーによる処理中断