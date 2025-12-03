# 🧠 smart-digest

AI-powered digest generator for technical articles and release notes.

大量の技術記事やリリースノートを AI で分析し、自分の興味に関連するものだけを 3 行で要約してレポート出力する CLI ツールです。

## ✨ Features

- **AI による関連度スコアリング**: 設定した興味領域に基づいて 0-100 点で評価
- **3 行要約**: 記事の要点を日本語で簡潔に要約
- **複数 LLM 対応**: OpenAI API と Ollama (ローカル LLM) に対応
- **高速並行処理**: Worker Pool による効率的な処理
- **Unix 哲学準拠**: stdin/stdout によるパイプライン連携

## 🚀 Installation

```bash
# Clone repository
git clone https://github.com/taro33333/smart-digest.git
cd smart-digest

# Install dependencies
go mod tidy

# Build
go build -o smart-digest ./cmd/smart-digest

# (Optional) Install to PATH
go install ./cmd/smart-digest
```

## ⚙️ Configuration

設定ファイルを作成します：

```bash
# Copy example config
cp config.example.yaml config.yaml

# Or create in XDG config directory
mkdir -p ~/.config/smart-digest
cp config.example.yaml ~/.config/smart-digest/config.yaml
```

### 設定項目

| 項目 | 説明 | デフォルト |
|------|------|-----------|
| `llm_provider` | LLM プロバイダ (`openai` or `ollama`) | `openai` |
| `api_key` | OpenAI API キー (環境変数 `OPENAI_API_KEY` も可) | - |
| `model` | 使用モデル | `gpt-4o-mini` |
| `ollama_url` | Ollama サーバー URL | `http://localhost:11434` |
| `interests` | 興味領域のリスト | - |
| `threshold` | 出力する最低スコア (0-100) | `70` |
| `max_workers` | 並列ワーカー数 | `5` |
| `rate_limit_per_second` | 秒間 API コール数上限 | `10` |

### 興味領域の設定例

```yaml
interests:
  - "Go"
  - "Rust"
  - "System Design"
  - "Productivity"
  - "DevOps"
  - "Kubernetes"
  - "Performance Optimization"
```

## 📖 Usage

### 単一 URL の分析

```bash
smart-digest --url "https://go.dev/blog/go1.21"
```

### 複数 URL をパイプで入力

```bash
# JSON Lines 形式
echo '{"url":"https://example.com/article1"}
{"url":"https://example.com/article2"}' | smart-digest

# JSON Array 形式
echo '[{"url":"https://example.com/article1"},{"url":"https://example.com/article2"}]' | smart-digest
```

### update-watcher との連携

```bash
update-watcher | smart-digest
```

### CLI オプション

```bash
smart-digest --help

Flags:
  -c, --config string     Path to config file
  -f, --format string     Output format (markdown, json) (default "markdown")
  -h, --help              help for smart-digest
  -t, --threshold int     Override score threshold (0-100) (default -1)
  -u, --url string        URL to analyze
  -v, --verbose           Verbose output
  -w, --workers int       Override max workers (default -1)
      --version           version for smart-digest
```

## 📤 Output Format

### Markdown (default)

```markdown
# Smart Digest Report

_Generated: 2024-01-15 10:30_

**閾値:** 70点以上 | **処理数:** 5件 | **該当:** 3件

---

## 1. 🔥 Go 1.21 Release Notes

**URL:** https://go.dev/blog/go1.21

**スコア:** 95/100 | **カテゴリ:** `Go`

### 要約

- Go 1.21 では新しい組み込み関数 min, max, clear が追加された
- Profile Guided Optimization (PGO) が正式にサポートされ、パフォーマンスが向上
- 新しい log/slog パッケージにより構造化ロギングが標準で利用可能に
```

### JSON

```bash
smart-digest --format json --url "https://example.com"
```

```json
[
  {
    "url": "https://go.dev/blog/go1.21",
    "title": "Go 1.21 Release Notes",
    "score": 95,
    "category": "Go",
    "summary": "Go 1.21 では新しい組み込み関数 min, max, clear が追加された / ..."
  }
]
```

## 🔧 Development

### Project Structure

```
smart-digest/
├── cmd/
│   └── smart-digest/
│       └── main.go          # CLI entry point
├── internal/
│   ├── config/
│   │   └── config.go        # Configuration management
│   ├── fetcher/
│   │   └── fetcher.go       # URL fetching & content extraction
│   ├── input/
│   │   └── parser.go        # Input parsing (stdin/args)
│   ├── llm/
│   │   ├── interface.go     # LLM provider interface
│   │   ├── openai.go        # OpenAI implementation
│   │   └── ollama.go        # Ollama implementation
│   ├── output/
│   │   └── formatter.go     # Output formatting
│   └── processor/
│       └── processor.go     # Concurrent processing
├── config.example.yaml
├── go.mod
└── README.md
```

### Adding a New LLM Provider

1. `internal/llm/` に新しいファイルを作成
2. `Provider` インターフェースを実装
3. `internal/llm/interface.go` の `NewProvider` に追加

```go
// Example: Adding Anthropic support
type AnthropicProvider struct {
    client *anthropic.Client
    model  string
}

func (p *AnthropicProvider) Analyze(ctx context.Context, content string, interests []string) (*AnalysisResult, error) {
    // Implementation
}

func (p *AnthropicProvider) Name() string {
    return "Anthropic"
}
```

### Running Tests

```bash
go test ./...
```

### Building

```bash
# Development build
go build -o smart-digest ./cmd/smart-digest

# Release build with optimizations
go build -ldflags="-s -w" -o smart-digest ./cmd/smart-digest
```

## 🔗 Integration Examples

### With cron

```bash
# Daily digest at 9 AM
0 9 * * * update-watcher | smart-digest >> ~/digest-$(date +\%Y-\%m-\%d).md
```

### With notification

```bash
update-watcher | smart-digest | mail -s "Daily Tech Digest" you@example.com
```

## 📝 License

MIT License

## 🤝 Contributing

Issues and Pull Requests are welcome!
