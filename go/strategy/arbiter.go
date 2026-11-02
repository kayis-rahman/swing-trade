package strategy

import (
	"fmt"
	"sort"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

// ArbitrationRule selects how the signal tournament ranks competing BUY variants.
type ArbitrationRule string

const (
	// ArbitrationRuleHighestConfidence selects by confidence.
	ArbitrationRuleHighestConfidence ArbitrationRule = "HIGHEST_CONFIDENCE"
	// ArbitrationRuleEvidenceRanked selects by track record.
	ArbitrationRuleEvidenceRanked ArbitrationRule = "EVIDENCE_RANKED"
)

// ShadowClosedTrade mirrors Java's ShadowClosedTrade record used by the arbiter
// to compute a variant's track record.
type ShadowClosedTrade struct {
	PortfolioID string
	Symbol      string
	EntryDate   time.Time
	ExitDate    *time.Time
	EntryPrice  *coredecimal.Decimal
	ExitPrice   *coredecimal.Decimal
	StopLoss    *coredecimal.Decimal
	Target      *coredecimal.Decimal
	Quantity    int
	ExitReason  string
	Pnl         *coredecimal.Decimal
}

// VariantSignalOutcome is one variant's decision for a symbol/day, as the signal
// tournament sees it.
type VariantSignalOutcome struct {
	VariantID            string
	StrategyVersion      int
	Type                 domain.SignalType
	SentimentGateEnabled bool
	Persisted            bool
	Confidence           *coredecimal.Decimal
}

// Candidate is one competing BUY: which variant and how confident.
type Candidate struct {
	VariantID  string
	Confidence coredecimal.Decimal
}

// ArbiterMinEvidenceTrades is the minimum closed trades before a variant's
// track record counts.
const ArbiterMinEvidenceTrades = 5

// evidence computes the mean return (%) of a variant's closed shadow trades that
// exited before asOf, or false when fewer than MinEvidenceTrades.
func evidence(trades []ShadowClosedTrade, asOf time.Time) (float64, bool) {
	var returns []float64
	for _, trade := range trades {
		if trade.ExitDate == nil || !trade.ExitDate.Before(asOf) {
			continue
		}
		if trade.Pnl == nil || trade.EntryPrice == nil {
			continue
		}
		if trade.EntryPrice.Cmp(coredecimal.Decimal{}) <= 0 {
			continue
		}
		if trade.Quantity <= 0 {
			continue
		}
		pnl := trade.Pnl
		entry := trade.EntryPrice
		// convert Decimal to float64 via string parse is acceptable here; Java uses doubleValue.
		entryF, _ := parseDecimalFloat(entry.String())
		pnlF, _ := parseDecimalFloat(pnl.String())
		returns = append(returns, pnlF/(entryF*float64(trade.Quantity))*100.0)
	}
	if len(returns) < ArbiterMinEvidenceTrades {
		return 0, false
	}
	var sum float64
	for _, r := range returns {
		sum += r
	}
	return sum / float64(len(returns)), true
}

func parseDecimalFloat(s string) (float64, error) {
	var f float64
	// minimal
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}
	whole, frac := s, ""
	if i := byteIndexOf(s, '.'); i >= 0 {
		whole = s[:i]
		frac = s[i+1:]
	}
	// parse whole
	w := 0.0
	for _, c := range whole {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("bad")
		}
		w = w*10 + float64(c-'0')
	}
	f = w
	if frac != "" {
		scale := 1.0
		for range frac {
			scale *= 10
		}
		fracV := 0.0
		for _, c := range frac {
			if c < '0' || c > '9' {
				return 0, fmt.Errorf("bad")
			}
			fracV = fracV*10 + float64(c-'0')
		}
		f += fracV / scale
	}
	if neg {
		f = -f
	}
	return f, nil
}

func byteIndexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// pickVariant chooses winning variant id among candidates.
func pickVariant(candidates []Candidate, rule ArbitrationRule, evidenceByVariant map[string]float64) *string {
	if len(candidates) == 0 {
		return nil
	}
	// clone
	// sort candidates for determinism then max
	sorted := append([]Candidate(nil), candidates...)
	sort.Slice(sorted, func(i, j int) bool {
		// tie-break last
		return sorted[i].VariantID < sorted[j].VariantID
	})
	var winner *Candidate
	for i := range sorted {
		c := sorted[i]
		if winner == nil {
			w := c
			winner = &w
			continue
		}
		// compare
		if !better(c, *winner, rule, evidenceByVariant) {
			continue
		}
		w := c
		winner = &w
	}
	if winner == nil {
		return nil
	}
	id := winner.VariantID
	return &id
}

func better(c, w Candidate, rule ArbitrationRule, evidence map[string]float64) bool {
	if rule == ArbitrationRuleEvidenceRanked {
		es := evidence[c.VariantID] // zero default
		ew := evidence[w.VariantID]
		if es != ew {
			return es > ew
		}
	}
	// confidence
	if c.Confidence.Cmp(w.Confidence) != 0 {
		return c.Confidence.Cmp(w.Confidence) > 0
	}
	// Tie-break: smaller variant id wins (matches the ascending sort + strict-replace
	// behavior and the parity tests); confidence already tied here.
	return c.VariantID < w.VariantID
}

// Pick selects a BUY for the selected book.
func Pick(outcomes []VariantSignalOutcome) *VariantSignalOutcome {
	return PickFull(outcomes, ArbitrationRuleHighestConfidence, map[string]float64{}, nil, nil)
}

// PickWith selects using rule and evidence.
func PickWith(outcomes []VariantSignalOutcome, rule ArbitrationRule, evidence map[string]float64) *VariantSignalOutcome {
	return PickFull(outcomes, rule, evidence, nil, nil)
}

// PickFull selects with open-position and champion preferences.
//
//nolint:funlen
func PickFull(outcomes []VariantSignalOutcome, rule ArbitrationRule, evidenceByVariant map[string]float64, openPositionVariants *map[string]struct{}, champion *string) *VariantSignalOutcome {
	openSet := map[string]struct{}{}
	if openPositionVariants != nil {
		openSet = *openPositionVariants
	}
	var championOpt *string
	if champion != nil {
		championOpt = champion
	}
	eligible := []VariantSignalOutcome{}
	for _, o := range outcomes {
		if o.Type != domain.SignalTypeBuy {
			continue
		}
		if !o.Persisted {
			continue
		}
		if o.Confidence == nil {
			continue
		}
		eligible = append(eligible, o)
	}
	// open position winner
	openCandidates := []VariantSignalOutcome{}
	for _, o := range eligible {
		if _, ok := openSet[o.VariantID]; ok {
			openCandidates = append(openCandidates, o)
		}
	}
	if len(openCandidates) > 0 {
		sort.Slice(openCandidates, func(i, j int) bool { return openCandidates[i].VariantID < openCandidates[j].VariantID })
		w := openCandidates[0]
		return &w
	}
	if championOpt != nil {
		for _, o := range eligible {
			if o.VariantID == *championOpt {
				w := o
				return &w
			}
		}
	}
	// pick variant
	cands := make([]Candidate, 0, len(eligible))
	for _, o := range eligible {
		cands = append(cands, Candidate{VariantID: o.VariantID, Confidence: *o.Confidence})
	}
	ev := evidenceByVariant
	if rule != ArbitrationRuleEvidenceRanked {
		ev = map[string]float64{}
	}
	id := pickVariant(cands, rule, ev)
	if id == nil {
		return nil
	}
	for i := range eligible {
		if eligible[i].VariantID == *id {
			w := eligible[i]
			return &w
		}
	}
	return nil
}

// Evidence computes a variant track record.
func Evidence(trades []ShadowClosedTrade, asOf time.Time) (float64, bool) {
	return evidence(trades, asOf)
}

// PickVariant chooses a winning variant id.
func PickVariant(candidates []Candidate, rule ArbitrationRule, evidenceByVariant map[string]float64) *string {
	return pickVariant(candidates, rule, evidenceByVariant)
}

// Need import fmt for error (we didn't use fmt? yes). Add import.
