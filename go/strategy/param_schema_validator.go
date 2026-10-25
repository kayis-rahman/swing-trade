package strategy

import (
	"fmt"
	"strconv"
	"strings"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// ParamSchemaValidator mirrors Java's component: validates a raw params map
// against a schema and a type's cross-field rules (plan §4.3) - unknown keys
// rejected, per-key type and range checks, missing keys filled from defaults,
// cross-field rules last against the resolved map.
type ParamSchemaValidator struct{}

// Validate mirrors ParamSchemaValidator.validate. Resolved values keep Go
// types per parameter class: INT becomes int, DECIMAL becomes
// coredecimal.Decimal at financial scale, BOOL becomes bool, and ENUM stays a
// string, matching the classes Java's rules read back out of the map.
func (ParamSchemaValidator) Validate(
	schema ParamSchema,
	crossFieldRules []CrossFieldRule,
	rawParams map[string]any,
) ParamValidationResult {
	var errors []string
	input := rawParams
	if input == nil {
		input = map[string]any{}
	}

	known := make(map[string]struct{}, len(schema.Params))
	for _, def := range schema.Params {
		known[def.Name] = struct{}{}
	}
	for key := range input {
		if _, ok := known[key]; !ok {
			errors = append(errors, "Unknown param: "+key)
		}
	}

	resolved := make(map[string]any, len(schema.Params))
	for _, def := range schema.Params {
		rawValue, present := input[def.Name]
		if !present {
			rawValue = def.DefaultValue
		}
		resolved[def.Name] = coerceAndValidate(def, rawValue, &errors)
	}

	if len(errors) > 0 {
		return ParamValidationFailed(errors)
	}

	for _, rule := range crossFieldRules {
		if !rule.Test(resolved) {
			errors = append(errors, rule.Message)
		}
	}
	if len(errors) > 0 {
		return ParamValidationFailed(errors)
	}
	return ParamValidationOk(resolved)
}

func coerceAndValidate(def ParamDef, rawValue any, errors *[]string) any {
	if rawValue == nil {
		*errors = append(*errors, def.Name+": value is required")
		return nil
	}
	switch def.Type {
	case ParamTypeInt:
		intValue, err := toInt(rawValue)
		if err != nil {
			*errors = append(*errors, fmt.Sprintf("%s: expected %s, got '%v'", def.Name, def.Type, rawValue))
			return rawValue
		}
		if bound, err := intToDecimal(intValue); err == nil {
			checkRange(def, bound, errors)
		}
		return intValue
	case ParamTypeDecimal:
		decimalValue, err := toDecimal(rawValue)
		if err != nil {
			*errors = append(*errors, fmt.Sprintf("%s: expected %s, got '%v'", def.Name, def.Type, rawValue))
			return rawValue
		}
		checkRange(def, decimalValue, errors)
		return decimalValue
	case ParamTypeBool:
		return toBoolean(rawValue)
	case ParamTypeEnum:
		return fmt.Sprintf("%v", rawValue)
	default:
		return rawValue
	}
}

// toInt mirrors Java's toInt: Numbers truncate toward zero (like
// intValue()), anything else parses as a decimal integer.
func toInt(value any) (int, error) {
	switch typed := value.(type) {
	case int:
		return typed, nil
	case int32:
		return int(typed), nil
	case int64:
		return int(typed), nil
	case float64:
		return int(typed), nil
	case float32:
		return int(typed), nil
	default:
		parsed, err := strconv.ParseInt(strings.TrimSpace(fmt.Sprintf("%v", typed)), 10, 64)
		if err != nil {
			return 0, err
		}
		return int(parsed), nil
	}
}

// toDecimal mirrors Java's toDecimal over BigDecimal and Number inputs;
// non-numeric values parse from their text form at financial scale.
func toDecimal(value any) (coredecimal.Decimal, error) {
	switch typed := value.(type) {
	case coredecimal.Decimal:
		return typed, nil
	case int:
		return intToDecimal(typed)
	case int64:
		return intToDecimal(int(typed))
	case float64:
		return coredecimal.Parse(strconv.FormatFloat(typed, 'f', -1, 64))
	default:
		return coredecimal.Parse(strings.TrimSpace(fmt.Sprintf("%v", typed)))
	}
}

// toBoolean mirrors Java's Boolean.parseBoolean: only the string "true"
// (ignoring case) or a true boolean is true; everything else is false.
func toBoolean(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(typed, "true")
	default:
		return false
	}
}

func intToDecimal(value int) (coredecimal.Decimal, error) {
	return coredecimal.Parse(strconv.Itoa(value))
}

// checkRange mirrors Java's min/max comparison with inclusive bounds. The Go
// rendering uses financial-scale strings where Java prints plain BigDecimal
// text; the Java tests only pin substring contracts (name and bound keyword).
func checkRange(def ParamDef, value coredecimal.Decimal, errors *[]string) {
	if def.Min != nil && value.Cmp(*def.Min) < 0 {
		*errors = append(*errors, fmt.Sprintf("%s: %s is below minimum %s", def.Name, value.String(), def.Min.String()))
	}
	if def.Max != nil && value.Cmp(*def.Max) > 0 {
		*errors = append(*errors, fmt.Sprintf("%s: %s is above maximum %s", def.Name, value.String(), def.Max.String()))
	}
}
