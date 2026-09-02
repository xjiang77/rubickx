package interpreter

import "strconv"
import "strings"

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type Expression interface {
	Interpret(map[string]int, *int) (bool, error)
}
type Comparison struct{ left, operator, right string }

func resolve(token string, context map[string]int) (int, error) {
	if value, err := strconv.Atoi(token); err == nil {
		return value, nil
	}
	value, ok := context[token]
	if !ok {
		return 0, &PatternError{code: "unknown_identifier"}
	}
	return value, nil
}
func (c Comparison) Interpret(context map[string]int, counter *int) (bool, error) {
	*counter++
	left, err := resolve(c.left, context)
	if err != nil {
		return false, err
	}
	right, err := resolve(c.right, context)
	if err != nil {
		return false, err
	}
	if c.operator == ">=" {
		return left >= right, nil
	}
	return left == right, nil
}

type And struct{ children []Expression }

func (a And) Interpret(context map[string]int, counter *int) (bool, error) {
	for _, child := range a.children {
		ok, err := child.Interpret(context, counter)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}
func parseExpression(text string) (Expression, error) {
	tokens := strings.Fields(text)
	children := []Expression{}
	for index := 0; index < len(tokens); {
		if index+2 >= len(tokens) {
			return nil, &PatternError{code: "invalid_expression"}
		}
		operator := tokens[index+1]
		if operator != ">=" && operator != "==" {
			return nil, &PatternError{code: "unsupported_operator"}
		}
		children = append(children, Comparison{tokens[index], operator, tokens[index+2]})
		index += 3
		if index < len(tokens) {
			if tokens[index] != "and" {
				return nil, &PatternError{code: "invalid_expression"}
			}
			index++
		}
	}
	if len(children) == 0 {
		return nil, &PatternError{code: "invalid_expression"}
	}
	return And{children}, nil
}
func Evaluate(input map[string]any) (any, error) {
	expression, err := parseExpression(input["expression"].(string))
	if err != nil {
		return nil, err
	}
	context := map[string]int{}
	for key, value := range input["context"].(map[string]any) {
		context[key] = int(value.(float64))
	}
	count := 0
	allowed, err := expression.Interpret(context, &count)
	if err != nil {
		return nil, err
	}
	return map[string]any{"allowed": allowed, "comparisons": count}, nil
}
