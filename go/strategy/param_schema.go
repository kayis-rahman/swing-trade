package strategy

import (
	"fmt"
	"strings"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// ParamType mirrors Java's ParamType enum: the primitive type driving UI form
// generation and validation.
type ParamType string

// Parameter primitive types matching Java's enum constants.
const (
	ParamTypeInt     ParamType = "INT"
	ParamTypeDecimal ParamType = "DECIMAL"
	ParamTypeBool    ParamType = "BOOL"
	ParamTypeEnum    ParamType = "ENUM"
)

// ParamDef mirrors Java's ParamDef record: one configurable parameter's type,
// inclusive range, default, and UI metadata.
type ParamDef struct {
	Name         string
	Type         ParamType
	Min          *coredecimal.Decimal
	Max          *coredecimal.Decimal
	DefaultValue any
	Description  string
	Group        string
}

// NewParamDef mirrors the Java compact constructor's validation.
func NewParamDef(
	name string,
	paramType ParamType,
	min *coredecimal.Decimal,
	max *coredecimal.Decimal,
	defaultValue any,
	description string,
	group string,
) (ParamDef, error) {
	if strings.TrimSpace(name) == "" {
		return ParamDef{}, fmt.Errorf("Param name cannot be null or blank") //nolint:staticcheck // exact Java message
	}
	if paramType == "" {
		return ParamDef{}, fmt.Errorf("Param type cannot be null") //nolint:staticcheck // exact Java message
	}
	return ParamDef{
		Name: name, Type: paramType, Min: min, Max: max,
		DefaultValue: defaultValue, Description: description, Group: group,
	}, nil
}

// ParamSchema mirrors Java's ParamSchema record: the full parameter set a
// strategy type accepts, defensively copied.
type ParamSchema struct {
	Params []ParamDef
}

// NewParamSchema mirrors the Java compact constructor: nil becomes an empty
// immutable list and the slice is copied.
func NewParamSchema(params []ParamDef) ParamSchema {
	return ParamSchema{Params: append([]ParamDef{}, params...)}
}

// ParamValidationResult mirrors Java's record: either a list of errors or the
// resolved params map with defaults filled in for omitted keys (plan §4.3).
type ParamValidationResult struct {
	Valid          bool
	Errors         []string
	ResolvedParams map[string]any
}

// ParamValidationOk mirrors ParamValidationResult.ok (copies the map).
func ParamValidationOk(resolvedParams map[string]any) ParamValidationResult {
	resolved := make(map[string]any, len(resolvedParams))
	for key, value := range resolvedParams {
		resolved[key] = value
	}
	return ParamValidationResult{Valid: true, Errors: []string{}, ResolvedParams: resolved}
}

// ParamValidationFailed mirrors ParamValidationResult.failed.
func ParamValidationFailed(errors []string) ParamValidationResult {
	return ParamValidationResult{Valid: false, Errors: append([]string{}, errors...), ResolvedParams: map[string]any{}}
}

// CrossFieldRule mirrors Java's CrossFieldRule record: a multi-param
// predicate evaluated last against the fully-resolved map (plan §4.3).
type CrossFieldRule struct {
	Message string
	Rule    func(params map[string]any) bool
}

// NewCrossFieldRule mirrors the Java compact constructor's validation.
func NewCrossFieldRule(message string, rule func(params map[string]any) bool) (CrossFieldRule, error) {
	if strings.TrimSpace(message) == "" {
		return CrossFieldRule{}, fmt.Errorf("message cannot be null or blank")
	}
	if rule == nil {
		return CrossFieldRule{}, fmt.Errorf("rule cannot be null")
	}
	return CrossFieldRule{Message: message, Rule: rule}, nil
}

// Test reports whether the resolved params satisfy the rule.
func (rule CrossFieldRule) Test(params map[string]any) bool {
	return rule.Rule(params)
}
