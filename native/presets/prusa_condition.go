package presets

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// compatible_printers_condition is a small PrusaSlicer expression. This
// evaluator understands ==, !=, =~, !~, and, or, parentheses, double-quoted
// strings, numbers, and setting names with an optional list index such as
// name[0]. Regular expressions use the /pattern/ form and Go's RE2 engine, which
// runs in linear time. Anything else is an error. A preset whose condition
// errors is treated as incompatible, never guessed at.

const maxPrusaConditionBytes = 2048
const maxPrusaConditionTokens = 256
const maxPrusaRegexBytes = 256

type prusaTokenKind int

const (
	tokEnd prusaTokenKind = iota
	tokLParen
	tokRParen
	tokCompare
	tokAnd
	tokOr
	tokString
	tokRegex
	tokNumber
	tokSetting
)

type prusaToken struct {
	kind  prusaTokenKind
	text  string
	index int // list index for tokSetting; -1 when not indexed
}

func evalPrusaCondition(expr string, values prusaSection) (bool, error) {
	tokens, err := tokenizePrusaCondition(strings.TrimSpace(expr))
	if err != nil {
		return false, err
	}
	parser := &prusaConditionParser{tokens: tokens, values: values}
	result, err := parser.orExpr()
	if err != nil {
		return false, err
	}
	if parser.peek().kind != tokEnd {
		return false, fmt.Errorf("unexpected token in compatibility condition")
	}
	return result, nil
}

func tokenizePrusaCondition(expr string) ([]prusaToken, error) {
	if expr == "" {
		return nil, fmt.Errorf("compatibility condition is empty")
	}
	if len(expr) > maxPrusaConditionBytes {
		return nil, fmt.Errorf("compatibility condition exceeds %d bytes", maxPrusaConditionBytes)
	}
	tokens := []prusaToken{}
	for i := 0; i < len(expr); {
		c := expr[i]
		var token prusaToken
		switch {
		case c == ' ' || c == '\t':
			i++
			continue
		case c == '(':
			token, i = prusaToken{kind: tokLParen}, i+1
		case c == ')':
			token, i = prusaToken{kind: tokRParen}, i+1
		case (c == '=' || c == '!') && i+1 < len(expr) && (expr[i+1] == '=' || expr[i+1] == '~'):
			token, i = prusaToken{kind: tokCompare, text: expr[i : i+2]}, i+2
		case c == '/':
			end, closed := i+1, false
			for end < len(expr) && !closed {
				switch expr[end] {
				case '\\':
					end++
				case '/':
					closed = true
				}
				if !closed {
					end++
				}
			}
			if !closed || end-i-1 > maxPrusaRegexBytes {
				return nil, fmt.Errorf("compatibility condition has an unterminated or oversized regular expression")
			}
			token, i = prusaToken{kind: tokRegex, text: expr[i+1 : end]}, end+1
		case c == '"':
			end := strings.IndexByte(expr[i+1:], '"')
			if end < 0 {
				return nil, fmt.Errorf("compatibility condition has an unterminated string")
			}
			token, i = prusaToken{kind: tokString, text: expr[i+1 : i+1+end]}, i+end+2
		case isPrusaDigit(c):
			end := i
			for end < len(expr) && (isPrusaDigit(expr[end]) || expr[end] == '.') {
				end++
			}
			token, i = prusaToken{kind: tokNumber, text: expr[i:end]}, end
		case isPrusaWordStart(c):
			end := i
			for end < len(expr) && (isPrusaWordStart(expr[end]) || isPrusaDigit(expr[end])) {
				end++
			}
			word := expr[i:end]
			switch word {
			case "and":
				token = prusaToken{kind: tokAnd}
			case "or":
				token = prusaToken{kind: tokOr}
			default:
				token = prusaToken{kind: tokSetting, text: word, index: -1}
				if end < len(expr) && expr[end] == '[' {
					closing := strings.IndexByte(expr[end:], ']')
					if closing < 0 {
						return nil, fmt.Errorf("compatibility condition has an unterminated index")
					}
					index, err := strconv.Atoi(expr[end+1 : end+closing])
					if err != nil || index < 0 || index > 63 {
						return nil, fmt.Errorf("compatibility condition has an unsupported index")
					}
					token.index = index
					end += closing + 1
				}
			}
			i = end
		default:
			return nil, fmt.Errorf("compatibility condition uses an unsupported operator or character")
		}
		tokens = append(tokens, token)
		if len(tokens) > maxPrusaConditionTokens {
			return nil, fmt.Errorf("compatibility condition has more than %d tokens", maxPrusaConditionTokens)
		}
	}
	return append(tokens, prusaToken{kind: tokEnd}), nil
}

func isPrusaDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isPrusaWordStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

type prusaConditionParser struct {
	tokens []prusaToken
	pos    int
	values prusaSection
}

func (p *prusaConditionParser) peek() prusaToken {
	return p.tokens[p.pos]
}

func (p *prusaConditionParser) next() prusaToken {
	token := p.tokens[p.pos]
	if token.kind != tokEnd {
		p.pos++
	}
	return token
}

// Both sides of every operator are always parsed, so syntax errors cannot hide
// behind a short-circuit.
func (p *prusaConditionParser) orExpr() (bool, error) {
	left, err := p.andExpr()
	if err != nil {
		return false, err
	}
	for p.peek().kind == tokOr {
		p.next()
		right, err := p.andExpr()
		if err != nil {
			return false, err
		}
		left = left || right
	}
	return left, nil
}

func (p *prusaConditionParser) andExpr() (bool, error) {
	left, err := p.primary()
	if err != nil {
		return false, err
	}
	for p.peek().kind == tokAnd {
		p.next()
		right, err := p.primary()
		if err != nil {
			return false, err
		}
		left = left && right
	}
	return left, nil
}

func (p *prusaConditionParser) primary() (bool, error) {
	if p.peek().kind == tokLParen {
		p.next()
		value, err := p.orExpr()
		if err != nil {
			return false, err
		}
		if p.next().kind != tokRParen {
			return false, fmt.Errorf("compatibility condition is missing a closing parenthesis")
		}
		return value, nil
	}
	left, err := p.operand()
	if err != nil {
		return false, err
	}
	operator := p.next()
	if operator.kind != tokCompare {
		return false, fmt.Errorf("compatibility condition needs a comparison")
	}
	if operator.text == "=~" || operator.text == "!~" {
		pattern := p.next()
		if pattern.kind != tokRegex {
			return false, fmt.Errorf("regular expression comparison needs a /pattern/")
		}
		re, err := regexp.Compile(pattern.text)
		if err != nil {
			return false, fmt.Errorf("compatibility condition has an invalid regular expression")
		}
		matched := re.MatchString(left)
		if operator.text == "!~" {
			return !matched, nil
		}
		return matched, nil
	}
	right, err := p.operand()
	if err != nil {
		return false, err
	}
	equal := prusaValuesEqual(left, right)
	if operator.text == "!=" {
		return !equal, nil
	}
	return equal, nil
}

// operand returns the text of a literal or the value of a setting. A setting
// that is absent reads as an empty value, as PrusaSlicer treats it.
func (p *prusaConditionParser) operand() (string, error) {
	token := p.next()
	switch token.kind {
	case tokString, tokNumber:
		return token.text, nil
	case tokSetting:
		raw := p.values[token.text]
		if token.index < 0 {
			return raw, nil
		}
		items, err := splitPrusaValues(raw)
		if err != nil {
			return "", fmt.Errorf("setting %s is malformed", token.text)
		}
		if token.index >= len(items) {
			return "", nil
		}
		return items[token.index], nil
	}
	return "", fmt.Errorf("compatibility condition has a missing operand")
}

// prusaValuesEqual compares numerically when both sides are numbers, so 0.2
// and 0.20 match, and compares text otherwise.
func prusaValuesEqual(a, b string) bool {
	fa, errA := strconv.ParseFloat(a, 64)
	fb, errB := strconv.ParseFloat(b, 64)
	if errA == nil && errB == nil {
		return fa == fb
	}
	return a == b
}
