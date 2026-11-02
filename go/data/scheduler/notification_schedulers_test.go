package scheduler

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

type fakeAttr struct{ report StrategyReport }

func (f fakeAttr) Report(ctx context.Context, from, to time.Time) (StrategyReport, error) {
	r := f.report
	r.From = from
	r.To = to
	return r, nil
}

type fakeEmbed2 struct{ calls int; last string }

func (f *fakeEmbed2) SendEmbed(title, description string, color int) bool {
	f.calls++
	f.last = description
	return true
}

func TestWeeklyStrategyDigestScheduler(t *testing.T) {
	attr := fakeAttr{report: StrategyReport{Totals: StrategyReportTotals{Tournaments: 5, Executed: 2, Blocked: 1, SelectedPnL: "120.00"}, Variants: []StrategyReportVariant{{VariantID: "breakout-v1", StrategyType: "BREAKOUT", TimesSelected: 3, BuySignals: 2, ShadowTrades: 3, ShadowWinRate: ptr(50.0), ShadowPnL: "30.00"}}}}
	emb := &fakeEmbed2{}
	now := func() time.Time { return time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC) }
	s, err := NewWeeklyStrategyDigestScheduler(attr, emb, true, slog.Default(), now)
	if err != nil { t.Fatal(err) }
	s.SendWeeklyStrategyDigest(context.Background())
	if emb.calls != 1 { t.Fatalf("calls=%d", emb.calls) }
	text := FormatStrategyDigest(attr.report)
	if !contains(text, "5 tournaments") || !contains(text, "**breakout-v1**") { t.Fatalf("text=%s", text) }
	disabled, _ := NewWeeklyStrategyDigestScheduler(attr, emb, false, slog.Default(), now)
	disabled.SendWeeklyStrategyDigest(context.Background())
	if emb.calls != 1 { t.Fatal("disabled should not send") }
	_ = NextWeeklyStrategyDigestRun(now())
}

func TestWeeklySectorDigestScheduler(t *testing.T) {
	type prov struct{ msg string }
	// use closure provider
	provider := sectorProvider{msg: "sector digest"}
	emb := &fakeEmbed2{}
	now := func() time.Time { return time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC) }
	s, err := NewWeeklySectorDigestScheduler(provider, emb, true, slog.Default(), now)
	if err != nil { t.Fatal(err) }
	s.SendWeeklySectorDigest(context.Background())
	if emb.calls != 1 || emb.last != "sector digest" { t.Fatalf("calls=%d last=%s", emb.calls, emb.last) }
	disabled, _ := NewWeeklySectorDigestScheduler(provider, emb, false, slog.Default(), now)
	disabled.SendWeeklySectorDigest(context.Background())
	_ = NextWeeklySectorDigestRun(now())
}

type sectorProvider struct{ msg string }

func (s sectorProvider) GenerateSectorDigestForLastWeek() string { return s.msg }

func TestMonthlyReportScheduler(t *testing.T) {
	ps := &posStore{positions: []MonthlyPosition{{Status: "TARGET_HIT", RealizedPnL: ptrF(10)}}}
	ss := &sigStore{signals: []MonthlySignal{{Type: "BUY"}, {Type: "BUY"}}}
	now := func() time.Time { return time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC) }
	s, err := NewMonthlyReportScheduler(ps, ss, true, slog.Default(), now)
	if err != nil { t.Fatal(err) }
	s.GenerateMonthlyReport()
	report := BuildMonthlyReport(now(), ps.positions, ss.signals)
	if report["totalSignals"].(int) != 2 { t.Fatalf("signals=%v", report["totalSignals"]) }
	if !contains(FormatMonthlyReport(report), "MONTHLY STRATEGY REVIEW") { t.Fatal("format missing") }
	disabled, _ := NewMonthlyReportScheduler(ps, ss, false, slog.Default(), now)
	disabled.GenerateMonthlyReport()
	_ = NextMonthlyReportRun(now())
}

type posStore struct {
	positions []MonthlyPosition
}

func (p *posStore) ListPositions(ctx context.Context) ([]MonthlyPosition, error) { return p.positions, nil }

type sigStore struct {
	signals []MonthlySignal
}

func (s *sigStore) ListSignals(ctx context.Context) ([]MonthlySignal, error) { return s.signals, nil }

func ptr(f float64) *float64 { return &f }
func ptrF(f float64) *float64 { return &f }

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub { return true }
	}
	return false
}
