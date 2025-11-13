package main

import (
	"fmt"
	"strconv"
	"strings"
)

type policyRule struct {
	Effect string // "allow" or "deny"
	Action string // "*" or specific action like "read", "write"
	Expr   expr
}

type expr interface {
	Eval(ctx map[string]string) bool
}

type comparisonExpr struct {
	Identifier string
	Operator   string
	Value      string
}

func (e *comparisonExpr) Eval(ctx map[string]string) bool {
	val, ok := ctx[e.Identifier]
	switch e.Operator {
	case "==":
		return ok && val == e.Value
	case "!=":
		return !ok || val != e.Value
	}
	return false
}

type binaryExpr struct {
	Op    string // "and" or "or"
	Left  expr
	Right expr
}

func (e *binaryExpr) Eval(ctx map[string]string) bool {
	switch e.Op {
	case "and":
		return e.Left.Eval(ctx) && e.Right.Eval(ctx)
	case "or":
		return e.Left.Eval(ctx) || e.Right.Eval(ctx)
	}
	return false
}

type notExpr struct {
	Inner expr
}

func (e *notExpr) Eval(ctx map[string]string) bool {
	return !e.Inner.Eval(ctx)
}

// funcExpr supports function calls like all() and contains()
type funcExpr struct {
	Name string
	Args []string
}

func (e *funcExpr) Eval(ctx map[string]string) bool {
	// only supports contains() and all() for now
	if len(e.Args) < 2 {
		return false
	}
	identifier := e.Args[0]
	value := e.Args[1]
	val, ok := ctx[identifier]
	switch e.Name {
	case "contains":
		// treat context value as comma-separated list or string
		if !ok {
			return false
		}
		// if value is in comma-separated list
		parts := strings.Split(val, ",")
		for _, part := range parts {
			if strings.TrimSpace(part) == value {
				return true
			}
		}
		// fallback: substring match for string
		return strings.Contains(val, value)
	case "all":
		// treat context value as comma-separated list
		if !ok {
			return false
		}
		parts := strings.Split(val, ",")
		for _, part := range parts {
			if strings.TrimSpace(part) != value {
				return false
			}
		}
		return len(parts) > 0
	}
	return false
}

type Policy struct {
	Rules []policyRule
}

type PolicyBuilder struct {
	rules []policyRule
}

func NewPolicyBuilder(input string) (*PolicyBuilder, error) {
	engine, err := ParsePolicies(input)
	if err != nil {
		return nil, err
	}
	return &PolicyBuilder{rules: engine.Rules}, nil
}

func (b *PolicyBuilder) Build() *Policy {
	return &Policy{Rules: b.rules}
}

func NewPolicy(input string) *Policy {
	builder, err := NewPolicyBuilder(input)
	if err != nil {
		panic(err)
	}
	return builder.Build()
}

func ParsePolicies(input string) (*Policy, error) {
	trimmed := strings.TrimSpace(input)
	if strings.HasPrefix(strings.ToLower(trimmed), "allow if ") || strings.HasPrefix(strings.ToLower(trimmed), "deny if ") {
		rule, err := parseRule(trimmed)
		if err != nil {
			return nil, err
		}
		return &Policy{Rules: []policyRule{rule}}, nil
	}
	lines := strings.Split(input, "\n")
	var rules []policyRule
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		rule, err := parseRule(line)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return &Policy{Rules: rules}, nil
}

func parseRule(line string) (policyRule, error) {
	lower := strings.ToLower(line)
	var effect, action string
	if strings.HasPrefix(lower, "allow ") {
		effect = "allow"
		line = line[6:]
	} else if strings.HasPrefix(lower, "deny ") {
		effect = "deny"
		line = line[5:]
	} else {
		return policyRule{}, fmt.Errorf("rule must start with 'allow <action> if' or 'deny <action> if'")
	}
	// expect format: allow <action> if ...
	parts := strings.SplitN(line, "if", 2)
	if len(parts) != 2 {
		return policyRule{}, fmt.Errorf("rule must contain 'if'")
	}
	action = strings.TrimSpace(parts[0])
	if action == "" {
		return policyRule{}, fmt.Errorf("missing action in rule")
	}
	condition := strings.TrimSpace(parts[1])
	tokens := tokenize(condition)
	ast, _, err := parseExpr(tokens, 0)
	if err != nil {
		return policyRule{}, err
	}
	return policyRule{Effect: effect, Action: action, Expr: ast}, nil
}

