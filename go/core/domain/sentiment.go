package domain

import (
	"fmt"
	"time"
)

// SentimentScore mirrors Java's SentimentResult.SentimentScore enum.
type SentimentScore string

// Sentiment score values matching Java's enum constants.
const (
	SentimentScorePositive SentimentScore = "POSITIVE"
	SentimentScoreNeutral  SentimentScore = "NEUTRAL"
	SentimentScoreNegative SentimentScore = "NEGATIVE"
	SentimentScoreUnknown  SentimentScore = "UNKNOWN"
)

// DisplayName mirrors SentimentScore.getDisplayName().
func (score SentimentScore) DisplayName() string {
	switch score {
	case SentimentScorePositive:
		return "Positive"
	case SentimentScoreNeutral:
		return "Neutral"
	case SentimentScoreNegative:
		return "Negative"
	case SentimentScoreUnknown:
		return "Unknown"
	default:
		return ""
	}
}

// Rank mirrors the ordinal comparison used by Java's SentimentGateConfig.rank:
// NEGATIVE (0) < NEUTRAL and UNKNOWN (1) < POSITIVE (2).
func (score SentimentScore) Rank() int {
	switch score {
	case SentimentScoreNegative:
		return 0
	case SentimentScoreNeutral, SentimentScoreUnknown:
		return 1
	case SentimentScorePositive:
		return 2
	default:
		return -1
	}
}

// IsPositive mirrors SentimentScore == POSITIVE.
func (score SentimentScore) IsPositive() bool { return score == SentimentScorePositive }

// IsNeutral mirrors SentimentScore == NEUTRAL.
func (score SentimentScore) IsNeutral() bool { return score == SentimentScoreNeutral }

// IsNegative mirrors SentimentScore == NEGATIVE.
func (score SentimentScore) IsNegative() bool { return score == SentimentScoreNegative }

// IsUnknown mirrors SentimentScore == UNKNOWN.
func (score SentimentScore) IsUnknown() bool { return score == SentimentScoreUnknown }

// SupportsEntry mirrors SentimentResult.supportsEntry(): POSITIVE or NEUTRAL.
func (score SentimentScore) SupportsEntry() bool {
	return score.IsPositive() || score.IsNeutral()
}

// SentimentResult mirrors Java's com.swingtrade.domain.SentimentResult record.
// Nullable record components use pointers; Java's compatibility constructors
// collapse into Go struct literals, and CreateSentimentResult mirrors the
// create(...) factory including its confidence clamp and defaults.
type SentimentResult struct {
	ID             *int64
	Symbol         string
	Date           time.Time
	Score          SentimentScore
	Summary        *string
	RawContent     *string
	Confidence     *float64
	AnalyzedAt     time.Time
	RedFlags       []string
	Catalysts      []string
	PromptHash     *string
	ModelVersion   *string
	ArticleCount   int
	Source         string
	ArticleIDs     []int64
	AuditRequestID *string
	DegradedReason *string
}

// CreateSentimentResult mirrors SentimentResult.create with empty red-flag and
// catalyst lists: confidence is clamped to [0,1], analyzedAt becomes the
// current date like Java's LocalDate.now(), id, prompt hash, model version,
// and audit request stay unset, source defaults to "DEFAULT", and article
// count starts at zero.
func CreateSentimentResult(
	symbol string,
	date time.Time,
	score SentimentScore,
	summary *string,
	rawContent *string,
	confidence *float64,
	redFlags []string,
	catalysts []string,
) SentimentResult {
	normalized := (*float64)(nil)
	if confidence != nil {
		clamped := *confidence
		if clamped < 0 {
			clamped = 0
		}
		if clamped > 1 {
			clamped = 1
		}
		normalized = &clamped
	}
	if redFlags == nil {
		redFlags = []string{}
	}
	if catalysts == nil {
		catalysts = []string{}
	}
	return SentimentResult{
		Symbol: symbol, Date: date, Score: score,
		Summary: summary, RawContent: rawContent,
		Confidence: normalized, AnalyzedAt: time.Now(),
		RedFlags: redFlags, Catalysts: catalysts,
		ArticleCount: 0, Source: "DEFAULT",
		ArticleIDs: []int64{},
	}
}

// IsPositive mirrors SentimentResult.isPositive().
func (result SentimentResult) IsPositive() bool { return result.Score.IsPositive() }

// IsNeutral mirrors SentimentResult.isNeutral().
func (result SentimentResult) IsNeutral() bool { return result.Score.IsNeutral() }

// IsNegative mirrors SentimentResult.isNegative().
func (result SentimentResult) IsNegative() bool { return result.Score.IsNegative() }

// IsUnknown mirrors SentimentResult.isUnknown().
func (result SentimentResult) IsUnknown() bool { return result.Score.IsUnknown() }

// SupportsEntry mirrors SentimentResult.supportsEntry().
func (result SentimentResult) SupportsEntry() bool { return result.Score.SupportsEntry() }

// String renders the Java record's toString shape that the domain tests pin:
// SentimentResult[field=value, ...] with null for absent pointers.
func (result SentimentResult) String() string {
	return fmt.Sprintf(
		"SentimentResult[id=%s, symbol=%s, date=%s, score=%s, summary=%s, rawContent=%s, confidence=%s, analyzedAt=%s, redFlags=%s, catalysts=%s, promptHash=%s, modelVersion=%s, articleCount=%d, source=%s, articleIds=%v, auditRequestId=%s, degradedReason=%s]",
		formatNullInt64(result.ID), result.Symbol, result.Date.Format("2006-01-02"), result.Score,
		formatNullString(result.Summary), formatNullString(result.RawContent),
		formatNullConfidence(result.Confidence),
		result.AnalyzedAt.Format("2006-01-02"),
		result.RedFlags, result.Catalysts,
		formatNullString(result.PromptHash), formatNullString(result.ModelVersion),
		result.ArticleCount, result.Source, result.ArticleIDs,
		formatNullString(result.AuditRequestID), formatNullString(result.DegradedReason),
	)
}

func formatNullInt64(value *int64) string {
	if value == nil {
		return "null"
	}
	return fmt.Sprintf("%d", *value)
}

func formatNullString(value *string) string {
	if value == nil {
		return "null"
	}
	return *value
}

func formatNullConfidence(value *float64) string {
	if value == nil {
		return "null"
	}
	return fmt.Sprintf("%v", *value)
}
