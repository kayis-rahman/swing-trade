package strategy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// ConfigHasher mirrors Java's StrategyConfigHasher component: the SHA-256 of the canonical
// JSON of {strategyType, params, overlays} for the strategy_config params_hash
// column (plan section4.1/4.3). Canonical means lexicographically sorted keys
// at every map level; Go's encoding/json sorts map keys exactly that way, and
// the encoder runs with HTML escaping disabled to match Jackson's defaults.
func ConfigHasher(strategyType string, params, overlays map[string]any) (string, error) {
	canonical := map[string]any{
		"strategyType": strategyType,
		"params":       canonicalAnyMap(params),
		"overlays":     canonicalAnyMap(overlays),
	}
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(canonical); err != nil {
		return "", fmt.Errorf("encode canonical params: %w", err)
	}
	// Encoder.Encode appends a newline; Jackson's writeValueAsString does not.
	payload := bytes.TrimRight(buffer.Bytes(), "\n")
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

// DecodeParamsJSON decodes request params with json.Number so numeric
// spelling survives hashing exactly as Jackson's Integer/Long/Double/BigDecimal
// types would.
func DecodeParamsJSON(payload []byte) (map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	params := map[string]any{}
	if err := decoder.Decode(&params); err != nil {
		return nil, fmt.Errorf("decode params: %w", err)
	}
	return params, nil
}

// canonicalAnyMap converts nested maps for sorted canonical encoding; the
// shared map type keeps Go's lexicographic marshal order at every level.
func canonicalAnyMap(values map[string]any) any {
	if values == nil {
		return map[string]any{}
	}
	canonical := make(map[string]any, len(values))
	for key, value := range values {
		canonical[key] = canonicalValue(value)
	}
	return canonical
}

func canonicalValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return canonicalAnyMap(typed)
	case []any:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, canonicalValue(item))
		}
		return items
	case coredecimal.Decimal:
		// Jackson serializes BigDecimal as an unquoted number keeping its scale.
		return json.RawMessage(typed.String())
	case float64:
		// Jackson's Double keeps a minimum of one decimal place.
		text := strconv.FormatFloat(typed, 'g', -1, 64)
		if !strings.ContainsAny(text, ".eE") {
			text += ".0"
		}
		return json.RawMessage(text)
	default:
		return value
	}
}
