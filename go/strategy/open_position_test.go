package strategy

import (
	"testing"
	"time"
)

func TestOpenPositionSeedsHighWaterMarkAtEntry(t *testing.T) {
	entryPrice := mustParseDecisionDecimal(t, "100.5")
	stop := mustParseDecisionDecimal(t, "95")
	target := mustParseDecisionDecimal(t, "112")
	entryDate := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)

	position, err := OpenPositionAt(7, entryDate, entryPrice, stop, target, 10)
	if err != nil {
		t.Fatalf("OpenPositionAt() error = %v", err)
	}
	if position.EntryIndex != 7 || !position.EntryDate.Equal(entryDate) || position.Quantity != 10 {
		t.Fatalf("position = %+v", position)
	}
	if position.HighWaterMark != entryPrice {
		t.Fatalf("high-water mark = %s, want the entry price", position.HighWaterMark)
	}
	// An explicit mark survives construction.
	explicit := mustParseDecisionDecimal(t, "101")
	withMark, err := NewOpenPosition(7, entryDate, entryPrice, stop, target, 10, &explicit)
	if err != nil || withMark.HighWaterMark != explicit {
		t.Fatalf("explicit mark = %+v, %v", withMark, err)
	}
}

func TestOpenPositionRejectsMissingEntryDate(t *testing.T) {
	_, err := OpenPositionAt(0, time.Time{},
		mustParseDecisionDecimal(t, "100"), mustParseDecisionDecimal(t, "95"),
		mustParseDecisionDecimal(t, "110"), 1)
	if err == nil || err.Error() != "entryDate/entryPrice/stopLoss/target cannot be null" {
		t.Fatalf("validation = %v", err)
	}
}

func TestAdvanceHighWaterMarkKeepsTheMaximum(t *testing.T) {
	entryDate := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	position, err := OpenPositionAt(0, entryDate,
		mustParseDecisionDecimal(t, "100"), mustParseDecisionDecimal(t, "90"),
		mustParseDecisionDecimal(t, "115"), 1)
	if err != nil {
		t.Fatal(err)
	}
	advanced := position.AdvanceHighWaterMark(mustParseDecisionDecimal(t, "108"))
	if advanced.HighWaterMark.String() != "108.0000" {
		t.Fatalf("advanced mark = %s", advanced.HighWaterMark)
	}
	lower := advanced.AdvanceHighWaterMark(mustParseDecisionDecimal(t, "101"))
	if lower.HighWaterMark.String() != "108.0000" {
		t.Fatalf("regressed mark = %s", lower.HighWaterMark)
	}
	if position.HighWaterMark.String() != "100.0000" {
		t.Fatalf("advance mutated the original: %s", position.HighWaterMark)
	}
}
