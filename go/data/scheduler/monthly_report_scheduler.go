package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type MonthlyPosition struct {
	Status      string
	RealizedPnL *float64
}

type MonthlySignal struct {
	Type string
}

type MonthlyPositionStore interface {
	ListPositions(ctx context.Context) ([]MonthlyPosition, error)
}

type MonthlySignalStore interface {
	ListSignals(ctx context.Context) ([]MonthlySignal, error)
}

type MonthlyReportScheduler struct {
	positions MonthlyPositionStore
	signals   MonthlySignalStore
	enabled   bool
	now       func() time.Time
	logger    *slog.Logger
}

func NewMonthlyReportScheduler(
	positions MonthlyPositionStore,
	signals MonthlySignalStore,
	enabled bool,
	logger *slog.Logger,
	now func() time.Time,
) (*MonthlyReportScheduler, error) {
	if logger == nil || now == nil {
		return nil, fmt.Errorf("logger and clock are required")
	}
	return &MonthlyReportScheduler{positions: positions, signals: signals, enabled: enabled, logger: logger, now: now}, nil
}

func (s *MonthlyReportScheduler) Enabled() bool { return s.enabled }

func (s *MonthlyReportScheduler) GenerateMonthlyReport() {
	s.GenerateMonthlyReportWithContext(context.Background())
}

func (s *MonthlyReportScheduler) GenerateMonthlyReportWithContext(ctx context.Context) {
	if !s.enabled {
		if s.logger != nil {
			s.logger.Debug("monthly report disabled")
		}
		return
	}
	if s.logger != nil {
		s.logger.Info("Generating monthly strategy review report")
	}
	now := s.now().In(ist)
	reportDate := now
	var positions []MonthlyPosition
	var signals []MonthlySignal
	if s.positions != nil {
		ps, err := s.positions.ListPositions(ctx)
		if err != nil {
			if s.logger != nil {
				s.logger.Error("monthly report positions failed", "error", err)
			}
			return
		}
		positions = ps
	}
	if s.signals != nil {
		ss, err := s.signals.ListSignals(ctx)
		if err != nil {
			if s.logger != nil {
				s.logger.Error("monthly report signals failed", "error", err)
			}
			return
		}
		signals = ss
	}
	report := BuildMonthlyReport(reportDate, positions, signals)
	msg := FormatMonthlyReport(report)
	if s.logger != nil {
		s.logger.Info("Monthly report: " + msg)
		s.logger.Info("Monthly report generated")
	}
}

func (s *MonthlyReportScheduler) Generate(ctx context.Context) {
	s.GenerateMonthlyReportWithContext(ctx)
}

func (s *MonthlyReportScheduler) Run(ctx context.Context) { s.GenerateMonthlyReportWithContext(ctx) }

func BuildMonthlyReport(reportDate time.Time, positions []MonthlyPosition, signals []MonthlySignal) map[string]any {
	report := map[string]any{}
	report["reportDate"] = reportDate.Format("2006-01-02")

	closed := []MonthlyPosition{}
	for _, p := range positions {
		if p.Status != "OPEN" {
			closed = append(closed, p)
		}
	}
	report["totalTrades"] = len(closed)
	report["totalSignals"] = len(signals)

	winRate := 0.0
	if len(closed) > 0 {
		profitable := 0
		for _, p := range closed {
			if p.Status == "TARGET_HIT" || (p.RealizedPnL != nil && *p.RealizedPnL > 0) {
				profitable++
			}
		}
		winRate = float64(profitable) * 100.0 / float64(len(closed))
	}
	avgWin := averageOfMonthly(closed, true)
	avgLoss := averageOfMonthly(closed, false)
	totalPnL := 0.0
	for _, p := range closed {
		if p.RealizedPnL != nil {
			totalPnL += *p.RealizedPnL
		}
	}
	profitFactor := 0.0
	totalWins := 0.0
	totalLosses := 0.0
	for _, p := range closed {
		if p.RealizedPnL == nil {
			continue
		}
		if *p.RealizedPnL > 0 {
			totalWins += *p.RealizedPnL
		} else if *p.RealizedPnL < 0 {
			totalLosses += -*p.RealizedPnL
		}
	}
	if totalLosses > 0 {
		profitFactor = totalWins / totalLosses
	} else if totalWins > 0 {
		profitFactor = 999.99
	}

	report["winRate"] = fmt.Sprintf("%.2f%%", winRate)
	report["totalPnL"] = fmt.Sprintf("Rs. %.2f", totalPnL)
	report["avgWinAmount"] = fmt.Sprintf("Rs. %.2f", avgWin)
	report["avgLossAmount"] = fmt.Sprintf("Rs. %.2f", avgLoss)
	report["profitFactor"] = fmt.Sprintf("%.2f", profitFactor)

	stopLoss := 0
	targetHit := 0
	manual := 0
	for _, p := range positions {
		switch p.Status {
		case "STOPPED":
			stopLoss++
		case "TARGET_HIT":
			targetHit++
		case "MANUAL":
			manual++
		}
	}
	exitReasons := map[string]any{"STOP_LOSS": stopLoss, "TARGET_HIT": targetHit, "MANUAL": manual}
	report["exitReasons"] = exitReasons

	buy := 0
	sell := 0
	hold := 0
	for _, s := range signals {
		switch s.Type {
		case "BUY":
			buy++
		case "SELL":
			sell++
		case "HOLD":
			hold++
		}
	}
	signalDistribution := map[string]any{"BUY": buy, "SELL": sell, "HOLD": hold}
	report["signalDistribution"] = signalDistribution

	report["totalReturn"] = "N/A"
	report["sharpeRatio"] = "N/A"
	report["maxDrawdown"] = "N/A"

	return report
}

