# Soba - AI-Driven Development Workflow Automation

[![Go Version](https://img.shields.io/badge/go-1.23-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

> **日本語版のREADMEは [こちら](README_ja.md) をご覧ください**

Soba is a revolutionary AI-powered development automation tool that transforms GitHub Issues into complete, production-ready implementations through fully autonomous workflows.

## 🎯 What is Soba?

Soba creates a **24/7 autonomous development cycle** where:
- **GitHub Issues** automatically become **Pull Requests**
- **AI agents** handle implementation, testing, and review
- **Zero human intervention** required for routine development tasks
- **tmux integration** provides real-time workflow visibility

### Key Benefits

- 🚀 **90% reduction** in issue resolution time
- 🤖 **Fully autonomous** development cycle
- 📊 **Consistent code quality** through AI review
- 🔄 **24/7 continuous** development workflow
- 👀 **Full transparency** via tmux session monitoring

## 🏗️ Architecture

```
GitHub Issue → AI Planning → Implementation → Testing → Review → Merge
     ↓             ↓             ↓           ↓         ↓       ↓
  [soba:todo] → [soba:ready] → [soba:doing] → [soba:review] → [closed]
```

Each phase is handled by Claude Code AI with full automation:
- **Planning**: Requirements analysis and implementation strategy
- **Implementation**: Code generation and file modifications
- **Testing**: Automated test execution and validation
- **Review**: AI-powered code review and quality assurance

## 🚀 Quick Start

### Prerequisites

- **Go 1.23+**
- **Git 2.0+**
- **tmux 2.0+** (for session management)
- **GitHub CLI** (recommended) or GitHub token
- **Claude Code** installed and configured

### Installation

#### Quick Install (Recommended)

```bash
# Download and install the latest release
curl -L https://github.com/douhashi/osoba/releases/latest/download/soba_$(uname -s | tr '[:upper:]' '[:lower:]')_$(uname -m | sed 's/x86_64/x86_64/; s/aarch64/arm64/').tar.gz | tar xz -C /tmp && sudo mv /tmp/soba /usr/local/bin/
```

#### Alternative Installation Methods

```bash
# Build from source
git clone https://github.com/douhashi/soba.git
cd soba
go build -o soba cmd/soba/main.go

# Or install with Go
go install github.com/douhashi/soba/cmd/soba@latest
```

### Initial Setup

```bash
# Initialize configuration
soba init

# Configure GitHub authentication (recommended)
gh auth login

# Or set environment variable
export GITHUB_TOKEN="ghp_xxxxxxxxxxxx"

# Start the daemon
soba start
```

## 📋 Usage

### Basic Workflow

1. **Create a GitHub Issue** with clear requirements
2. **Add the `soba:todo` label** to trigger automation
3. **Monitor progress** through tmux sessions or GitHub updates
4. **Review the Pull Request** (optional - can be fully automated)

### CLI Commands

```bash
# Start the daemon (default: background mode)
soba start

# Start in foreground with verbose logging
soba start -f --verbose

# Check daemon status
soba status

# Stop the daemon
soba stop

# View active tmux sessions
soba sessions

# Open specific issue session
soba open issue-123-feature

# Display configuration
soba config

# Clean up completed worktrees
soba cleanup
```

### Label-Based State Management

Soba uses GitHub labels to track issue lifecycle:

- `soba:todo` - Ready for processing
- `soba:ready` - Planning phase
- `soba:doing` - Implementation in progress
- `soba:review` - Under AI review
- `soba:done` - Implementation complete, ready for merge

## ⚙️ Configuration

### Configuration File

Soba uses `.soba/config.yml` for configuration:

```yaml
github:
  repository: "owner/repo"
  auth_method: "gh_cli"  # or "token"
  token: "${GITHUB_TOKEN}"

workflow:
  interval: 10           # Polling interval (seconds)
  max_parallel: 3        # Maximum parallel issues
  timeout: 3600          # Timeout per issue (seconds)
  auto_merge_enabled: true

tmux:
  use_tmux: true
  command_delay: 3       # Delay between commands (seconds)

logging:
  level: "info"
  format: "json"
```

### Environment Variables

```bash
# GitHub authentication
export GITHUB_TOKEN="ghp_xxxxxxxxxxxx"

# Logging configuration
export SOBA_LOG_LEVEL="debug"
export SOBA_LOG_FORMAT="json"
```

## 🔧 Advanced Usage

### Custom Issue Templates

Create issues with structured templates for better AI understanding:

```markdown
## Overview
Brief description of the feature/bug

## Requirements
- Specific requirement 1
- Specific requirement 2

## Acceptance Criteria
- [ ] Test A passes
- [ ] Documentation updated
- [ ] No breaking changes

## Implementation Notes
- Use existing pattern X
- Consider performance implications
```

### Monitoring and Debugging

```bash
# View daemon logs
tail -f /tmp/soba.log

# Monitor specific issue progress
tmux attach -t soba-issue-123-feature

# Check GitHub API connectivity
soba test-connection

# View processing statistics
soba stats
```

### Batch Processing

Process multiple issues simultaneously:

```bash
# Add labels to multiple issues
gh issue edit 123 124 125 --add-label "soba:todo"

# Monitor all active sessions
tmux list-sessions | grep soba
```

## 🛠️ Development

### Building from Source

```bash
git clone https://github.com/douhashi/soba.git
cd soba
go mod download
go build -o soba cmd/soba/main.go
```

### Running Tests

```bash
# Run unit tests
go test ./...

# Run integration tests
go test ./... -tags=integration

# Run with coverage
go test -cover ./...
```

### Project Structure

```
soba/
├── cmd/soba/           # Main application entry point
├── internal/
│   ├── cli/            # CLI commands and interface
│   ├── config/         # Configuration management
│   ├── domain/         # Core business logic
│   ├── infra/          # External integrations
│   │   ├── github/     # GitHub API client
│   │   ├── tmux/       # tmux session management
│   │   └── slack/      # Slack notifications
│   └── service/        # Application services
├── docs/               # Documentation
└── .soba/             # Configuration templates
```

## 🔍 Troubleshooting

### Common Issues

**Issue processing not starting:**
```bash
# Check labels
gh issue view 123 --json labels

# Verify daemon status
soba status

# Check logs
tail -f /tmp/soba.log
```

**tmux session issues:**
```bash
# List all sessions
tmux list-sessions

# Kill stuck session
tmux kill-session -t soba-issue-123-feature

# Restart daemon
soba stop && soba start
```

**Git worktree problems:**
```bash
# List worktrees
git worktree list

# Clean up automatically
soba cleanup

# Manual cleanup
git worktree remove .git/soba/worktrees/issue-123
```

### Performance Tuning

For high-volume repositories:

```yaml
workflow:
  interval: 5           # Faster polling
  max_parallel: 5       # More concurrent issues
  timeout: 7200         # Longer timeout for complex issues
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

### Code Standards

- Follow Go conventions and idioms
- Write comprehensive tests
- Update documentation for new features
- Use structured logging
- Handle errors gracefully

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built on the foundation of [soba-cli (Ruby version)](https://github.com/douhashi/soba-cli)
- Powered by [Claude Code](https://claude.ai/code) for AI-driven development
- Uses [Cobra](https://github.com/spf13/cobra) for CLI framework
- Configuration management by [Viper](https://github.com/spf13/viper)

## 📞 Support

- 📚 **Documentation**: Check the `docs/` directory
- 🐛 **Issues**: [GitHub Issues](https://github.com/douhashi/soba/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/douhashi/soba/discussions)

---

**Soba** - Transforming software development through autonomous AI workflows.