package domain

import "testing"

func TestFilter_BooleanAndIdentifier(t *testing.T) {
	n, err := ParseFilter("active == true")
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := n.Evaluate(&ClientMetadata{Active: true}); !ok {
		t.Error("active==true")
	}
	if ok, _ := n.Evaluate(&ClientMetadata{Active: false}); ok {
		t.Error("active!=false")
	}
}

func TestFilter_FloatNumbers(t *testing.T) {
	n, err := ParseFilter("state.x > 1.5")
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := n.Evaluate(&ClientMetadata{InnerState: map[string]interface{}{"x": 2.0}}); !ok {
		t.Error("2.0 > 1.5")
	}
}

func TestFilter_NegativeNumbers(t *testing.T) {
	n, err := ParseFilter("state.x < -5")
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := n.Evaluate(&ClientMetadata{InnerState: map[string]interface{}{"x": -10}}); !ok {
		t.Error("-10 < -5")
	}
}

func TestFilter_ContainsList(t *testing.T) {
	// Use a list-typed inner state field
	n, _ := ParseFilter("state.packages CONTAINS 'nginx'")
	c := &ClientMetadata{InnerState: map[string]interface{}{"packages": []interface{}{"nginx", "redis"}}}
	if ok, _ := n.Evaluate(c); !ok {
		t.Error("expected to find nginx in list")
	}
}

func TestFilter_EmptyList_In(t *testing.T) {
	n, _ := ParseFilter("os IN []")
	if ok, _ := n.Evaluate(&ClientMetadata{OS: "linux"}); ok {
		t.Error("empty list never matches")
	}
}

func TestParseFilter_ErrorPaths(t *testing.T) {
	// Bad operator
	if _, err := ParseFilter("os ~ 'a'"); err == nil {
		t.Error("expected error for ~")
	}
	// Bad value
	if _, err := ParseFilter("os ==  ,"); err == nil {
		t.Error("expected bad value error")
	}
}

func TestCompareValues_TypeCoercion(t *testing.T) {
	// String "3" can compare numerically against int 3
	got, err := CompareValues("3", OpEq, 3)
	if err != nil || !got {
		t.Errorf("expected numeric coercion eq: %v %v", got, err)
	}
	// Bool to float
	got, _ = CompareValues(true, OpEq, 1)
	if !got {
		t.Error("true == 1")
	}
	// nil handling
	got, _ = CompareValues(nil, OpEq, nil)
	if !got {
		t.Error("nil == nil")
	}
}
