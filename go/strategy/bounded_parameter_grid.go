package strategy

import (
	"fmt"
	"sort"
	"strings"
)

// ExpandParameterGrid mirrors Java's deterministic, bounded Cartesian product.
func ExpandParameterGrid(ranges map[string][]any, maxCandidates int) ([]map[string]any, error) {
	if len(ranges) == 0 {
		return []map[string]any{{}}, nil
	}
	if maxCandidates < 1 {
		return nil, fmt.Errorf("maxCandidates must be positive")
	}
	keys := make([]string, 0, len(ranges))
	for k := range ranges {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := make([]map[string]any, 0)
	var rec func(index int, current map[string]any) error
	rec = func(index int, current map[string]any) error {
		if len(result) >= maxCandidates {
			return fmt.Errorf("parameter grid exceeds maxCandidates=%d", maxCandidates)
		}
		if index == len(keys) {
			cp := make(map[string]any, len(current))
			for k, v := range current {
				cp[k] = v
			}
			result = append(result, cp)
			return nil
		}
		k := keys[index]
		vals := ranges[k]
		if strings.TrimSpace(k) == "" {
			return fmt.Errorf("parameter names and values are required")
		}
		if len(vals) == 0 {
			return fmt.Errorf("parameter names and values are required")
		}
		for _, v := range vals {
			if v == nil {
				return fmt.Errorf("parameter values cannot be null")
			}
			current[k] = v
			if err := rec(index+1, current); err != nil {
				return err
			}
		}
		delete(current, k)
		return nil
	}
	if err := rec(0, map[string]any{}); err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(result))
	copy(out, result)
	return out, nil
}
