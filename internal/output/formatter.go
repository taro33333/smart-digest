// Package output handles result formatting and output generation.
package output

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/taro33333/smart-digest/internal/processor"
)

// Formatter handles result formatting.
type Formatter struct {
	threshold int
}

// New creates a new Formatter with the given threshold.
func New(threshold int) *Formatter {
	return &Formatter{
		threshold: threshold,
	}
}

// FormatMarkdown generates Markdown output from results.
func (f *Formatter) FormatMarkdown(w io.Writer, results []processor.Result) error {
	// Filter and sort results
	var filtered []processor.Result
	var errors []processor.Result

	for _, r := range results {
		if r.Error != nil {
			errors = append(errors, r)
			continue
		}
		if r.Analysis != nil && r.Analysis.Score >= f.threshold {
			filtered = append(filtered, r)
		}
	}

	// Sort by score descending
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Analysis.Score > filtered[j].Analysis.Score
	})

	// Generate header
	fmt.Fprintf(w, "# Smart Digest Report\n\n")
	fmt.Fprintf(w, "_Generated: %s_\n\n", time.Now().Format("2006-01-02 15:04"))
	fmt.Fprintf(w, "**閾値:** %d点以上 | **処理数:** %d件 | **該当:** %d件\n\n",
		f.threshold, len(results), len(filtered))

	if len(filtered) == 0 {
		fmt.Fprintf(w, "> 該当する記事はありませんでした。\n\n")
	}

	// Generate entries
	fmt.Fprintf(w, "---\n\n")
	for i, r := range filtered {
		f.formatEntry(w, i+1, r)
	}

	// Error summary
	if len(errors) > 0 {
		fmt.Fprintf(w, "## ⚠️ エラー (%d件)\n\n", len(errors))
		for _, r := range errors {
			fmt.Fprintf(w, "- **%s**\n  - `%s`\n",
				r.Job.URL, r.Error.Error())
		}
		fmt.Fprintf(w, "\n")
	}

	return nil
}

// formatEntry formats a single result entry.
func (f *Formatter) formatEntry(w io.Writer, num int, r processor.Result) {
	title := r.Article.Title
	if title == "" {
		title = r.Job.URL
	}

	// Score emoji
	scoreEmoji := getScoreEmoji(r.Analysis.Score)

	fmt.Fprintf(w, "## %d. %s %s\n\n", num, scoreEmoji, title)
	fmt.Fprintf(w, "**URL:** %s\n\n", r.Job.URL)
	fmt.Fprintf(w, "**スコア:** %d/100 | **カテゴリ:** `%s`\n\n",
		r.Analysis.Score, r.Analysis.Category)

	// Version info if available
	if r.Job.Project != "" || r.Job.Version != "" {
		if r.Job.Project != "" && r.Job.Version != "" {
			fmt.Fprintf(w, "**プロジェクト:** %s v%s\n\n", r.Job.Project, r.Job.Version)
		} else if r.Job.Project != "" {
			fmt.Fprintf(w, "**プロジェクト:** %s\n\n", r.Job.Project)
		}
	}

	// Summary
	fmt.Fprintf(w, "### 要約\n\n")
	for _, point := range r.Analysis.Summary {
		fmt.Fprintf(w, "- %s\n", point)
	}
	fmt.Fprintf(w, "\n---\n\n")
}

// FormatJSON generates JSON output from results.
func (f *Formatter) FormatJSON(w io.Writer, results []processor.Result) error {
	var filtered []processor.Result
	for _, r := range results {
		if r.Error == nil && r.Analysis != nil && r.Analysis.Score >= f.threshold {
			filtered = append(filtered, r)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Analysis.Score > filtered[j].Analysis.Score
	})

	fmt.Fprintf(w, "[\n")
	for i, r := range filtered {
		summary := strings.Join(r.Analysis.Summary, " / ")
		fmt.Fprintf(w, "  {\n")
		fmt.Fprintf(w, "    \"url\": %q,\n", r.Job.URL)
		fmt.Fprintf(w, "    \"title\": %q,\n", r.Article.Title)
		fmt.Fprintf(w, "    \"score\": %d,\n", r.Analysis.Score)
		fmt.Fprintf(w, "    \"category\": %q,\n", r.Analysis.Category)
		fmt.Fprintf(w, "    \"summary\": %q\n", summary)
		if i < len(filtered)-1 {
			fmt.Fprintf(w, "  },\n")
		} else {
			fmt.Fprintf(w, "  }\n")
		}
	}
	fmt.Fprintf(w, "]\n")

	return nil
}

// getScoreEmoji returns an emoji based on score.
func getScoreEmoji(score int) string {
	switch {
	case score >= 90:
		return "🔥"
	case score >= 80:
		return "⭐"
	case score >= 70:
		return "📌"
	default:
		return "📄"
	}
}
