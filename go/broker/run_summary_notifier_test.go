package broker

import "testing"

type fakeEmbed struct {
	called      bool
	title, desc string
	color       int
}

func (f *fakeEmbed) SendEmbed(title, description string, color int) bool {
	f.called = true
	f.title = title
	f.desc = description
	f.color = color
	return true
}

func TestRunSummaryNotifier(t *testing.T) {
	fe := &fakeEmbed{}
	n := NewRunSummaryNotifier(fe)
	summary := JobRunSummary{RunID: "r1", Status: "COMPLETED", CompletedSymbols: 10, TotalSymbols: 10, FailedSymbols: 0}
	n.NotifyRun(summary)
	if fe.called {
		t.Fatal("clean run should not notify")
	}
	warn := JobRunSummary{RunID: "r2", Status: "COMPLETED_WITH_WARNINGS", DegradedStages: 2, DegradedBreakdown: []RunDegradedStage{{Stage: "SENTIMENT", Reason: "KEYWORD_FALLBACK", Count: 2}}, SkippedStrategies: 1, SkippedBreakdown: []RunSkippedStrategy{{VariantID: "pullback-v1", Outcome: "SKIPPED", Reason: "unsupported", Symbols: 10}}}
	n.NotifyRun(warn)
	if !fe.called {
		t.Fatal("warnings should notify")
	}
	if fe.color != ColorYellow {
		t.Fatal("yellow expected")
	}
	text := FormatRunSummary(warn)
	if !contains(text, "SENTIMENT (KEYWORD_FALLBACK): 2") {
		t.Fatal("degraded missing")
	}
	if !contains(text, "pullback-v1 skipped: unsupported (10 symbol(s))") {
		t.Fatalf("skipped missing: %s", text)
	}
	fail := JobRunSummary{RunID: "r3", Status: "FAILED"}
	n2 := NewRunSummaryNotifier(fe)
	fe.called = false
	n2.NotifyRun(fail)
	if !fe.called || fe.color != ColorRed {
		t.Fatal("failed color")
	}
	// nil discord
	NewRunSummaryNotifier(nil).NotifyRun(warn)
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	}())
}
