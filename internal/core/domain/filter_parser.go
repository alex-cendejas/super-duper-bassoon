package domain

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type tokenType int

const (
	tIdent tokenType = iota
	tString
	tNumber
	tOp
	tLParen
	tRParen
	tComma
	tLBracket
	tRBracket
	tEOF
)

type token struct {
	typ tokenType
	val string
}

func tokenize(input string) ([]token, error) {
	var tokens []token
	runes := []rune(input)
	i := 0
	for i < len(runes) {
		c := runes[i]
		switch {
		case unicode.IsSpace(c):
			i++
		case c == '(':
			tokens = append(tokens, token{tLParen, "("})
			i++
		case c == ')':
			tokens = append(tokens, token{tRParen, ")"})
			i++
		case c == '[':
			tokens = append(tokens, token{tLBracket, "["})
			i++
		case c == ']':
			tokens = append(tokens, token{tRBracket, "]"})
			i++
		case c == ',':
			tokens = append(tokens, token{tComma, ","})
			i++
		case c == '\'' || c == '"':
			end := c
			i++
			start := i
			for i < len(runes) && runes[i] != end {
				i++
			}
			if i >= len(runes) {
				return nil, fmt.Errorf("%w: unterminated string", ErrInvalidFilter)
			}
			tokens = append(tokens, token{tString, string(runes[start:i])})
			i++
		case unicode.IsDigit(c) || (c == '-' && i+1 < len(runes) && unicode.IsDigit(runes[i+1])):
			start := i
			if c == '-' {
				i++
			}
			for i < len(runes) && (unicode.IsDigit(runes[i]) || runes[i] == '.') {
				i++
			}
			tokens = append(tokens, token{tNumber, string(runes[start:i])})
		case c == '=' || c == '!' || c == '<' || c == '>':
			start := i
			i++
			if i < len(runes) && runes[i] == '=' {
				i++
			}
			tokens = append(tokens, token{tOp, string(runes[start:i])})
		case unicode.IsLetter(c) || c == '_':
			start := i
			for i < len(runes) && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '_' || runes[i] == '.') {
				i++
			}
			word := string(runes[start:i])
			upper := strings.ToUpper(word)
			switch upper {
			case "AND", "OR", "NOT", "IN", "NOT_IN", "CONTAINS", "NOT_CONTAINS":
				tokens = append(tokens, token{tOp, upper})
			case "TRUE", "FALSE":
				tokens = append(tokens, token{tIdent, strings.ToLower(word)})
			default:
				tokens = append(tokens, token{tIdent, word})
			}
		default:
			return nil, fmt.Errorf("%w: unexpected char %q", ErrInvalidFilter, c)
		}
	}
	tokens = append(tokens, token{tEOF, ""})
	return tokens, nil
}

type parser struct {
	tokens []token
	pos    int
}

func (p *parser) peek() token { return p.tokens[p.pos] }
func (p *parser) advance() token {
	t := p.tokens[p.pos]
	p.pos++
	return t
}

func ParseFilter(expr string) (*FilterNode, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, nil
	}
	toks, err := tokenize(expr)
	if err != nil {
		return nil, err
	}
	p := &parser{tokens: toks}
	node, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.peek().typ != tEOF {
		return nil, fmt.Errorf("%w: trailing tokens", ErrInvalidFilter)
	}
	return node, nil
}

func (p *parser) parseOr() (*FilterNode, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().typ == tOp && p.peek().val == "OR" {
		p.advance()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &FilterNode{Logical: LogicOr, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseAnd() (*FilterNode, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}
	for p.peek().typ == tOp && p.peek().val == "AND" {
		p.advance()
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &FilterNode{Logical: LogicAnd, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseNot() (*FilterNode, error) {
	if p.peek().typ == tOp && p.peek().val == "NOT" {
		p.advance()
		inner, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return &FilterNode{Logical: LogicNot, Left: inner}, nil
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (*FilterNode, error) {
	t := p.peek()
	if t.typ == tLParen {
		p.advance()
		n, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.peek().typ != tRParen {
			return nil, fmt.Errorf("%w: missing closing paren", ErrInvalidFilter)
		}
		p.advance()
		return n, nil
	}
	return p.parseCondition()
}

func (p *parser) parseCondition() (*FilterNode, error) {
	if p.peek().typ != tIdent {
		return nil, fmt.Errorf("%w: expected identifier", ErrInvalidFilter)
	}
	field := p.advance().val
	if p.peek().typ != tOp {
		return nil, fmt.Errorf("%w: expected operator", ErrInvalidFilter)
	}
	rawOp := p.advance().val
	op := FilterOp(rawOp)
	if !isKnownOp(op) {
		return nil, fmt.Errorf("%w: unknown operator %q", ErrInvalidFilter, rawOp)
	}
	value, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	return &FilterNode{Condition: &FilterCondition{Field: field, Op: op, Value: value}}, nil
}

func isKnownOp(op FilterOp) bool {
	switch op {
	case OpEq, OpNeq, OpLt, OpGt, OpLte, OpGte, OpIn, OpNotIn, OpContains, OpNotContains:
		return true
	}
	return false
}

func (p *parser) parseValue() (interface{}, error) {
	t := p.advance()
	switch t.typ {
	case tString:
		return t.val, nil
	case tNumber:
		if strings.Contains(t.val, ".") {
			f, err := strconv.ParseFloat(t.val, 64)
			if err != nil {
				return nil, err
			}
			return f, nil
		}
		n, err := strconv.ParseInt(t.val, 10, 64)
		if err != nil {
			return nil, err
		}
		return n, nil
	case tIdent:
		switch t.val {
		case "true":
			return true, nil
		case "false":
			return false, nil
		}
		return t.val, nil
	case tLBracket:
		var values []interface{}
		if p.peek().typ == tRBracket {
			p.advance()
			return values, nil
		}
		for {
			v, err := p.parseValue()
			if err != nil {
				return nil, err
			}
			values = append(values, v)
			if p.peek().typ == tComma {
				p.advance()
				continue
			}
			if p.peek().typ == tRBracket {
				p.advance()
				return values, nil
			}
			return nil, fmt.Errorf("%w: bad list", ErrInvalidFilter)
		}
	}
	return nil, fmt.Errorf("%w: bad value token %q", ErrInvalidFilter, t.val)
}
