package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// StrategyConfigMode mirrors Java's StrategyConfig.Mode enum.
type StrategyConfigMode string

// Strategy variant modes matching Java's enum constants.
const (
	StrategyConfigModeOff          StrategyConfigMode = "OFF"
	StrategyConfigModeBacktestOnly StrategyConfigMode = "BACKTEST_ONLY"
	StrategyConfigModeShadow       StrategyConfigMode = "SHADOW"
	StrategyConfigModeChampion     StrategyConfigMode = "CHAMPION"
)

// StrategyConfig mirrors Java's immutable, versioned strategy-variant
// configuration record. The params and overlays maps are private with
// copy-on-access because Go maps are mutable (Java uses unmodifiable maps);
// mutating what an accessor returns never changes the stored config.
type StrategyConfig struct {
	ID           *int64
	VariantID    string
	Version      int
	StrategyType string
	ParamsHash   string
	Mode         StrategyConfigMode
	PaperCapital coredecimal.Decimal
	Current      bool
	Notes        *string
	CreatedAt    time.Time

	params   map[string]any
	overlays map[string]any
}

// Params returns a copy of the parameter map, mirroring Java's
// UnsupportedOperationException semantics through copy-on-read.
func (config StrategyConfig) Params() map[string]any {
	return copyAnyMap(config.params)
}

// Overlays returns a copy of the overlay map.
func (config StrategyConfig) Overlays() map[string]any {
	return copyAnyMap(config.overlays)
}

// NewStrategyConfig mirrors the Java compact constructor's validation and
// stores defensive copies of both maps.
func NewStrategyConfig(
	id *int64,
	variantID string,
	version int,
	strategyType string,
	params map[string]any,
	overlays map[string]any,
	paramsHash string,
	mode StrategyConfigMode,
	paperCapital coredecimal.Decimal,
	current bool,
	notes *string,
	createdAt time.Time,
) (StrategyConfig, error) {
	if len(variantID) == 0 || len(variantID) > 40 {
		return StrategyConfig{}, fmt.Errorf("Variant id must contain 1-40 characters") //nolint:staticcheck // exact Java message
	}
	if version < 1 {
		return StrategyConfig{}, fmt.Errorf("Version must be positive") //nolint:staticcheck // exact Java message
	}
	if len(strategyType) == 0 || len(strategyType) > 30 {
		return StrategyConfig{}, fmt.Errorf("Strategy type must contain 1-30 characters") //nolint:staticcheck // exact Java message
	}
	if params == nil {
		return StrategyConfig{}, fmt.Errorf("Params are required")
	}
	if overlays == nil {
		return StrategyConfig{}, fmt.Errorf("Overlays are required")
	}
	if !isSHA256Hex(paramsHash) {
		return StrategyConfig{}, fmt.Errorf("Params hash must be a SHA-256 hex digest")
	}
	if mode == "" {
		return StrategyConfig{}, fmt.Errorf("Mode is required") //nolint:staticcheck // exact Java message
	}
	if paperCapital.Cmp(coredecimal.Decimal{}) < 0 {
		return StrategyConfig{}, fmt.Errorf("Paper capital must be non-negative") //nolint:staticcheck // exact Java message
	}
	if createdAt.IsZero() {
		return StrategyConfig{}, fmt.Errorf("Created at is required") //nolint:staticcheck // exact Java message
	}
	return StrategyConfig{
		ID: id, VariantID: variantID, Version: version, StrategyType: strategyType,
		ParamsHash: paramsHash, Mode: mode, PaperCapital: paperCapital,
		Current: current, Notes: notes, CreatedAt: createdAt,
		params: copyAnyMap(params), overlays: copyAnyMap(overlays),
	}, nil
}

// CreateStrategyConfig mirrors StrategyConfig.create: computes the params hash
// over the canonical form of (strategyType, params, overlays) with id unset.
func CreateStrategyConfig(
	variantID string,
	version int,
	strategyType string,
	params map[string]any,
	overlays map[string]any,
	mode StrategyConfigMode,
	paperCapital coredecimal.Decimal,
	current bool,
	notes *string,
	createdAt time.Time,
) (StrategyConfig, error) {
	hash := StrategyConfigHash(strategyType, params, overlays)
	return NewStrategyConfig(nil, variantID, version, strategyType, params, overlays,
		hash, mode, paperCapital, current, notes, createdAt)
}

// StrategyConfigHash mirrors StrategyConfig.hash: the SHA-256 of the Java
// class-prefixed canonical form of strategyType, params, and overlays
// concatenated in that order. Vectors are pinned against independently
// computed digests so persisted params_hash values stay comparable.
func StrategyConfigHash(strategyType string, params, overlays map[string]any) string {
	canonical := canonicalJava(strategyType) + canonicalJava(params) + canonicalJava(overlays)
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}

// canonicalJava mirrors Java's StrategyConfig.canonical: maps render as
// {"java.lang.String:<key><canonical value>...} with lexicographically sorted
// keys, lists as [ ... ], null as null, and scalars as
// "<java class>:<value>".
func canonicalJava(value any) string {
	switch typed := value.(type) {
	case nil:
		return "null"
	case map[string]any:
		return canonicalJavaMap(typed)
	case []any:
		return canonicalJavaList(typed)
	case string:
		return "java.lang.String:" + typed
	case bool:
		return "java.lang.Boolean:" + strconv.FormatBool(typed)
	case int:
		return javaIntegralCanonical(int64(typed))
	case int32:
		return javaIntegralCanonical(int64(typed))
	case int64:
		return javaIntegralCanonical(typed)
	case float64:
		return "java.lang.Double:" + javaDoubleString(typed)
	case coredecimal.Decimal:
		return "java.math.BigDecimal:" + typed.String()
	case json.Number:
		text := typed.String()
		if strings.ContainsAny(text, ".eE") {
			return "java.lang.Double:" + text
		}
		integer, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return "java.lang.Double:" + javaDoubleString(0)
		}
		return javaIntegralCanonical(integer)
	default:
		return fmt.Sprintf("java.lang.Object:%v", typed)
	}
}

// canonicalJavaMap renders a map with lexicographically sorted keys.
func canonicalJavaMap(values map[string]any) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var builder strings.Builder
	builder.WriteString("{")
	for _, key := range keys {
		builder.WriteString(canonicalJava(key))
		builder.WriteString(canonicalJava(values[key]))
	}
	builder.WriteString("}")
	return builder.String()
}

// canonicalJavaList renders a list in order.
func canonicalJavaList(values []any) string {
	var builder strings.Builder
	builder.WriteString("[")
	for _, item := range values {
		builder.WriteString(canonicalJava(item))
	}
	builder.WriteString("]")
	return builder.String()
}

func javaIntegralCanonical(value int64) string {
	if value >= -2147483648 && value <= 2147483647 {
		return "java.lang.Integer:" + strconv.FormatInt(value, 10)
	}
	return "java.lang.Long:" + strconv.FormatInt(value, 10)
}

// javaDoubleString mirrors Java's Double.toString minimum-one-decimal form for
// the integral values that reach hashes in practice.
func javaDoubleString(value float64) string {
	text := strconv.FormatFloat(value, 'g', -1, 64)
	if !strings.ContainsAny(text, ".eE") {
		text += ".0"
	}
	return text
}

func isSHA256Hex(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func copyAnyMap(source map[string]any) map[string]any {
	copied := make(map[string]any, len(source))
	for key, value := range source {
		copied[key] = value
	}
	return copied
}
