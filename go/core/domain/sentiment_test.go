package domain

import (
	"strings"
	"testing"
	"time"
)

var (
	testSymbol     = "RELIANCE"
	testDate       = time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	testSummary    = "Positive earnings report and strong guidance"
	testRawContent = "News article content about company earnings"
	testConfidence = 0.85
)

func stringPtrDomain(value string) *string  { return &value }
func floatPtrDomain(value float64) *float64 { return &value }

func createTestResult(t *testing.T, score SentimentScore, confidence *float64) SentimentResult {
	t.Helper()
	return CreateSentimentResult(testSymbol, testDate, score,
		stringPtrDomain(testSummary), stringPtrDomain(testRawContent), confidence, nil, nil)
}

// Ported from SentimentResultTest.SentimentResultCreation.
func TestCreateSentimentResultMatchesJavaFactory(t *testing.T) {
	result := createTestResult(t, SentimentScorePositive, &testConfidence)
	if result.Symbol != testSymbol || !result.Date.Equal(testDate) ||
		result.Score != SentimentScorePositive ||
		result.Summary == nil || *result.Summary != testSummary ||
		result.RawContent == nil || *result.RawContent != testRawContent ||
		result.Confidence == nil || *result.Confidence != 0.85 ||
		result.ID != nil {
		t.Fatalf("created result = %+v", result)
	}
	today := time.Now().Format("2006-01-02")
	if result.AnalyzedAt.Format("2006-01-02") != today {
		t.Fatalf("analyzedAt = %s, want today (%s)", result.AnalyzedAt, today)
	}
	if result.Source != "DEFAULT" || result.ArticleCount != 0 ||
		result.PromptHash != nil || result.ModelVersion != nil ||
		result.AuditRequestID != nil || result.DegradedReason != nil {
		t.Fatalf("create defaults = %+v", result)
	}
	if len(result.RedFlags) != 0 || len(result.Catalysts) != 0 || len(result.ArticleIDs) != 0 {
		t.Fatalf("create list defaults = %+v", result)
	}
}

func TestCreateSentimentResultOtherScores(t *testing.T) {
	for _, score := range []SentimentScore{SentimentScoreNeutral, SentimentScoreNegative} {
		result := createTestResult(t, score, &testConfidence)
		if result.Score != score {
			t.Fatalf("score = %s, want %s", result.Score, score)
		}
	}
}

// Ported from SentimentResultTest confidence normalization group.
func TestCreateSentimentResultClampsConfidenceLikeJava(t *testing.T) {
	for name, tc := range map[string]struct {
		confidence *float64
		want       *float64
	}{
		"in range":              {confidence: floatPtrDomain(0.75), want: floatPtrDomain(0.75)},
		"negative clamps to 0":  {confidence: floatPtrDomain(-0.3), want: floatPtrDomain(0)},
		"above one clamps to 1": {confidence: floatPtrDomain(1.5), want: floatPtrDomain(1)},
		"zero boundary":         {confidence: floatPtrDomain(0), want: floatPtrDomain(0)},
		"one boundary":          {confidence: floatPtrDomain(1), want: floatPtrDomain(1)},
		"null stays null":       {confidence: nil, want: nil},
		"high precision":        {confidence: floatPtrDomain(0.9999), want: floatPtrDomain(0.9999)},
	} {
		t.Run(name, func(t *testing.T) {
			result := createTestResult(t, SentimentScorePositive, tc.confidence)
			switch {
			case tc.want == nil && result.Confidence != nil:
				t.Fatalf("confidence = %v, want nil", *result.Confidence)
			case tc.want == nil:
			case result.Confidence == nil:
				t.Fatalf("confidence = nil, want %v", *tc.want)
			case *result.Confidence != *tc.want:
				t.Fatalf("confidence = %v, want %v", *result.Confidence, *tc.want)
			}
		})
	}
}

