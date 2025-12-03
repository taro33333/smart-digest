// Package llm provides abstractions for LLM providers.
package llm

import (
	"context"
	"fmt"

	"github.com/taro33333/smart-digest/internal/config"
)

// AnalysisResult represents the structured output from LLM analysis.
type AnalysisResult struct {
	Score    int      `json:"score"`
	Summary  []string `json:"summary"`
	Category string   `json:"category"`
}

// Provider defines the interface for LLM backends.
// This abstraction allows easy addition of new providers (Anthropic, Gemini, etc.)
type Provider interface {
	// Analyze sends article content to the LLM and returns analysis.
	Analyze(ctx context.Context, articleContent string, interests []string) (*AnalysisResult, error)

	// Name returns the provider name for logging.
	Name() string
}

// NewProvider creates an appropriate LLM provider based on configuration.
func NewProvider(cfg *config.Config) (Provider, error) {
	switch cfg.LLMProvider {
	case config.ProviderOpenAI:
		return NewOpenAIProvider(cfg.APIKey, cfg.Model)
	case config.ProviderOllama:
		return NewOllamaProvider(cfg.OllamaURL, cfg.Model)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.LLMProvider)
	}
}

// BuildSystemPrompt generates the system prompt for LLM analysis.
func BuildSystemPrompt(interests []string) string {
	interestList := ""
	for i, interest := range interests {
		if i > 0 {
			interestList += ", "
		}
		interestList += interest
	}

	return fmt.Sprintf(`あなたは優秀なエンジニアのアシスタントです。以下の記事本文を読み、ユーザーの興味関心領域: [%s] に基づいて 0〜100点でスコアリングし、日本語で要約してください。

## スコアリング基準
- 90-100: 興味関心に直接関連し、実務で即座に活用できる内容
- 70-89: 興味関心に関連があり、参考になる内容
- 50-69: 間接的に関連があるかもしれない内容
- 30-49: 関連性が薄い内容
- 0-29: 興味関心とほぼ無関係

## 出力形式
必ず以下のJSON形式のみで出力してください。他の文章は一切含めないでください。

{
  "score": <0-100の整数>,
  "summary": [
    "<要点1: 1文で簡潔に>",
    "<要点2: 1文で簡潔に>",
    "<要点3: 1文で簡潔に>"
  ],
  "category": "<最も適切な1つのカテゴリタグ>"
}`, interestList)
}

// BuildUserPrompt generates the user prompt with article content.
func BuildUserPrompt(articleContent string) string {
	return fmt.Sprintf(`以下の記事を分析してください:

---
%s
---`, articleContent)
}
