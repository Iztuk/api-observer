package crs

import (
	"fmt"
	"strings"
)

func ParseSecRule(rule string) (CRSRule, error) {
	parts := make([]string, 4)
	var sb strings.Builder

	p := 0
	for i := 0; p < 4; i++ {
		if rule[i] == ' ' {
			parts[p] = sb.String()
			p++
			sb.Reset()
			continue
		}

		sb.WriteRune(rune(rule[i]))
	}

	var cr CRSRule
	if !parseDirective(parts[0]) {
		return CRSRule{}, fmt.Errorf("configuration directive '%s' not supported", parts[0])
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

	return cr, nil
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

func parseActions(raw string)
