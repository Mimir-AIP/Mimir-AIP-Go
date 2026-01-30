package Data_Processing

import (
	"fmt"
	"strconv"
	"strings"
)

// Very small expression evaluator for filter expressions.
// Supports identifiers (field names), numeric literals, string literals (single or double quoted),
// comparison operators == != > >= < <=, logical AND, OR, NOT, and parentheses.

type tokenType int

const (
	tokEOF tokenType = iota
	tokLParen
	tokRParen
	tokAnd
	tokOr
	tokNot
	tokOp
	tokIdent
	tokNumber
	tokString
	tokComma
)

type token struct {
	typ tokenType
	val string
}

func tokenize(input string) ([]token, error) {
	s := strings.TrimSpace(input)
	var toks []token
	for i := 0; i < len(s); {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			i++
			continue
		}
		if c == '(' {
			toks = append(toks, token{tokLParen, "("})
			i++
			continue
		}
		if c == ')' {
			toks = append(toks, token{tokRParen, ")"})
			i++
			continue
		}
		// two-char ops
		if i+1 < len(s) {
			two := s[i : i+2]
			if two == ">=" || two == "<=" || two == "==" || two == "!=" {
				toks = append(toks, token{tokOp, two})
				i += 2
				continue
			}
		}
		if c == '>' || c == '<' {
			toks = append(toks, token{tokOp, string(c)})
			i++
			continue
		}
		// identifiers and keywords
		if isAlpha(c) {
			j := i + 1
			for j < len(s) && (isAlphaNum(s[j]) || s[j] == '.' || s[j] == '_') {
				j++
			}
			word := s[i:j]
			lw := strings.ToUpper(word)
			if lw == "AND" {
				toks = append(toks, token{tokAnd, "AND"})
			} else if lw == "OR" {
				toks = append(toks, token{tokOr, "OR"})
			} else if lw == "NOT" {
				toks = append(toks, token{tokNot, "NOT"})
			} else if lw == "CONTAINS" || lw == "STARTSWITH" || lw == "MATCHES" {
				// treat these as operators
				toks = append(toks, token{tokOp, lw})
			} else {
				toks = append(toks, token{tokIdent, word})
			}
			i = j
			continue
		}
		// numbers
		if (c >= '0' && c <= '9') || c == '.' {
			j := i + 1
			for j < len(s) && ((s[j] >= '0' && s[j] <= '9') || s[j] == '.') {
				j++
			}
			toks = append(toks, token{tokNumber, s[i:j]})
			i = j
			continue
		}
		// strings
		if c == '\'' || c == '"' {
			quote := c
			j := i + 1
			for j < len(s) && s[j] != quote {
				j++
			}
			if j >= len(s) {
				return nil, fmt.Errorf("unterminated string")
			}
			toks = append(toks, token{tokString, s[i+1 : j]})
			i = j + 1
			continue
		}
		// operators like = (single) treat as ==
		if c == '=' {
			toks = append(toks, token{tokOp, "=="})
			i++
			continue
		}
		if c == ',' {
			toks = append(toks, token{tokComma, ","})
			i++
			continue
		}
		return nil, fmt.Errorf("unexpected character '%c'", c)
	}
	toks = append(toks, token{tokEOF, ""})
	return toks, nil
}

func isAlpha(c byte) bool    { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }
func isAlphaNum(c byte) bool { return isAlpha(c) || (c >= '0' && c <= '9') }

// parse -> convert to RPN using shunting-yard, then evaluate per row

