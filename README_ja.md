# Soba - AI駆動開発ワークフロー自動化ツール

[![Go Version](https://img.shields.io/badge/go-1.23-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

> **For English README, see [here](README.md)**

SobaはGitHub Issueを完全な本番環境対応の実装に変換する、革新的なAI駆動開発自動化ツールです。完全自律的なワークフローを通じて開発プロセスを変革します。

## 🎯 Sobaとは？

Sobaは**24時間365日稼働する自律的な開発サイクル**を構築します：
- **GitHub Issue**が自動的に**Pull Request**になる
- **AIエージェント**が実装、テスト、レビューを担当
- 日常的な開発タスクに**人間の介入は不要**
- **tmux統合**によりリアルタイムでワークフローを可視化

### 主要メリット

- 🚀 Issue解決時間を**90%削減**
- 🤖 **完全自律**の開発サイクル
- 📊 AIレビューによる**一貫したコード品質**
- 🔄 **24時間365日継続**する開発ワークフロー
- 👀 tmuxセッション監視による**完全な透明性**

## 🏗️ アーキテクチャ

```
GitHub Issue → AI企画 → 実装 → テスト → レビュー → マージ
     ↓          ↓      ↓     ↓       ↓       ↓
  [soba:todo] → [soba:ready] → [soba:doing] → [soba:review] → [closed]
```

各フェーズはClaude Code AIによる完全自動処理：
- **企画**: 要件分析と実装戦略
- **実装**: コード生成とファイル修正
- **テスト**: 自動テスト実行と検証
- **レビュー**: AI駆動のコードレビューと品質保証

## 🚀 クイックスタート

### 前提条件

- **Go 1.23+**
- **Git 2.0+**
- **tmux 2.0+** (セッション管理用)
- **GitHub CLI** (推奨) またはGitHubトークン
- **Claude Code** インストール・設定済み

### インストール

#### クイックインストール（推奨）

```bash
# 最新リリースをダウンロード・インストール
curl -L https://github.com/douhashi/osoba/releases/latest/download/soba_$(uname -s | tr '[:upper:]' '[:lower:]')_$(uname -m | sed 's/x86_64/x86_64/; s/aarch64/arm64/').tar.gz | tar xz -C /tmp && sudo mv /tmp/soba /usr/local/bin/
```

#### その他のインストール方法

```bash
# ソースからビルド
git clone https://github.com/douhashi/soba.git
cd soba
go build -o soba cmd/soba/main.go

# またはGoでインストール
go install github.com/douhashi/soba/cmd/soba@latest
```

### 初期設定

```bash
# 設定ファイル初期化
soba init

# GitHub認証設定（推奨）
gh auth login

# または環境変数で設定
export GITHUB_TOKEN="ghp_xxxxxxxxxxxx"

# デーモン起動
soba start
```

## 📋 使用方法

### 基本ワークフロー

1. **GitHub Issue作成** - 明確な要件を記載
2. **`soba:todo`ラベル追加** - 自動化トリガー
3. **進捗監視** - tmuxセッションやGitHub更新を通じて
4. **Pull Requestレビュー** (オプション - 完全自動化可能)

### CLIコマンド

```bash
# デーモン起動（デフォルト: バックグラウンドモード）
soba start

# フォアグラウンドで詳細ログ付き起動
soba start -f --verbose

# デーモン状態確認
soba status

# デーモン停止
soba stop

# アクティブなtmuxセッション表示
soba sessions

# 特定のIssueセッションを開く
soba open issue-123-feature

# 設定表示
soba config

# 完了したworktreeのクリーンアップ
soba cleanup
```

### ラベルベース状態管理

SobaはGitHubラベルでIssueライフサイクルを追跡：

- `soba:todo` - 処理準備完了
- `soba:ready` - 企画フェーズ
- `soba:doing` - 実装進行中
- `soba:review` - AIレビュー中
- `soba:done` - 実装完了、マージ準備完了

## ⚙️ 設定

### 設定ファイル

Sobaは`.soba/config.yml`で設定：

```yaml
github:
  repository: "owner/repo"
  auth_method: "gh_cli"  # または "token"
  token: "${GITHUB_TOKEN}"

workflow:
  interval: 10           # ポーリング間隔（秒）
  max_parallel: 3        # 最大並行Issue数
  timeout: 3600          # Issue毎のタイムアウト（秒）
  auto_merge_enabled: true

tmux:
  use_tmux: true
  command_delay: 3       # コマンド間の遅延（秒）

logging:
  level: "info"
  format: "json"
```

### 環境変数

```bash
# GitHub認証
export GITHUB_TOKEN="ghp_xxxxxxxxxxxx"

# ログ設定
export SOBA_LOG_LEVEL="debug"
export SOBA_LOG_FORMAT="json"
```

## 🔧 高度な使用方法

### カスタムIssueテンプレート

AI理解向上のための構造化テンプレートでIssue作成：

```markdown
## 概要
機能/バグの簡潔な説明

## 要件
- 具体的な要件1
- 具体的な要件2

## 完了条件
- [ ] テストAが通る
- [ ] ドキュメント更新済み
- [ ] 破壊的変更なし

## 実装メモ
- 既存パターンXを使用
- パフォーマンスへの影響を考慮
```

### 監視とデバッグ

```bash
# デーモンログ確認
tail -f /tmp/soba.log

# 特定Issue進捗監視
tmux attach -t soba-issue-123-feature

# GitHub API接続確認
soba test-connection

# 処理統計表示
soba stats
```

### バッチ処理

複数Issueの同時処理：

```bash
# 複数Issueにラベル追加
gh issue edit 123 124 125 --add-label "soba:todo"

# すべてのアクティブセッション監視
tmux list-sessions | grep soba
```

## 🛠️ 開発

### ソースからビルド

```bash
git clone https://github.com/douhashi/soba.git
cd soba
go mod download
go build -o soba cmd/soba/main.go
```

### テスト実行

```bash
# 単体テスト実行
go test ./...

# 統合テスト実行
go test ./... -tags=integration

# カバレッジ付きテスト
go test -cover ./...
```

### プロジェクト構造

```
soba/
├── cmd/soba/           # メインアプリケーションエントリーポイント
├── internal/
│   ├── cli/            # CLIコマンドとインターフェース
│   ├── config/         # 設定管理
│   ├── domain/         # コアビジネスロジック
│   ├── infra/          # 外部システム統合
│   │   ├── github/     # GitHub APIクライアント
│   │   ├── tmux/       # tmuxセッション管理
│   │   └── slack/      # Slack通知
│   └── service/        # アプリケーションサービス
├── docs/               # ドキュメント
└── .soba/             # 設定テンプレート
```

## 🔍 トラブルシューティング

### よくある問題

**Issue処理が開始されない:**
```bash
# ラベル確認
gh issue view 123 --json labels

# デーモン状態確認
soba status

# ログ確認
tail -f /tmp/soba.log
```

**tmuxセッション問題:**
```bash
# セッション一覧
tmux list-sessions

# 停止したセッション削除
tmux kill-session -t soba-issue-123-feature

# デーモン再起動
soba stop && soba start
```

**Git worktree問題:**
```bash
# worktree一覧
git worktree list

# 自動クリーンアップ
soba cleanup

# 手動クリーンアップ
git worktree remove .git/soba/worktrees/issue-123
```

### パフォーマンスチューニング

高ボリュームリポジトリ向け：

```yaml
workflow:
  interval: 5           # より高速なポーリング
  max_parallel: 5       # より多くの並行Issue
  timeout: 7200         # 複雑なIssue用の長いタイムアウト
```

## 🤝 貢献

貢献を歓迎します！詳細は[貢献ガイド](CONTRIBUTING.md)をご覧ください。

### 開発環境セットアップ

1. リポジトリをフォーク
2. 機能ブランチを作成
3. 変更を実装
4. 新機能用のテストを追加
5. プルリクエストを送信

### コーディング標準

- Go慣例とイディオムに従う
- 包括的なテストを作成
- 新機能のドキュメントを更新
- 構造化ログを使用
- エラーハンドリングを適切に行う

## 📄 ライセンス

このプロジェクトはMITライセンス下にあります - 詳細は[LICENSE](LICENSE)ファイルをご覧ください。

## 🙏 謝辞

- [soba-cli (Ruby版)](https://github.com/douhashi/soba-cli)の基盤の上に構築
- AI駆動開発のため[Claude Code](https://claude.ai/code)を使用
- CLIフレームワークに[Cobra](https://github.com/spf13/cobra)を使用
- 設定管理に[Viper](https://github.com/spf13/viper)を使用

## 📞 サポート

- 📚 **ドキュメント**: `docs/`ディレクトリを確認
- 🐛 **Issues**: [GitHub Issues](https://github.com/douhashi/soba/issues)
- 💬 **ディスカッション**: [GitHub Discussions](https://github.com/douhashi/soba/discussions)

---

**Soba** - 自律的AIワークフローによるソフトウェア開発の変革。