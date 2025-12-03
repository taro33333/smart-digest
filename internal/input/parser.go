// Package input handles parsing of input from stdin and CLI arguments.
package input

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/taro33333/smart-digest/internal/processor"
)

// JSONInput represents the expected JSON format from stdin.
type JSONInput struct {
	URL     string `json:"url"`
	Project string `json:"project"`
	Version string `json:"version"`
}

// Parser handles input parsing from various sources.
type Parser struct{}

// New creates a new Parser.
func New() *Parser {
	return &Parser{}
}

// ParseArgs creates jobs from CLI URL arguments.
func (p *Parser) ParseArgs(urls []string) []processor.Job {
	jobs := make([]processor.Job, 0, len(urls))
	for _, url := range urls {
		url = strings.TrimSpace(url)
		if url != "" {
			jobs = append(jobs, processor.Job{URL: url})
		}
	}
	return jobs
}

// ParseStdin reads and parses JSON input from stdin.
// Supports both JSON Lines format and JSON array format.
func (p *Parser) ParseStdin(r io.Reader) ([]processor.Job, error) {
	var jobs []processor.Job

	// Read all content first
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read stdin: %w", err)
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return nil, nil
	}

	// Try parsing as JSON array first
	if strings.HasPrefix(content, "[") {
		var inputs []JSONInput
		if err := json.Unmarshal([]byte(content), &inputs); err == nil {
			for _, input := range inputs {
				if input.URL != "" {
					jobs = append(jobs, processor.Job{
						URL:     input.URL,
						Project: input.Project,
						Version: input.Version,
					})
				}
			}
			return jobs, nil
		}
	}

	// Parse as JSON Lines (newline-delimited JSON)
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var input JSONInput
		if err := json.Unmarshal([]byte(line), &input); err != nil {
			// If it looks like a plain URL, use it directly
			if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
				jobs = append(jobs, processor.Job{URL: line})
				continue
			}
			return nil, fmt.Errorf("failed to parse line %d: %w", lineNum, err)
		}

		if input.URL != "" {
			jobs = append(jobs, processor.Job{
				URL:     input.URL,
				Project: input.Project,
				Version: input.Version,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading input: %w", err)
	}

	return jobs, nil
}
