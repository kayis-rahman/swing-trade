package domain

import (
	"strings"
	"testing"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

func mustCapital(t *testing.T, value string) coredecimal.Decimal {
	t.Helper()
	capital, err := coredecimal.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return capital
}

// Ported from StrategyConfigTest.createCalculatesHashAndDefensivelyCopiesMaps
// with Go's copy-on-read replacing Java's UnsupportedOperationException pin.
func TestStrategyConfigCreateHashesAndCopiesMaps(t *testing.T) {
	params := map[string]any{"emaFast": 20}
	created, err := CreateStrategyConfig("BREAKOUT_STRICT", 1, "BREAKOUT", params,
		map[string]any{"sentimentGate": true}, StrategyConfigModeShadow,
		mustCapital(t, "500000"), true, nil,
		time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("create error = %v", err)
	}
	params["emaFast"] = 99
	if created.Params()["emaFast"] != 20 {
		t.Fatalf("params leaked the caller's mutation: %#v", created.Params())
	}
	if len(created.ParamsHash) != 64 || !strings.ContainsAny(created.ParamsHash, "0123456789abcdef") {
		t.Fatalf("params hash = %q", created.ParamsHash)
	}
	mutated := created.Params()
	mutated["x"] = true
	if _, present := created.Params()["x"]; present {
		t.Fatal("accessor returned a live reference to stored params")
	}
}

// Ported from StrategyConfigTest.rejectsInvalidVersionModeCapitalAndHash.
func TestStrategyConfigRejectsInvalidConfiguration(t *testing.T) {
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	hash := StrategyConfigHash("BREAKOUT", nil, nil)
	if _, err := NewStrategyConfig(nil, "A", 0, "BREAKOUT", map[string]any{}, map[string]any{},
		hash, StrategyConfigModeOff, mustCapital(t, "1"), false, nil, now); err == nil ||
		err.Error() != "Version must be positive" {
		t.Fatalf("version validation = %v", err)
	}
	if _, err := CreateStrategyConfig("A", 1, "BREAKOUT", map[string]any{}, map[string]any{},
		StrategyConfigModeOff, mustCapital(t, "-1"), false, nil, now); err == nil ||
		err.Error() != "Paper capital must be non-negative" {
		t.Fatalf("capital validation = %v", err)
	}
}

func TestStrategyConfigConstructorValidatesEveryField(t *testing.T) {
	now := time.Now()
	hash := StrategyConfigHash("BREAKOUT", map[string]any{}, map[string]any{})
	for name, tc := range map[string]struct {
		variantID    string
		version      int
		strategyType string
		params       map[string]any
		overlays     map[string]any
		paramsHash   string
		mode         StrategyConfigMode
		capital      string
		createdAt    time.Time
		want         string
	}{
		"variant too long": {"", 1, "BREAKOUT", map[string]any{}, map[string]any{}, hash, StrategyConfigModeOff, "1", now,
			"Variant id must contain 1-40 characters"},
		"variant missing": {strings.Repeat("v", 41), 1, "BREAKOUT", map[string]any{}, map[string]any{}, hash, StrategyConfigModeOff, "1", now,
			"Variant id must contain 1-40 characters"},
		"type too long": {"v", 1, strings.Repeat("t", 31), map[string]any{}, map[string]any{}, hash, StrategyConfigModeOff, "1", now,
			"Strategy type must contain 1-30 characters"},
		"nil params": {"v", 1, "BREAKOUT", nil, map[string]any{}, hash, StrategyConfigModeOff, "1", now,
			"Params are required"},
		"nil overlays": {"v", 1, "BREAKOUT", map[string]any{}, nil, hash, StrategyConfigModeOff, "1", now,
			"Overlays are required"},
		"bad hash": {"v", 1, "BREAKOUT", map[string]any{}, map[string]any{}, "nope", StrategyConfigModeOff, "1", now,
			"Params hash must be a SHA-256 hex digest"},
		"missing mode": {"v", 1, "BREAKOUT", map[string]any{}, map[string]any{}, hash, "", "1", now,
			"Mode is required"},
		"missing created at": {"v", 1, "BREAKOUT", map[string]any{}, map[string]any{}, hash, StrategyConfigModeOff, "1", time.Time{},
			"Created at is required"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := NewStrategyConfig(nil, tc.variantID, tc.version, tc.strategyType, tc.params, tc.overlays,
				tc.paramsHash, tc.mode, mustCapital(t, tc.capital), false, nil, tc.createdAt)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

// Vectors computed independently from Java's StrategyConfig.canonical rules:
// class-prefixed scalars, sorted map keys, concatenated sections.
func TestStrategyConfigHashVectorsMatchJavaCanonical(t *testing.T) {
	for _, tc := range []struct {
		strategyType string
		params       map[string]any
		overlays     map[string]any
		want         string
	}{
		{"BREAKOUT", map[string]any{"emaFast": 20, "emaSlow": 50}, map[string]any{"sentimentGate": true},
			"2d0de9b868ce73de686f230f02e26a107908158eb3023a14da63d717941847f5"},
		{"BREAKOUT", map[string]any{"emaFast": 20}, map[string]any{},
			"c7ccd554a30fdf783edec9a0dd016a0f03cd27cfaaa53af0811bbd07c8085fa5"},
		{"WIDE_RSI", map[string]any{"rsiLower": "45", "rsiUpper": "72"}, map[string]any{},
			"d508fd558ea25dd112e8b282e6c4f9b8906731e7bc5a44795c643f5191d32dab"},
	} {
		got := StrategyConfigHash(tc.strategyType, tc.params, tc.overlays)
		if got != tc.want {
			t.Fatalf("hash(%s, %v, %v) = %s, want %s", tc.strategyType, tc.params, tc.overlays, got, tc.want)
		}
	}
}

func TestCanonicalJavaTypeMapping(t *testing.T) {
	if got := canonicalJava(20); got != "java.lang.Integer:20" {
		t.Fatalf("int canonical = %q", got)
	}
	if got := canonicalJava(int64(1) << 40); !strings.HasPrefix(got, "java.lang.Long:") {
		t.Fatalf("long canonical = %q", got)
	}
	if got := canonicalJava(true); got != "java.lang.Boolean:true" {
		t.Fatalf("bool canonical = %q", got)
	}
	if got := canonicalJava(float64(20)); got != "java.lang.Double:20.0" {
		t.Fatalf("double canonical = %q", got)
	}
	if got := canonicalJava(nil); got != "null" {
		t.Fatalf("null canonical = %q", got)
	}
	if got := canonicalJava(map[string]any{"b": 1, "a": 2}); got !=
		"{java.lang.String:ajava.lang.Integer:2java.lang.String:bjava.lang.Integer:1}" {
		t.Fatalf("map canonical = %q", got)
	}
	if got := canonicalJava([]any{1, "x"}); got != "[java.lang.Integer:1java.lang.String:x]" {
		t.Fatalf("list canonical = %q", got)
	}
	decimalValue := mustCapital(t, "0.97")
	if got := canonicalJava(decimalValue); got != "java.math.BigDecimal:0.9700" {
		t.Fatalf("decimal canonical = %q", got)
	}
}
