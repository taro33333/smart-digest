// Package fetcher handles URL fetching and content extraction.
package fetcher

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	readability "github.com/go-shiori/go-readability"
)

// Article represents the extracted content from a URL.
type Article struct {
	URL     string
	Title   string
	Content string
	Excerpt string
}

// Fetcher handles HTTP requests and content extraction.
type Fetcher struct {
	client  *http.Client
	timeout time.Duration
}

// New creates a new Fetcher with sensible defaults.
func New() *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		timeout: 30 * time.Second,
	}
}

// Fetch retrieves and extracts clean content from a URL.
func (f *Fetcher) Fetch(ctx context.Context, targetURL string) (*Article, error) {
	// Validate URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %s: %w", targetURL, err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent to avoid being blocked
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; SmartDigest/1.0; +https://github.com/taro33333/smart-digest)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5,ja;q=0.3")

	// Execute request
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL %s: %w", targetURL, err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d for URL %s", resp.StatusCode, targetURL)
	}

	// Parse with readability
	article, err := readability.FromReader(resp.Body, parsedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse content from %s: %w", targetURL, err)
	}

	// Clean up extracted text
	content := cleanText(article.TextContent)

	if len(content) < 100 {
		return nil, fmt.Errorf("extracted content too short from %s (got %d chars)", targetURL, len(content))
	}

	// Truncate if too long (LLM context limit consideration)
	const maxChars = 15000
	if len(content) > maxChars {
		content = content[:maxChars] + "\n...[truncated]"
	}

	return &Article{
		URL:     targetURL,
		Title:   article.Title,
		Content: content,
		Excerpt: truncateString(article.Excerpt, 300),
	}, nil
}

// cleanText removes excessive whitespace and normalizes line breaks.
func cleanText(text string) string {
	// Replace multiple newlines with double newline
	lines := strings.Split(text, "\n")
	var cleaned []string
	prevEmpty := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if !prevEmpty {
				cleaned = append(cleaned, "")
				prevEmpty = true
			}
		} else {
			cleaned = append(cleaned, line)
			prevEmpty = false
		}
	}

	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

// truncateString truncates a string to maxLen with ellipsis.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
