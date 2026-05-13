package domain

import (
	"fmt"
	"strconv"
	"strings"
)

type FilterOp string

const (
	OpEq          FilterOp = "=="
	OpNeq         FilterOp = "!="
	OpLt          FilterOp = "<"
	OpGt          FilterOp = ">"
	OpLte         FilterOp = "<="
	OpGte         FilterOp = ">="
	OpIn          FilterOp = "IN"
	OpNotIn       FilterOp = "NOT_IN"
	OpContains    FilterOp = "CONTAINS"
	OpNotContains FilterOp = "NOT_CONTAINS"
)

type FilterCondition struct {
	Field string
	Op    FilterOp
	Value interface{}
}

type LogicalOp string

const (
	LogicAnd LogicalOp = "AND"
	LogicOr  LogicalOp = "OR"
	LogicNot LogicalOp = "NOT"
)

type FilterNode struct {
	Condition *FilterCondition
	Logical   LogicalOp
	Left      *FilterNode
	Right     *FilterNode
}

func (n *FilterNode) Evaluate(client *ClientMetadata) (bool, error) {
	if n == nil {
		return true, nil
	}
	if n.Condition != nil {
		return n.evalCondition(client)
	}
	switch n.Logical {
	case LogicNot:
		v, err := n.Left.Evaluate(client)
		if err != nil {
			return false, err
		}
		return !v, nil
	case LogicAnd:
		l, err := n.Left.Evaluate(client)
		if err != nil {
			return false, err
		}
		if !l {
			return false, nil
		}
		return n.Right.Evaluate(client)
	case LogicOr:
		l, err := n.Left.Evaluate(client)
		if err != nil {
			return false, err
		}
		if l {
			return true, nil
		}
		return n.Right.Evaluate(client)
	}
	return false, ErrInvalidFilter
}

func (n *FilterNode) evalCondition(client *ClientMetadata) (bool, error) {
	c := n.Condition
	leftVal, err := client.GetField(c.Field)
	if err != nil {
		// Unknown field => doesn't match
		return false, nil
	}
	return CompareValues(leftVal, c.Op, c.Value)
}

func CompareValues(left interface{}, op FilterOp, right interface{}) (bool, error) {
	switch op {
	case OpEq:
		return equalsAny(left, right), nil
	case OpNeq:
		return !equalsAny(left, right), nil
	case OpLt, OpGt, OpLte, OpGte:
		return numericCompare(left, op, right)
	case OpIn:
		return inAny(left, right)
	case OpNotIn:
		v, err := inAny(left, right)
		if err != nil {
			return false, err
		}
		return !v, nil
	case OpContains:
		return containsAny(left, right)
	case OpNotContains:
		v, err := containsAny(left, right)
		if err != nil {
			return false, err
		}
		return !v, nil
	}
	return false, ErrInvalidOperator
}

func equalsAny(a, b interface{}) bool {
	if a == nil || b == nil {
		return a == b
	}
	// Try numeric equality
	af, aOk := toFloat(a)
	bf, bOk := toFloat(b)
	if aOk && bOk {
		return af == bf
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func numericCompare(a interface{}, op FilterOp, b interface{}) (bool, error) {
	af, aOk := toFloat(a)
	bf, bOk := toFloat(b)
	if !aOk || !bOk {
		return false, ErrTypeMismatch
	}
	switch op {
	case OpLt:
		return af < bf, nil
	case OpGt:
		return af > bf, nil
	case OpLte:
		return af <= bf, nil
	case OpGte:
		return af >= bf, nil
	}
	return false, ErrInvalidOperator
}

func inAny(a, b interface{}) (bool, error) {
	switch list := b.(type) {
	case []interface{}:
		for _, v := range list {
			if equalsAny(a, v) {
				return true, nil
			}
		}
		return false, nil
	case []string:
		s := fmt.Sprintf("%v", a)
		for _, v := range list {
			if v == s {
				return true, nil
			}
		}
		return false, nil
	}
	return false, ErrTypeMismatch
}

func containsAny(a, b interface{}) (bool, error) {
	as, aOk := a.(string)
	bs, bOk := b.(string)
	if aOk && bOk {
		return strings.Contains(as, bs), nil
	}
	if list, ok := a.([]interface{}); ok {
		for _, v := range list {
			if equalsAny(v, b) {
				return true, nil
			}
		}
		return false, nil
	}
	return false, ErrTypeMismatch
}

func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint64:
		return float64(n), true
	case string:
		f, err := strconv.ParseFloat(n, 64)
		if err == nil {
			return f, true
		}
	case bool:
		if n {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

type FilterResult struct {
	MatchingClientIDs []string
	TotalEvaluated    int
	MatchCount        int
}

func (r *FilterResult) IsEmpty() bool { return r.MatchCount == 0 }
func (r *FilterResult) GetMatchPercentage() float64 {
	if r.TotalEvaluated == 0 {
		return 0
	}
	return 100.0 * float64(r.MatchCount) / float64(r.TotalEvaluated)
}
