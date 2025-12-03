package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAIProvider implements Provider interface for OpenAI API.
type OpenAIProvider struct {
	client *openai.Client
	model  string
}

// NewOpenAIProvider creates a new OpenAI provider.
func NewOpenAIProvider(apiKey, model string) (*OpenAIProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	client := openai.NewClient(apiKey)
	return &OpenAIProvider{
		client: client,
		model:  model,
	}, nil
}

// Name returns the provider name.
func (p *OpenAIProvider) Name() string {
	return "OpenAI"
}

// Analyze sends content to OpenAI and returns structured analysis.
func (p *OpenAIProvider) Analyze(ctx context.Context, articleContent string, interests []string) (*AnalysisResult, error) {
	systemPrompt := BuildSystemPrompt(interests)
	userPrompt := BuildUserPrompt(articleContent)

	resp, err := p.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: p.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userPrompt,
				},
			},
			Temperature: 0.3, // Lower temperature for consistent JSON output
			MaxTokens:   500,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	content := resp.Choices[0].Message.Content
	return parseAnalysisResult(content)
}

// parseAnalysisResult extracts JSON from LLM response.
func parseAnalysisResult(content string) (*AnalysisResult, error) {
	// Clean up response - sometimes LLMs wrap JSON in markdown code blocks
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var result AnalysisResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w\nResponse was: %s", err, content)
	}

	// Validate result
	if result.Score < 0 {
		result.Score = 0
	}
	if result.Score > 100 {
		result.Score = 100
	}

	if len(result.Summary) == 0 {
		return nil, fmt.Errorf("LLM returned empty summary")
	}

	if result.Category == "" {
		result.Category = "未分類"
	}

	return &result, nil
}
