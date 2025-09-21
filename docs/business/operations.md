# Soba運用ガイド

## セットアップ

### 1. 初期設定
```bash
# 設定ファイル作成
soba init

# 設定確認
soba config
```

### 2. GitHub認証設定

#### 方法1: GitHub CLI (推奨)
```bash
# GitHub CLIでログイン
gh auth login

# 認証確認
gh auth status
```

#### 方法2: 環境変数
```bash
export GITHUB_TOKEN="ghp_xxxxxxxxxxxx"
```

#### 方法3: 設定ファイル
```yaml
# .soba/config.yml
github:
  token: ${GITHUB_TOKEN}
  auth_method: gh_cli  # または token
```

### 3. リポジトリ設定
```yaml
github:
  repository: owner/repo
```

## 基本操作

### ワークフロー起動
```bash
# フォアグラウンドで起動（デフォルト）
soba start

# デーモンとして起動
soba start -d
soba start --daemon

# デバッグモード
soba start --verbose
```

### 状態確認
```bash
# 実行状態確認
soba status

# tmuxセッション確認
tmux ls | grep soba
```

### ワークフロー停止
```bash
# 正常停止
soba stop

# 強制停止
soba stop --force
```

### tmuxセッションアクセス
```bash
# セッションにアタッチ
soba open issue-123-implement

# セッション一覧
soba sessions
```

## Issue作成ガイド

### 基本的なIssue
```markdown
Title: 機能Xを実装する

## 概要
機能Xの実装が必要です。

## 要件
- 要件1
- 要件2

## 完了条件
- テストが通る
- ドキュメント更新
```

### ラベル付与
```bash
# GitHub CLIでラベル付与
gh issue create --label "soba:todo"

# 既存Issueにラベル追加
gh issue edit 123 --add-label "soba:todo"
```

## 運用パターン

### 1. 継続的開発
```bash
# デーモン起動
soba start --daemon

# Issueを順次作成・ラベル付与
# → 自動的に処理される
```

### 2. バッチ処理
```bash
# 複数Issueにラベル付与
for i in 123 124 125; do
  gh issue edit $i --add-label "soba:todo"
done

# ワークフロー起動
soba start
```

### 3. 単一Issue処理
```bash
# 特定Issueのみ処理
soba process --issue 123
```

## トラブルシューティング

### Issue処理が進まない

#### 1. ラベル確認
```bash
gh issue view 123 --json labels
```

#### 2. ログ確認
```bash
# デーモンログ
tail -f /tmp/soba.log

# tmuxセッション出力
tmux capture-pane -t soba-issue-123-implement -p
```

#### 3. 手動介入
```bash
# ラベル修正
gh issue edit 123 --remove-label "soba:doing" --add-label "soba:ready"

# セッション削除
tmux kill-session -t soba-issue-123-implement
```

### プロセス管理

#### プロセス確認
```bash
# PIDファイル確認
cat /tmp/soba.pid

# プロセス確認
ps aux | grep soba
```

#### ゾンビプロセス処理
```bash
# PIDファイル削除
rm /tmp/soba.pid

# プロセス終了
kill -TERM $(pgrep soba)
```

### Git worktree管理

#### worktree確認
```bash
git worktree list
```

#### 不要worktree削除
```bash
# 自動クリーンアップ
soba cleanup

# 手動削除
git worktree remove .git/soba/worktrees/issue-123
```

## モニタリング

### メトリクス確認
```bash
# 処理統計
soba stats

# アクティブIssue
soba status --active

# 完了Issue
soba status --completed
```

### ヘルスチェック
```bash
# デーモン生存確認
soba health

# GitHub API接続確認
soba test-connection
```

## 設定チューニング

### パフォーマンス設定
```yaml
workflow:
  interval: 10  # ポーリング間隔(秒)
  max_parallel: 3  # 最大並行数
  timeout: 3600  # タイムアウト(秒)
```

### tmux設定
```yaml
workflow:
  use_tmux: true
  tmux_command_delay: 3  # コマンド間の遅延(秒)
```

### 自動化設定
```yaml
workflow:
  auto_merge_enabled: true
  closed_issue_cleanup_enabled: true
  closed_issue_cleanup_interval: 300  # 秒
```

## セキュリティ

### トークン管理
- 環境変数使用を推奨
- 設定ファイルは`.gitignore`に追加
- 権限は必要最小限に設定

### アクセス制御
```bash
# 設定ファイル権限
chmod 600 .soba/config.yml

# ログファイル権限
chmod 600 /tmp/soba.log
```

## バックアップとリカバリ

### 設定バックアップ
```bash
cp -r .soba .soba.backup
```

### 状態リストア
```bash
# 状態ファイルバックアップ
cp /tmp/soba_state.json /tmp/soba_state.json.backup

# リストア
cp /tmp/soba_state.json.backup /tmp/soba_state.json
soba start
```