func toRPN(toks []token) ([]token, error) {
	var out []token
	var opstack []token
	prec := func(t token) int {
		if t.typ == tokNot {
			return 5
		}
		if t.typ == tokOp {
			return 4
		}
		if t.typ == tokAnd {
			return 2
		}
		if t.typ == tokOr {
			return 1
		}
		return 0
	}
	isRightAssoc := func(t token) bool { return t.typ == tokNot }

	for i := 0; i < len(toks); i++ {
		t := toks[i]
		switch t.typ {
		case tokNumber, tokIdent, tokString:
			out = append(out, t)
		case tokOp, tokAnd, tokOr, tokNot:
			for len(opstack) > 0 {
				top := opstack[len(opstack)-1]
				if (prec(top) > prec(t)) || (prec(top) == prec(t) && !isRightAssoc(t)) {
					out = append(out, top)
					opstack = opstack[:len(opstack)-1]
					continue
				}
				break
			}
			opstack = append(opstack, t)
		case tokComma:
			// pop until left paren (argument separator)
			for len(opstack) > 0 && opstack[len(opstack)-1].typ != tokLParen {
				out = append(out, opstack[len(opstack)-1])
				opstack = opstack[:len(opstack)-1]
			}
		case tokLParen:
			opstack = append(opstack, t)
		case tokRParen:
			found := false
			for len(opstack) > 0 {
				top := opstack[len(opstack)-1]
				opstack = opstack[:len(opstack)-1]
				if top.typ == tokLParen {
					found = true
					break
				}
				out = append(out, top)
			}
			if !found {
				return nil, fmt.Errorf("mismatched parentheses")
			}
		case tokEOF:
			break
		}
	}
	for len(opstack) > 0 {
		top := opstack[len(opstack)-1]
		opstack = opstack[:len(opstack)-1]
		if top.typ == tokLParen || top.typ == tokRParen {
			return nil, fmt.Errorf("mismatched parentheses")
		}
		out = append(out, top)
	}
	return out, nil
}

func evalRPN(rpn []token, row map[string]any) (bool, error) {
	var stack []any
	push := func(v any) { stack = append(stack, v) }
	pop := func() any { v := stack[len(stack)-1]; stack = stack[:len(stack)-1]; return v }

	for _, t := range rpn {
		switch t.typ {
		case tokNumber:
			if f, err := strconv.ParseFloat(t.val, 64); err == nil {
				push(f)
			} else {
				return false, err
			}
		case tokString:
			push(t.val)
		case tokIdent:
			// lookup in row
			if v, ok := row[t.val]; ok {
				push(v)
			} else {
				push(nil)
			}
		case tokOp:
			// comparison needs two operands
			b := pop()
			a := pop()
			res, err := compare(a, b, t.val)
			if err != nil {
				return false, err
			}
			push(res)
		case tokNot:
			v := pop()
			bv := toBool(v)
			push(!bv)
		case tokAnd:
			b := pop()
			a := pop()
			push(toBool(a) && toBool(b))
		case tokOr:
			b := pop()
			a := pop()
			push(toBool(a) || toBool(b))
		default:
			return false, fmt.Errorf("unexpected token in RPN: %v", t)
		}
	}
	if len(stack) != 1 {
		return false, fmt.Errorf("invalid expression evaluation")
	}
	return toBool(stack[0]), nil
}

func compare(a, b any, op string) (bool, error) {
	// try numeric
	af, aok := toFloat64OK(a)
	bf, bok := toFloat64OK(b)
	if aok && bok {
		switch op {
		case "==":
			return af == bf, nil
		case "!=":
			return af != bf, nil
		case ">":
			return af > bf, nil
		case "<":
			return af < bf, nil
		case ">=":
			return af >= bf, nil
		case "<=":
			return af <= bf, nil
		}
	}
	// fallback to string compare
	sa := fmt.Sprintf("%v", a)
	sb := fmt.Sprintf("%v", b)
	switch op {
	case "==":
		return sa == sb, nil
	case "!=":
		return sa != sb, nil
	}
	return false, fmt.Errorf("unsupported comparison for op %s", op)
}

func toFloat64OK(v any) (float64, bool) {
	if v == nil {
		return 0, false
	}
	switch t := v.(type) {
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case float32:
		return float64(t), true
	case float64:
		return t, true
	case string:
		if f, err := strconv.ParseFloat(t, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

func toBool(v any) bool {
	if v == nil {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case int:
		return t != 0
	case int64:
		return t != 0
	case float32:
		return t != 0
	case float64:
		return t != 0
	case string:
		lw := strings.ToLower(t)
		return lw == "true" || lw == "1"
	default:
		return true
	}
}

// EvaluateExpression evaluates expression against a row map
func EvaluateExpression(expr string, row map[string]any) (bool, error) {
	toks, err := tokenize(expr)
	if err != nil {
		return false, err
	}
	rpn, err := toRPN(toks)
	if err != nil {
		return false, err
	}
	return evalRPN(rpn, row)
}