func tokenize(s string) []string {
	var tokens []string
	i := 0
	for i < len(s) {
		// skip all whitespace (spaces, tabs, newlines, carriage returns)
		for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
			i++
		}
		if i >= len(s) {
			break
		}
		switch {
		case isAlphaNum(s[i]):
			j := i
			for j < len(s) && (isAlphaNum(s[j]) || s[j] == '.' || s[j] == '_') {
				j++
			}
			// check for function call
			if j < len(s) && s[j] == '(' {
				tokens = append(tokens, s[i:j])
				tokens = append(tokens, "(")
				i = j + 1
			} else {
				tokens = append(tokens, s[i:j])
				i = j
			}
		case s[i] == '(' || s[i] == ')':
			tokens = append(tokens, string(s[i]))
			i++
		case s[i] == ',':
			tokens = append(tokens, ",")
			i++
		case s[i] == '=' && i+1 < len(s) && s[i+1] == '=':
			tokens = append(tokens, "==")
			i += 2
		case s[i] == '!' && i+1 < len(s) && s[i+1] == '=':
			tokens = append(tokens, "!=")
			i += 2
		case s[i] == '"':
			j := i + 1
			for j < len(s) && s[j] != '"' {
				j++
			}
			if j < len(s) {
				tokens = append(tokens, s[i:j+1])
				i = j + 1
			} else {
				tokens = append(tokens, s[i:])
				i = len(s)
			}
		default:
			i++
		}
	}
	return tokens
}

func isAlphaNum(b byte) bool {
	return (b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9')
}

func parseExpr(tokens []string, pos int) (expr, int, error) {
	left, pos, err := parseTerm(tokens, pos)
	if err != nil {
		return nil, pos, err
	}
	for pos < len(tokens) {
		if tokens[pos] == "and" || tokens[pos] == "or" {
			op := tokens[pos]
			right, nextPos, err := parseTerm(tokens, pos+1)
			if err != nil {
				return nil, pos, err
			}
			left = &binaryExpr{Op: op, Left: left, Right: right}
			pos = nextPos
		} else {
			break
		}
	}
	return left, pos, nil
}

func parseTerm(tokens []string, pos int) (expr, int, error) {
	if pos < len(tokens) && tokens[pos] == "not" {
		inner, nextPos, err := parseTerm(tokens, pos+1)
		if err != nil {
			return nil, pos, err
		}
		return &notExpr{Inner: inner}, nextPos, nil
	}
	// function call: all(identifier, value), contains(identifier, value)
	if pos+3 < len(tokens) && (tokens[pos] == "all" || tokens[pos] == "contains") && tokens[pos+1] == "(" {
		funcName := tokens[pos]
		args := []string{}
		argPos := pos + 2
		for argPos < len(tokens) && tokens[argPos] != ")" {
			if tokens[argPos] == "," {
				argPos++
				continue
			}
			arg := tokens[argPos]
			if len(arg) >= 2 && arg[0] == '"' && arg[len(arg)-1] == '"' {
				unquoted, err := strconv.Unquote(arg)
				if err == nil {
					arg = unquoted
				}
			}
			args = append(args, arg)
			argPos++
		}
		if argPos >= len(tokens) || tokens[argPos] != ")" {
			return nil, pos, fmt.Errorf("missing closing parenthesis in function call")
		}
		return &funcExpr{Name: funcName, Args: args}, argPos + 1, nil
	}
	if pos < len(tokens) && tokens[pos] == "(" {
		e, nextPos, err := parseExpr(tokens, pos+1)
		if err != nil {
			return nil, pos, err
		}
		if nextPos >= len(tokens) || tokens[nextPos] != ")" {
			return nil, pos, fmt.Errorf("missing closing parenthesis")
		}
		return e, nextPos + 1, nil
	}
	return parseComparison(tokens, pos)
}

func parseComparison(tokens []string, pos int) (expr, int, error) {
	if pos+2 >= len(tokens) {
		return nil, pos, fmt.Errorf("expected comparison at token %d", pos)
	}
	identifier := tokens[pos]
	operator := tokens[pos+1]
	value := tokens[pos+2]
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		unquoted, err := strconv.Unquote(value)
		if err == nil {
			value = unquoted
		}
	}
	return &comparisonExpr{
		Identifier: identifier,
		Operator:   operator,
		Value:      value,
	}, pos + 3, nil
}

func (p *Policy) Evaluate(ctx map[string]string) string {
	action := ctx["action"]
	for _, rule := range p.Rules {
		if (rule.Action == "*" || rule.Action == action) && rule.Expr.Eval(ctx) {
			return rule.Effect
		}
	}
	return "deny"
}
