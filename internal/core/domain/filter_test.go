package domain

import "testing"

func TestParseFilter_Empty(t *testing.T) {
	n, err := ParseFilter("")
	if err != nil {
		t.Fatal(err)
	}
	if n != nil {
		t.Error("expected nil node for empty filter")
	}
	ok, _ := n.Evaluate(&ClientMetadata{})
	if !ok {
		t.Error("nil node should match everything")
	}
}

func TestParseFilter_Simple(t *testing.T) {
	n, err := ParseFilter("os == 'linux'")
	if err != nil {
		t.Fatal(err)
	}
	c := &ClientMetadata{OS: "linux"}
	ok, err := n.Evaluate(c)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected match")
	}
	c.OS = "windows"
	ok, _ = n.Evaluate(c)
	if ok {
		t.Error("expected no match")
	}
}

func TestParseFilter_Numeric(t *testing.T) {
	n, err := ParseFilter("state.config_version >= 2")
	if err != nil {
		t.Fatal(err)
	}
	c := &ClientMetadata{InnerState: map[string]interface{}{"config_version": 3.0}}
	ok, err := n.Evaluate(c)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected match")
	}
	c.InnerState["config_version"] = 1.0
	ok, _ = n.Evaluate(c)
	if ok {
		t.Error("expected no match")
	}
}

func TestParseFilter_AndOr(t *testing.T) {
	n, err := ParseFilter("os == 'linux' AND (state.power == 'on' OR state.config_version > 1)")
	if err != nil {
		t.Fatal(err)
	}
	c := &ClientMetadata{OS: "linux", InnerState: map[string]interface{}{"power": "on", "config_version": 0}}
	ok, err := n.Evaluate(c)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected match")
	}
	c.InnerState["power"] = "off"
	c.InnerState["config_version"] = 0
	ok, _ = n.Evaluate(c)
	if ok {
		t.Error("expected no match")
	}
	c.InnerState["config_version"] = 5
	ok, _ = n.Evaluate(c)
	if !ok {
		t.Error("expected match via config_version")
	}
}

func TestParseFilter_Not(t *testing.T) {
	n, err := ParseFilter("NOT os == 'linux'")
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := n.Evaluate(&ClientMetadata{OS: "linux"}); ok {
		t.Error("expected NOT to invert")
	}
	if ok, _ := n.Evaluate(&ClientMetadata{OS: "darwin"}); !ok {
		t.Error("expected NOT-of-linux to match darwin")
	}
}

func TestParseFilter_In(t *testing.T) {
	n, err := ParseFilter("os IN ['linux','darwin']")
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := n.Evaluate(&ClientMetadata{OS: "darwin"}); !ok {
		t.Error("expected darwin to be in list")
	}
	if ok, _ := n.Evaluate(&ClientMetadata{OS: "windows"}); ok {
		t.Error("expected windows to NOT be in list")
	}
}

func TestParseFilter_NotIn(t *testing.T) {
	n, _ := ParseFilter("os NOT_IN ['linux']")
	if ok, _ := n.Evaluate(&ClientMetadata{OS: "darwin"}); !ok {
		t.Error("NOT_IN failed")
	}
	if ok, _ := n.Evaluate(&ClientMetadata{OS: "linux"}); ok {
		t.Error("NOT_IN failed inverse")
	}
}

func TestParseFilter_Contains(t *testing.T) {
	n, _ := ParseFilter("os CONTAINS 'lin'")
	if ok, _ := n.Evaluate(&ClientMetadata{OS: "linux"}); !ok {
		t.Error("expected CONTAINS match")
	}
	n, _ = ParseFilter("os NOT_CONTAINS 'lin'")
	if ok, _ := n.Evaluate(&ClientMetadata{OS: "windows"}); !ok {
		t.Error("expected NOT_CONTAINS match")
	}
}

func TestParseFilter_LabelsAndUnknownField(t *testing.T) {
	n, _ := ParseFilter("labels.env == 'prod'")
	c := &ClientMetadata{Labels: map[string]string{"env": "prod"}}
	if ok, _ := n.Evaluate(c); !ok {
		t.Error("expected label match")
	}
	n, _ = ParseFilter("unknown_field == 'x'")
	if ok, _ := n.Evaluate(c); ok {
		t.Error("unknown field should not match")
	}
}

func TestParseFilter_Errors(t *testing.T) {
	bad := []string{
		"os ==",
		"(os == 'linux'",
		"os = 'linux'",
		"@@ bad",
		"os == 'unterminated",
	}
	for _, b := range bad {
		if _, err := ParseFilter(b); err == nil {
			t.Errorf("expected error for %q", b)
		}
	}
}

func TestFilterResult_Methods(t *testing.T) {
	r := &FilterResult{TotalEvaluated: 0}
	if r.GetMatchPercentage() != 0 {
		t.Error("0/0 should be 0")
	}
	r.TotalEvaluated = 10
	r.MatchCount = 5
	if r.GetMatchPercentage() != 50 {
		t.Errorf("got %v", r.GetMatchPercentage())
	}
	if r.IsEmpty() {
		t.Error("not empty")
	}
	r.MatchCount = 0
	if !r.IsEmpty() {
		t.Error("should be empty")
	}
}

func TestCompareValues_TypeMismatch(t *testing.T) {
	if _, err := CompareValues("a", OpLt, "b"); err == nil {
		t.Error("expected error for string LT comparison")
	}
}

func TestClientMetadata_GetField(t *testing.T) {
	c := &ClientMetadata{
		ClientID: "id1",
		OS:       "linux",
		Active:   true,
		Labels:   map[string]string{"env": "prod"},
		InnerState: map[string]interface{}{
			"power": "on",
			"nested": map[string]interface{}{"x": 1},
		},
	}
	got, err := c.GetField("client_id")
	if err != nil || got != "id1" {
		t.Errorf("client_id: %v %v", got, err)
	}
	got, err = c.GetField("id")
	if err != nil || got != "id1" {
		t.Errorf("id alias: %v %v", got, err)
	}
	if got, err := c.GetField("active"); err != nil || got != true {
		t.Errorf("active failed: %v %v", got, err)
	}
	if got, err := c.GetField("labels.env"); err != nil || got != "prod" {
		t.Errorf("labels.env: %v %v", got, err)
	}
	if _, err := c.GetField("labels.missing"); err == nil {
		t.Error("expected unknown")
	}
	if got, err := c.GetField("state.power"); err != nil || got != "on" {
		t.Errorf("state.power: %v %v", got, err)
	}
	if got, err := c.GetField("state.nested.x"); err != nil || got.(int) != 1 {
		t.Errorf("nested: %v %v", got, err)
	}
	if _, err := c.GetField("state.nested.missing"); err == nil {
		t.Error("expected missing")
	}
	if _, err := c.GetField("bogus"); err == nil {
		t.Error("expected error")
	}
}
