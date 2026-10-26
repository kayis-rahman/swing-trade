package strategy

import (
	"strings"
	"testing"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

func validatorTestSchema(t *testing.T) ParamSchema {
	t.Helper()
	emaFastMin := mustParseDecisionDecimal(t, "5")
	emaFastMax := mustParseDecisionDecimal(t, "100")
	emaSlowMin := mustParseDecisionDecimal(t, "20")
	emaSlowMax := mustParseDecisionDecimal(t, "250")
	proximityMin := mustParseDecisionDecimal(t, "0.85")
	proximityMax := mustParseDecisionDecimal(t, "1")
	defs := []ParamDef{
		{Name: "emaFast", Type: ParamTypeInt, Min: &emaFastMin, Max: &emaFastMax, DefaultValue: 20, Description: "fast", Group: "trend"},
		{Name: "emaSlow", Type: ParamTypeInt, Min: &emaSlowMin, Max: &emaSlowMax, DefaultValue: 50, Description: "slow", Group: "trend"},
		{Name: "highProximity", Type: ParamTypeDecimal, Min: &proximityMin, Max: &proximityMax, DefaultValue: mustParseDecisionDecimal(t, "0.97"), Description: "proximity", Group: "trend"},
		{Name: "sentimentGate", Type: ParamTypeBool, DefaultValue: false, Description: "gate", Group: "overlay"},
		{Name: "mode", Type: ParamTypeEnum, DefaultValue: "STRICT", Description: "mode", Group: "scoring"},
	}
	return NewParamSchema(defs)
}

// Ported from ParamSchemaValidatorTest.
func TestParamValidatorFillsMissingKeysWithDefaults(t *testing.T) {
	result := ParamSchemaValidator{}.Validate(validatorTestSchema(t), nil, map[string]any{"emaFast": 10})
	if !result.Valid {
		t.Fatalf("result invalid: %v", result.Errors)
	}
	if result.ResolvedParams["emaFast"] != 10 || result.ResolvedParams["emaSlow"] != 50 ||
		result.ResolvedParams["sentimentGate"] != false || result.ResolvedParams["mode"] != "STRICT" {
		t.Fatalf("resolved params = %+v", result.ResolvedParams)
	}
}

func TestParamValidatorRejectsUnknownKeys(t *testing.T) {
	result := ParamSchemaValidator{}.Validate(validatorTestSchema(t), nil, map[string]any{"bogusKey": 1})
	if result.Valid {
		t.Fatal("unknown key accepted")
	}
	if !containsSubstringInAny(result.Errors, "Unknown param: bogusKey") {
		t.Fatalf("errors = %v", result.Errors)
	}
}

func TestParamValidatorRejectsOutOfRangeValues(t *testing.T) {
	result := ParamSchemaValidator{}.Validate(validatorTestSchema(t), nil, map[string]any{"emaFast": 1000})
	if result.Valid {
		t.Fatal("out-of-range value accepted")
	}
	found := false
	for _, message := range result.Errors {
		if strings.Contains(message, "emaFast") && strings.Contains(message, "maximum") {
			found = true
		}
	}
	if !found {
		t.Fatalf("errors = %v", result.Errors)
	}
}

func emaFastBelowSlowRule(t *testing.T) CrossFieldRule {
	t.Helper()
	rule, err := NewCrossFieldRule("emaFast must be less than emaSlow", func(params map[string]any) bool {
		return params["emaFast"].(int) < params["emaSlow"].(int)
	})
	if err != nil {
		t.Fatal(err)
	}
	return rule
}

func TestParamValidatorAppliesCrossFieldRules(t *testing.T) {
	result := ParamSchemaValidator{}.Validate(validatorTestSchema(t),
		[]CrossFieldRule{emaFastBelowSlowRule(t)},
		map[string]any{"emaFast": 60, "emaSlow": 50})
	if result.Valid {
		t.Fatal("violated cross-field rule accepted")
	}
	if !containsSubstringInAny(result.Errors, "emaFast must be less than emaSlow") {
		t.Fatalf("errors = %v", result.Errors)
	}
}

func TestParamValidatorAcceptsValidCrossFieldParams(t *testing.T) {
	result := ParamSchemaValidator{}.Validate(validatorTestSchema(t),
		[]CrossFieldRule{emaFastBelowSlowRule(t)},
		map[string]any{"emaFast": 20, "emaSlow": 50})
	if !result.Valid {
		t.Fatalf("valid params rejected: %v", result.Errors)
	}
}

// Extra pins for Java branches the suite relies on indirectly.

func TestParamValidatorTypeCoercionAndMismatchMessages(t *testing.T) {
	schema := NewParamSchema([]ParamDef{
		{Name: "ratio", Type: ParamTypeDecimal, DefaultValue: mustParseDecisionDecimal(t, "0.5")},
		{Name: "enabled", Type: ParamTypeBool, DefaultValue: false},
		{Name: "mode", Type: ParamTypeEnum, DefaultValue: "STRICT"},
	})

	result := ParamSchemaValidator{}.Validate(schema, nil, map[string]any{
		"ratio": "0.97", "enabled": "true", "mode": 7,
	})
	if !result.Valid {
		t.Fatalf("coercible values rejected: %v", result.Errors)
	}
	if decimalValue, ok := result.ResolvedParams["ratio"].(coredecimal.Decimal); !ok ||
		decimalValue.String() != "0.9700" {
		t.Fatalf("resolved ratio = %#v", result.ResolvedParams["ratio"])
	}
	if result.ResolvedParams["enabled"] != true || result.ResolvedParams["mode"] != "7" {
		t.Fatalf("resolved = %+v", result.ResolvedParams)
	}

	mismatch := ParamSchemaValidator{}.Validate(schema, nil, map[string]any{"ratio": "abc"})
	if mismatch.Valid || !containsSubstringInAny(mismatch.Errors, "ratio: expected DECIMAL, got 'abc'") {
		t.Fatalf("mismatch errors = %v", mismatch.Errors)
	}
	intSchema := NewParamSchema([]ParamDef{{Name: "count", Type: ParamTypeInt, DefaultValue: 1}})
	mismatch = ParamSchemaValidator{}.Validate(intSchema, nil, map[string]any{"count": "abc"})
	if mismatch.Valid || !containsSubstringInAny(mismatch.Errors, "count: expected INT, got 'abc'") {
		t.Fatalf("INT mismatch errors = %v", mismatch.Errors)
	}
}

func TestParamValidatorBooleanParsingMirrorsJava(t *testing.T) {
	schema := NewParamSchema([]ParamDef{{Name: "flag", Type: ParamTypeBool, DefaultValue: false}})
	for raw, want := range map[string]any{
		"true": true, "TRUE": true, "TrUe": true, "junk": false, "1": false,
	} {
		result := ParamSchemaValidator{}.Validate(schema, nil, map[string]any{"flag": raw})
		if !result.Valid || result.ResolvedParams["flag"] != want {
			t.Fatalf("flag=%q result = %+v", raw, result)
		}
	}
}

func TestParamValidatorRequiresNullValues(t *testing.T) {
	schema := NewParamSchema([]ParamDef{{Name: "required", Type: ParamTypeInt, DefaultValue: nil}})
	result := ParamSchemaValidator{}.Validate(schema, nil, nil)
	if result.Valid || !containsSubstringInAny(result.Errors, "required: value is required") {
		t.Fatalf("null default result = %+v", result)
	}
}

func TestParamValidatorSkipsCrossFieldRulesAfterKeyErrors(t *testing.T) {
	ruleCalled := false
	rule, err := NewCrossFieldRule("never", func(map[string]any) bool {
		ruleCalled = true
		return false
	})
	if err != nil {
		t.Fatal(err)
	}
	schema := NewParamSchema([]ParamDef{{Name: "count", Type: ParamTypeInt, DefaultValue: 1}})
	result := ParamSchemaValidator{}.Validate(schema, []CrossFieldRule{rule}, map[string]any{"bogus": 1})
	if result.Valid || ruleCalled {
		t.Fatalf("cross-field ran after key errors: %+v", result)
	}
}

func TestParamSchemaAndConstructorsValidate(t *testing.T) {
	if _, err := NewParamDef(" ", ParamTypeInt, nil, nil, 0, "", ""); err == nil ||
		err.Error() != "Param name cannot be null or blank" {
		t.Fatalf("param name validation = %v", err)
	}
	if _, err := NewParamDef("name", "", nil, nil, 0, "", ""); err == nil ||
		err.Error() != "Param type cannot be null" {
		t.Fatalf("param type validation = %v", err)
	}
	if _, err := NewCrossFieldRule(" ", func(map[string]any) bool { return true }); err == nil ||
		err.Error() != "message cannot be null or blank" {
		t.Fatalf("rule message validation = %v", err)
	}
	if _, err := NewCrossFieldRule("msg", nil); err == nil || err.Error() != "rule cannot be null" {
		t.Fatalf("rule nil validation = %v", err)
	}
	schema := NewParamSchema(nil)
	if len(schema.Params) != 0 {
		t.Fatalf("nil schema params = %v", schema.Params)
	}
	def := ParamDef{Name: "x", Type: ParamTypeInt}
	schema = NewParamSchema([]ParamDef{def})
	def.Name = "mutated"
	if schema.Params[0].Name != "x" {
		t.Fatal("schema params alias the input slice")
	}
	ok := ParamValidationOk(map[string]any{"a": 1})
	ok.ResolvedParams["a"] = 2
	source := map[string]any{"a": 1}
	ParamValidationOk(source)
	if source["a"] != 1 {
		t.Fatal("ok() aliases the caller's map")
	}
	failed := ParamValidationFailed([]string{"first"})
	failed.Errors = append(failed.Errors, "second")
	if len(failed.Errors) != 2 && failed.ResolvedParams == nil {
		t.Fatalf("failed result = %+v", failed)
	}
}

func containsSubstringInAny(messages []string, needle string) bool {
	for _, message := range messages {
		if strings.Contains(message, needle) {
			return true
		}
	}
	return false
}
