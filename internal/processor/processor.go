// Package processor handles concurrent URL processing with worker pools.
package processor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/taro33333/smart-digest/internal/fetcher"
	"github.com/taro33333/smart-digest/internal/llm"
)

// Job represents a single URL processing job.
type Job struct {
	URL     string
	Project string
	Version string
}

// Result represents the processed output for a URL.
type Result struct {
	Job      Job
	Article  *fetcher.Article
	Analysis *llm.AnalysisResult
	Error    error
}

// Processor handles concurrent URL processing.
type Processor struct {
	fetcher       *fetcher.Fetcher
	llmProvider   llm.Provider
	interests     []string
	maxWorkers    int
	rateLimitTick time.Duration
}

// New creates a new Processor with the given configuration.
func New(f *fetcher.Fetcher, provider llm.Provider, interests []string, maxWorkers int, rateLimit float64) *Processor {
	// Calculate rate limit interval
	// e.g., rateLimit=0.05 means 1 request per 20 seconds (3 RPM)
	var tickDuration time.Duration
	if rateLimit >= 1 {
		tickDuration = time.Second / time.Duration(rateLimit)
	} else {
		// For rates < 1 per second, calculate interval in seconds
		tickDuration = time.Duration(float64(time.Second) / rateLimit)
	}

	return &Processor{
		fetcher:       f,
		llmProvider:   provider,
		interests:     interests,
		maxWorkers:    maxWorkers,
		rateLimitTick: tickDuration,
	}
}

// ProcessCallback is called for each processed result (for progress updates).
type ProcessCallback func(completed, total int, result *Result)

// Process handles multiple URLs concurrently and returns results.
func (p *Processor) Process(ctx context.Context, jobs []Job, callback ProcessCallback) []Result {
	if len(jobs) == 0 {
		return nil
	}

	// Create channels
	jobChan := make(chan Job, len(jobs))
	resultChan := make(chan Result, len(jobs))

	// Rate limiter ticker
	rateLimiter := time.NewTicker(p.rateLimitTick)
	defer rateLimiter.Stop()

	// Create a rate-limited job channel
	rateLimitedJobs := make(chan Job, p.maxWorkers)

	// Start rate limiter goroutine
	go func() {
		defer close(rateLimitedJobs)
		for job := range jobChan {
			select {
			case <-ctx.Done():
				return
			case <-rateLimiter.C:
				rateLimitedJobs <- job
			}
		}
	}()

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < p.maxWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			p.worker(ctx, workerID, rateLimitedJobs, resultChan)
		}(i)
	}

	// Send jobs
	go func() {
		for _, job := range jobs {
			select {
			case jobChan <- job:
			case <-ctx.Done():
				close(jobChan)
				return
			}
		}
		close(jobChan)
	}()

	// Close results when all workers are done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var results []Result
	completed := 0
	for result := range resultChan {
		completed++
		results = append(results, result)
		if callback != nil {
			callback(completed, len(jobs), &result)
		}
	}

	return results
}

// worker processes jobs from the channel.
func (p *Processor) worker(ctx context.Context, id int, jobs <-chan Job, results chan<- Result) {
	for job := range jobs {
		select {
		case <-ctx.Done():
			results <- Result{
				Job:   job,
				Error: ctx.Err(),
			}
			return
		default:
		}

		result := p.processJob(ctx, job)
		results <- result
	}
}

// processJob handles a single URL processing.
func (p *Processor) processJob(ctx context.Context, job Job) Result {
	result := Result{Job: job}

	// Step 1: Fetch and extract content
	article, err := p.fetcher.Fetch(ctx, job.URL)
	if err != nil {
		result.Error = fmt.Errorf("fetch failed: %w", err)
		return result
	}
	result.Article = article

	// Step 2: Analyze with LLM
	analysis, err := p.llmProvider.Analyze(ctx, article.Content, p.interests)
	if err != nil {
		result.Error = fmt.Errorf("analysis failed: %w", err)
		return result
	}
	result.Analysis = analysis

	return result
}