func averageOfMonthly(positions []MonthlyPosition, positive bool) float64 {
	sum := 0.0
	count := 0
	for _, p := range positions {
		if p.RealizedPnL == nil {
			continue
		}
		if positive && *p.RealizedPnL > 0 {
			sum += *p.RealizedPnL
			count++
		}
		if !positive && *p.RealizedPnL < 0 {
			sum += -*p.RealizedPnL
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func FormatMonthlyReport(report map[string]any) string {
	sb := "📊 <b>MONTHLY STRATEGY REVIEW</b>\n\n"
	sb += "📅 <b>Period:</b> " + fmt.Sprint(report["reportDate"]) + "\n\n"
	sb += "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
	sb += "📈 <b>PERFORMANCE METRICS</b>\n"
	sb += "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
	sb += "📊 <b>Total Trades:</b> <code>" + fmt.Sprint(report["totalTrades"]) + "</code>\n"
	sb += "📊 <b>Win Rate:</b> <b>" + fmt.Sprint(report["winRate"]) + "</b>\n"
	sb += "💰 <b>Total P&L:</b> <b>" + fmt.Sprint(report["totalPnL"]) + "</b>\n"
	sb += "✅ <b>Avg Win:</b> <code>" + fmt.Sprint(report["avgWinAmount"]) + "</code>\n"
	sb += "❌ <b>Avg Loss:</b> <code>" + fmt.Sprint(report["avgLossAmount"]) + "</code>\n"
	sb += "📉 <b>Profit Factor:</b> <b>" + fmt.Sprint(report["profitFactor"]) + "</b>\n\n"

	sb += "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
	sb += "🎯 <b>EXIT REASONS</b>\n"
	sb += "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
	exitReasons, _ := report["exitReasons"].(map[string]any)
	for _, reason := range []string{"STOP_LOSS", "TARGET_HIT", "MANUAL"} {
		sb += "🔸 <b>" + reason + ":</b> <code>" + fmt.Sprint(exitReasons[reason]) + "</code>\n"
	}
	sb += "\n"

	sb += "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
	sb += "📡 <b>SIGNAL DISTRIBUTION</b>\n"
	sb += "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
	signalDistribution, _ := report["signalDistribution"].(map[string]any)
	for _, typ := range []string{"BUY", "SELL", "HOLD"} {
		sb += "🔸 <b>" + typ + ":</b> <code>" + fmt.Sprint(signalDistribution[typ]) + "</code>\n"
	}

	return sb
}

func (s *MonthlyReportScheduler) StartLoop(ctx context.Context) {
	runOnSchedule(ctx, waitForDeadline, s.now, NextMonthlyReportRun, func(ctx context.Context) { s.GenerateMonthlyReportWithContext(ctx) })
}

func NextMonthlyReportRun(now time.Time) time.Time {
	local := now.In(ist)
	for day := 1; day <= 7; day++ {
		candidate := time.Date(local.Year(), local.Month(), day, 3, 0, 0, 0, ist)
		if candidate.After(now) && candidate.Weekday() == time.Sunday {
			return candidate
		}
	}
	// next month
	nextMonth := local.AddDate(0, 1, 0)
	for day := 1; day <= 7; day++ {
		candidate := time.Date(nextMonth.Year(), nextMonth.Month(), day, 3, 0, 0, 0, ist)
		if candidate.Weekday() == time.Sunday {
			return candidate
		}
	}
	return local.AddDate(0, 0, 1)
}
