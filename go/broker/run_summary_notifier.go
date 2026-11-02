package broker

import (
	"fmt"
	"strings"
)

// RunDegradedStage is part of the broker notification port.
type RunDegradedStage struct {
	Stage  string
	Reason string
	Count  int
}

// RunSkippedStrategy is part of the broker notification port.
type RunSkippedStrategy struct {
	VariantID string
	Outcome   string
	Reason    string
	Symbols   int
}

// JobRunSummary is part of the broker notification port.
type JobRunSummary struct {
	RunID             string
	Status            string
	CompletedSymbols  int
	TotalSymbols      int
	FailedSymbols     int
	ElapsedMillis     int64
	DegradedStages    int
	SkippedStrategies int
	DegradedBreakdown []RunDegradedStage
	SkippedBreakdown  []RunSkippedStrategy
}

// RunSummaryDiscord is part of the broker notification port.
type RunSummaryDiscord interface {
	SendEmbed(title, description string, color int) bool
}

// RunSummaryNotifier is part of the broker notification port.
type RunSummaryNotifier struct {
	discord RunSummaryDiscord
}

// NewRunSummaryNotifier is part of the broker notification port.
func NewRunSummaryNotifier(discord RunSummaryDiscord) *RunSummaryNotifier {
	return &RunSummaryNotifier{discord: discord}
}

// ShouldNotifyRun is part of the broker notification port.
func ShouldNotifyRun(summary JobRunSummary) bool {
	return summary.Status == "COMPLETED_WITH_WARNINGS" || summary.Status == "FAILED" ||
		summary.DegradedStages > 0 || summary.SkippedStrategies > 0
}

// NotifyRun is part of the broker notification port.
func (n *RunSummaryNotifier) NotifyRun(summary JobRunSummary) {
	if summary.RunID == "" && summary.Status == "" {
		return
	}
	if !ShouldNotifyRun(summary) {
		return
	}
	color := ColorYellow
	if summary.Status == "FAILED" {
		color = ColorRed
	}
	if n.discord == nil {
		return
	}
	func() {
		defer func() { _ = recover() }()
		n.discord.SendEmbed("Job run "+summary.Status, FormatRunSummary(summary), color)
	}()
}

const maxRunDescription = 3500

// FormatRunSummary is part of the broker notification port.
func FormatRunSummary(summary JobRunSummary) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Run %s\n%d/%d symbols processed, %d failed\n", summary.RunID, summary.CompletedSymbols, summary.TotalSymbols, summary.FailedSymbols)
	if len(summary.DegradedBreakdown) > 0 {
		fmt.Fprintf(&b, "\nDegraded stages (%d):\n", summary.DegradedStages)
		for _, d := range summary.DegradedBreakdown {
			fmt.Fprintf(&b, "- %s (%s): %d\n", d.Stage, d.Reason, d.Count)
		}
	}
	if len(summary.SkippedBreakdown) > 0 {
		b.WriteString("\nSkipped strategies:\n")
		for _, s := range summary.SkippedBreakdown {
			fmt.Fprintf(&b, "- %s %s: %s (%d symbol(s))\n", s.VariantID, strings.ToLower(s.Outcome), s.Reason, s.Symbols)
		}
	}
	text := b.String()
	runes := []rune(text)
	if len(runes) > maxRunDescription {
		text = string(runes[:maxRunDescription]) + "..."
	}
	return text
}