// Ported from SentimentResultTest enum groups (hashCode is Java-specific and
// intentionally not ported).
func TestSentimentScoreEnumContract(t *testing.T) {
	values := []SentimentScore{SentimentScorePositive, SentimentScoreNeutral, SentimentScoreNegative, SentimentScoreUnknown}
	if len(values) != 4 {
		t.Fatal("sentiment score enum must keep four values")
	}
	for _, score := range values {
		if score == "" || score.Rank() < 0 {
			t.Fatalf("score %q invalid", score)
		}
	}
	if SentimentScorePositive == SentimentScoreNeutral || SentimentScorePositive == SentimentScoreNegative {
		t.Fatal("distinct scores compared equal")
	}
	for score, displayName := range map[SentimentScore]string{
		SentimentScorePositive: "Positive",
		SentimentScoreNeutral:  "Neutral",
		SentimentScoreNegative: "Negative",
		SentimentScoreUnknown:  "Unknown",
	} {
		if got := score.DisplayName(); got != displayName {
			t.Fatalf("%s display name = %q, want %q", score, got, displayName)
		}
	}
}

// Ported from SentimentResultTest score-comparison and check-method groups.
func TestSentimentResultScorePredicates(t *testing.T) {
	positive := createTestResult(t, SentimentScorePositive, &testConfidence)
	neutral := createTestResult(t, SentimentScoreNeutral, &testConfidence)
	negative := createTestResult(t, SentimentScoreNegative, &testConfidence)

	if !positive.IsPositive() || positive.IsNeutral() || positive.IsNegative() {
		t.Fatalf("positive predicates wrong: %+v", positive.Score)
	}
	if neutral.IsPositive() || !neutral.IsNeutral() || neutral.IsNegative() {
		t.Fatalf("neutral predicates wrong: %+v", neutral.Score)
	}
	if negative.IsNegative() != true || negative.IsPositive() || negative.IsNeutral() {
		t.Fatalf("negative predicates wrong: %+v", negative.Score)
	}
}

// Ported from SentimentResultTest.SupportsEntryTests.
func TestSentimentResultSupportsEntryLikeJava(t *testing.T) {
	if !createTestResult(t, SentimentScorePositive, &testConfidence).SupportsEntry() {
		t.Fatal("positive sentiment must support entry")
	}
	if !createTestResult(t, SentimentScoreNeutral, &testConfidence).SupportsEntry() {
		t.Fatal("neutral sentiment must support entry")
	}
	if createTestResult(t, SentimentScoreNegative, &testConfidence).SupportsEntry() {
		t.Fatal("negative sentiment must block entry")
	}
	if SentimentScoreUnknown.SupportsEntry() {
		t.Fatal("unknown sentiment must not support entry by score rank")
	}
}

// The gate orders scores with SentimentGateConfig.rank: NEGATIVE0,
// NEUTRAL and UNKNOWN1, POSITIVE2.
func TestSentimentScoreRankMatchesGateOrdering(t *testing.T) {
	if SentimentScoreNegative.Rank() != 0 ||
		SentimentScoreNeutral.Rank() != 1 ||
		SentimentScoreUnknown.Rank() != 1 ||
		SentimentScorePositive.Rank() != 2 {
		t.Fatalf("ranks: neg=%d neu=%d unk=%d pos=%d",
			SentimentScoreNegative.Rank(), SentimentScoreNeutral.Rank(),
			SentimentScoreUnknown.Rank(), SentimentScorePositive.Rank())
	}
}

// Ported from SentimentResultTest.ToStringTests (contains-style assertions).
func TestSentimentResultStringKeepsJavaShape(t *testing.T) {
	result := createTestResult(t, SentimentScorePositive, &testConfidence)
	rendered := result.String()
	for _, want := range []string{
		"SentimentResult", "symbol=RELIANCE", "date=2024-01-15", "score=POSITIVE",
		"summary=" + testSummary, "confidence=0.85",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("String() missing %q: %s", want, rendered)
		}
	}
	nullConfidence := createTestResult(t, SentimentScorePositive, nil)
	if !strings.Contains(nullConfidence.String(), "confidence=null") {
		t.Fatalf("null confidence rendering: %s", nullConfidence.String())
	}
}
