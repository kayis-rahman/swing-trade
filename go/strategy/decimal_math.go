package strategy

import (
	"math/big"
	"strings"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// decimalProduct mirrors BigDecimal.multiply at the strategy layer.
func decimalProduct(left, right coredecimal.Decimal) coredecimal.Decimal {
	return ratToCoreDecimal(new(big.Rat).Mul(ratOf(left), ratOf(right)))
}

// decimalQuotientRounded mirrors BigDecimal.divide(other, scale, HALF_UP).
func decimalQuotientRounded(numerator, denominator coredecimal.Decimal, scale int) coredecimal.Decimal {
	if denominator.Cmp(coredecimal.Decimal{}) == 0 {
		return coredecimal.Decimal{}
	}
	quotient := new(big.Rat).Quo(ratOf(numerator), ratOf(denominator))
	rounded, err := coredecimal.Parse(quotient.FloatString(scale))
	if err != nil {
		return coredecimal.Decimal{}
	}
	return rounded
}

// absDecimal returns the absolute value.
func absDecimal(value coredecimal.Decimal) coredecimal.Decimal {
	if value.Cmp(coredecimal.Decimal{}) < 0 {
		return coredecimal.Decimal{}.Sub(value)
	}
	return value
}

// ratToCoreDecimal converts a rational to the fixed financial scale.
func ratToCoreDecimal(value *big.Rat) coredecimal.Decimal {
	decimalValue, err := coredecimal.Parse(value.FloatString(coredecimal.MoneyScale))
	if err != nil {
		return coredecimal.Decimal{}
	}
	return decimalValue
}

// scoreRules mirrors Java's per-strategy score: passed weight divided by total
// weight at scale 6 HALF_UP, or zero when every rule is weightless.
func scoreRules(rules []RuleOutcome) coredecimal.Decimal {
	zero := coredecimal.Decimal{}
	total := zero
	passed := zero
	for _, rule := range rules {
		total = total.Add(rule.Weight)
		if rule.Passed {
			passed = passed.Add(rule.Weight)
		}
	}
	if total.Cmp(zero) == 0 {
		return zero
	}
	return decimalQuotientRounded(passed, total, 6)
}

// ruleOutcome mirrors Java's new RuleOutcome; strategy code only builds valid
// outcomes, so the validation error is impossible here.
func ruleOutcome(key string, passed bool, mandatory bool, detail string) RuleOutcome {
	outcome, err := NewRuleOutcome(key, passed, oneDecimal(), mandatory, detail)
	if err != nil {
		panic(err)
	}
	return outcome
}

// oneDecimal returns BigDecimal.ONE at the financial scale.
func oneDecimal() coredecimal.Decimal {
	one, _ := coredecimal.Parse("1")
	return one
}

// zeroDecimal returns BigDecimal.ZERO at the financial scale.
func zeroDecimal() coredecimal.Decimal {
	return coredecimal.Decimal{}
}

// failedRuleKeys mirrors Java's reasoning suffix: the failed rule keys joined
// with ", ".
func failedRuleKeys(rules []RuleOutcome) string {
	keys := make([]string, 0, len(rules))
	for _, rule := range rules {
		if !rule.Passed {
			keys = append(keys, rule.Key)
		}
	}
	return strings.Join(keys, ", ")
}

// allMandatoryPassed mirrors Java's mandatory-rule conjunction.
func allMandatoryPassed(rules []RuleOutcome) bool {
	for _, rule := range rules {
		if rule.Mandatory && !rule.Passed {
			return false
		}
	}
	return true
}
