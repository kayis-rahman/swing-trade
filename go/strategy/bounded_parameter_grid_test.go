package strategy

import "testing"

func TestExpandParameterGridBasic(t *testing.T) {
	ranges := map[string][]any{
		"a": {1, 2},
		"b": {"x"},
	}
	res, err := ExpandParameterGrid(ranges, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 {
		t.Fatalf("got %d", len(res))
	}
}

func TestExpandParameterGridEmpty(t *testing.T) {
	res, err := ExpandParameterGrid(map[string][]any{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 {
		t.Fatalf("got %d", len(res))
	}
}

func TestExpandParameterGridExceeds(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	ranges := map[string][]any{"a": {1, 2, 3}}
	_, _ = ExpandParameterGrid(ranges, 2)
}

func TestWalkForwardEvaluationIsComplete(t *testing.T) {
	w := NewWalkForwardEvaluation([]Fold{{}}, 0, 0, 0)
	if !w.IsComplete(1) {
		t.Fatal("expected complete")
	}
}
