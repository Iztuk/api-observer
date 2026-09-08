package crs

import (
	"fmt"
	"strings"
)

func ParseSecRule(rule string) (CRSRule, error) {
	parts, err := splitSecRule(rule)
	if err != nil {
		return CRSRule{}, err
	}

	for _, part := range parts {
		fmt.Println("\n", part)
	}

	if len(parts) != 4 {
		return CRSRule{}, fmt.Errorf(
			"expected 4 SecRule components, got %d",
			len(parts),
		)
	}

	var cr CRSRule

	if !parseDirective(parts[0]) {
		return CRSRule{}, fmt.Errorf(
			"configuration directive '%s' not supported",
			parts[0],
		)
	}

	t, err := parseTargets(parts[1])
	if err != nil {
		return CRSRule{}, err
	}
	cr.Targets = t

	o, err := parseOperator(parts[2])
	if err != nil {
		return CRSRule{}, err
	}
	cr.Operator = o

	a, err := parseActions(parts[3])
	if err != nil {
		return CRSRule{}, err
	}
	cr.Actions = a

	return cr, nil
}

func splitSecRule(rule string) ([]string, error) {
	parts := make([]string, 0, 4)

	var sb strings.Builder

	inQuotes := false

	for i := 0; i < len(rule); i++ {
		ch := rule[i]

		switch {
		case ch == '\\':
			continue
		case ch == '"':
			inQuotes = !inQuotes
			sb.WriteByte(ch)

		case isWhitespace(ch) && !inQuotes:
			if sb.Len() == 0 {
				// Ignore additional whitespace between tokens.
				continue
			}

			parts = append(parts, sb.String())
			sb.Reset()

		default:
			sb.WriteByte(ch)
		}
	}

	if inQuotes {
		return nil, fmt.Errorf("unterminated quote in SecRule")
	}

	if sb.Len() > 0 {
		parts = append(parts, sb.String())
	}

	return parts, nil
}

func isWhitespace(ch byte) bool {
	switch ch {
	case ' ', '\t', '\n', '\r':
		return true
	default:
		return false
	}
}

func parseDirective(directive string) bool {
	return directive == "SecRule"
}

func parseTargets(raw string) ([]Target, error) {
	var targets []Target

	for target := range strings.SplitSeq(raw, "|") {
		target, excluded := strings.CutPrefix(target, "!")

		variableName, selector, _ := strings.Cut(target, ":")

		variable := Variable(variableName)
		if !variable.IsValid() {
			return nil, fmt.Errorf(
				"variable %q not supported",
				variableName,
			)
		}

		targets = append(targets, Target{
			Variable: variable,
			Selector: selector,
			Exclude:  excluded,
		})
	}

	return targets, nil
}

func parseOperator(raw string) (Operator, error) {
	raw = strings.Trim(raw, `"`)

	parts := strings.SplitN(raw, " ", 2)

	operatorName := parts[0]

	operatorName, negated := strings.CutPrefix(operatorName, "!")
	operatorName, _ = strings.CutPrefix(operatorName, "@")

	operatorType := OperatorType(operatorName)

	if !operatorType.IsValid() {
		return Operator{}, fmt.Errorf(
			"operator %q not supported",
			operatorName,
		)
	}

	var value string
	if len(parts) == 2 {
		value = parts[1]
	}

	return Operator{
		Type:    operatorType,
		Value:   value,
		Negated: negated,
	}, nil
}

func parseActions(raw string) (Actions, error) {
	var actions Actions

	raw = strings.Trim(raw, `"`)

	// parts := strings.Split(raw, ",")

	for part := range strings.SplitSeq(raw, ",") {
		part = strings.TrimSpace(part)

		// Standalone:
		// block
		// capture
		// pass
		if setter, ok := standaloneActions[part]; ok {
			setter(&actions)
			continue
		}

		// Value-based:
		// id:942100
		// phase:2
		// msg:'something'
		name, value, found := strings.Cut(part, ":")
		if !found {
			return Actions{}, fmt.Errorf(
				"unsupported action %q",
				part,
			)
		}

		setter, ok := valueActions[name]
		if !ok {
			return Actions{}, fmt.Errorf(
				"unsupported action %q",
				name,
			)
		}

		if err := setter(&actions, value); err != nil {
			return Actions{}, err
		}
	}

	return actions, nil
}
