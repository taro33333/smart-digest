// Package main provides the CLI entry point for smart-digest.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"

	"github.com/taro33333/smart-digest/internal/config"
	"github.com/taro33333/smart-digest/internal/fetcher"
	"github.com/taro33333/smart-digest/internal/input"
	"github.com/taro33333/smart-digest/internal/llm"
	"github.com/taro33333/smart-digest/internal/output"
	"github.com/taro33333/smart-digest/internal/processor"
)

var (
	version = "0.1.0"

	// CLI flags
	urlFlag        string
	configPath     string
	outputFormat   string
	thresholdFlag  int
	verboseFlag    bool
	maxWorkersFlag int
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "smart-digest [flags]",
	Short: "AI-powered digest generator for technical articles",
	Long: `smart-digest analyzes URLs and generates AI-powered summaries
based on your interests. It supports both CLI arguments and stdin for
integration with other tools.

Examples:
  # Single URL
  smart-digest --url "https://example.com/blog/post"

  # Multiple URLs from pipe
  echo '{"url":"https://example.com"}' | smart-digest

  # Integration with update-watcher
  update-watcher | smart-digest`,
	Version: version,
	RunE:    run,
}

func init() {
	rootCmd.Flags().StringVarP(&urlFlag, "url", "u", "", "URL to analyze")
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to config file")
	rootCmd.Flags().StringVarP(&outputFormat, "format", "f", "markdown", "Output format (markdown, json)")
	rootCmd.Flags().IntVarP(&thresholdFlag, "threshold", "t", -1, "Override score threshold (0-100)")
	rootCmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "Verbose output")
	rootCmd.Flags().IntVarP(&maxWorkersFlag, "workers", "w", -1, "Override max workers")
}

func run(cmd *cobra.Command, args []string) error {
	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Fprintln(os.Stderr, "\n⚠️  Interrupted, shutting down...")
		cancel()
	}()

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Override config with CLI flags
	if thresholdFlag >= 0 {
		cfg.Threshold = thresholdFlag
	}
	if maxWorkersFlag > 0 {
		cfg.MaxWorkers = maxWorkersFlag
	}

	// Collect jobs from input
	jobs, err := collectJobs(args)
	if err != nil {
		return fmt.Errorf("input error: %w", err)
	}

	if len(jobs) == 0 {
		return fmt.Errorf("no URLs provided. Use --url flag or pipe JSON to stdin")
	}

	if verboseFlag {
		fmt.Fprintf(os.Stderr, "📋 Processing %d URLs...\n", len(jobs))
		fmt.Fprintf(os.Stderr, "🤖 LLM: %s (%s)\n", cfg.LLMProvider, cfg.Model)
		fmt.Fprintf(os.Stderr, "🎯 Interests: %s\n", cfg.InterestsString())
		fmt.Fprintf(os.Stderr, "📊 Threshold: %d\n\n", cfg.Threshold)
	}

	// Initialize components
	f := fetcher.New()

	provider, err := llm.NewProvider(cfg)
	if err != nil {
		return fmt.Errorf("LLM initialization error: %w", err)
	}

	proc := processor.New(f, provider, cfg.Interests, cfg.MaxWorkers, cfg.RateLimit)
	formatter := output.New(cfg.Threshold)

	// Create progress bar
	var bar *progressbar.ProgressBar
	if !verboseFlag {
		bar = progressbar.NewOptions(len(jobs),
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionEnableColorCodes(true),
			progressbar.OptionShowCount(),
			progressbar.OptionShowElapsedTimeOnFinish(),
			progressbar.OptionSetWidth(40),
			progressbar.OptionSetDescription("[cyan]Processing...[reset]"),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "[green]=[reset]",
				SaucerHead:    "[green]>[reset]",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}),
		)
	}

	// Process URLs
	callback := func(completed, total int, result *processor.Result) {
		if bar != nil {
			bar.Add(1)
		}
		if verboseFlag {
			status := "✅"
			if result.Error != nil {
				status = "❌"
			}
			fmt.Fprintf(os.Stderr, "%s [%d/%d] %s\n", status, completed, total, result.Job.URL)
		}
	}

	results := proc.Process(ctx, jobs, callback)

	// Finish progress bar
	if bar != nil {
		bar.Finish()
		fmt.Fprintln(os.Stderr)
	}

	// Output results
	switch outputFormat {
	case "json":
		return formatter.FormatJSON(os.Stdout, results)
	default:
		return formatter.FormatMarkdown(os.Stdout, results)
	}
}

// collectJobs gathers URLs from all input sources.
func collectJobs(args []string) ([]processor.Job, error) {
	var jobs []processor.Job
	parser := input.New()

	// From --url flag
	if urlFlag != "" {
		jobs = append(jobs, processor.Job{URL: urlFlag})
	}

	// From positional args
	if len(args) > 0 {
		jobs = append(jobs, parser.ParseArgs(args)...)
	}

	// From stdin (if not a terminal)
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		stdinJobs, err := parser.ParseStdin(os.Stdin)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, stdinJobs...)
	}

	return jobs, nil
}
