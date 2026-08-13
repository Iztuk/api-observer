package query

import (
	"fmt"
	"strings"
	"time"
)

func evaluateExpression(expr Expression, item LogItem) (bool, error) {
	switch expr := expr.(type) {
	case ComparisonExpression:
		return evaluateComparison(expr, item)

	case AndExpression:
		left, err := evaluateExpression(expr.Left, item)
		if err != nil {
			return false, err
		}

		if !left {
			return false, nil
		}

		return evaluateExpression(expr.Right, item)

	case OrExpression:
		left, err := evaluateExpression(expr.Left, item)
		if err != nil {
			return false, err
		}

		if left {
			return true, nil
		}

		return evaluateExpression(expr.Right, item)

	case NotExpression:
		result, err := evaluateExpression(expr.Expression, item)
		if err != nil {
			return false, err
		}
		return !result, nil
	}

	return false, fmt.Errorf("unknown expression type %T", expr)
}

func evaluateComparison(expr ComparisonExpression, item LogItem) (bool, error) {
	switch expr.Field.Name {
	case "host":
		return evaluateHostField(expr, item)

	case "method":
		return evaluateMethod(expr, item)

	case "path":
		return evaluatePath(expr, item)

	case "status":
		return evaluateStatus(expr, item)

	case "timestamp":
		return evaluateTimestamp(expr, item)

	case "findings":
		return evaluateFindingsCount(expr, item)

	default:
		return false, fmt.Errorf(
			"unsupported field %q",
			expr.Field.Name,
		)
	}
}

func evaluateHostField(expr ComparisonExpression, item LogItem) (bool, error) {
	switch expr.Operator.Type {
	case OperatorTypeEqual:
		return (expr.Value.Value == item.Job.Host), nil

	case OperatorTypeNotEqual:
		return (expr.Value.Value != item.Job.Host), nil

	default:
		return false, fmt.Errorf(
			"operator %q is not supported for field %q; allowed operators are = and !=",
			expr.Operator.Name,
			expr.Field.Name,
		)
	}
}

func evaluateMethod(expr ComparisonExpression, item LogItem) (bool, error) {
	switch expr.Operator.Type {
	case OperatorTypeEqual:
		return (expr.Value.Value == item.Job.Method), nil

	case OperatorTypeNotEqual:
		return (expr.Value.Value != item.Job.Method), nil

	default:
		return false, fmt.Errorf(
			"operator %q is not supported for field %q; allowed operators are = and !=",
			expr.Operator.Name,
			expr.Field.Name,
		)
	}
}

func evaluatePath(
	expr ComparisonExpression, item LogItem) (bool, error) {
	expected, ok := expr.Value.Value.(string)
	if !ok {
		return false, fmt.Errorf(
			"field %q requires a string value",
			expr.Field.Name,
		)
	}

	matches := pathMatches(expected, item.Job.Path)

	switch expr.Operator.Type {
	case OperatorTypeEqual:
		return matches, nil

	case OperatorTypeNotEqual:
		return !matches, nil

	default:
		return false, fmt.Errorf(
			"operator %q is not supported for field %q; allowed operators are '=' and '!='",
			expr.Operator.Name,
			expr.Field.Name,
		)
	}
}

func pathMatches(pattern, actual string) bool {
	patternParts := splitPath(pattern)
	actualParts := splitPath(actual)

	if len(patternParts) != len(actualParts) {
		return false
	}

	for i := range patternParts {
		patternPart := patternParts[i]
		actualPart := actualParts[i]

		if isPathParameter(patternPart) {
			if actualPart == "" {
				return false
			}

			continue
		}

		if patternPart != actualPart {
			return false
		}
	}

	return true
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")

	if path == "" {
		return []string{}
	}

	return strings.Split(path, "/")
}

func isPathParameter(segment string) bool {
	return len(segment) >= 3 &&
		segment[0] == '{' &&
		segment[len(segment)-1] == '}'
}

func evaluateStatus(expr ComparisonExpression, item LogItem) (bool, error) {
	expected, ok := expr.Value.Value.(int)
	if !ok {
		return false, fmt.Errorf(
			"field %q requires a numeric value",
			expr.Field.Name,
		)
	}

	actual := item.Job.Status

	switch expr.Operator.Type {
	case OperatorTypeEqual:
		return actual == expected, nil

	case OperatorTypeNotEqual:
		return actual != expected, nil

	case OperatorTypeGreater:
		return actual > expected, nil

	case OperatorTypeGreaterEqual:
		return actual >= expected, nil

	case OperatorTypeLess:
		return actual < expected, nil

	case OperatorTypeLessEqual:
		return actual <= expected, nil

	default:
		return false, fmt.Errorf(
			"operator %q is not supported for field %q; allowed operators are '=', '!=', '>', '>=', '<', '<='",
			expr.Operator.Name,
			expr.Field.Name,
		)
	}
}

func evaluateTimestamp(expr ComparisonExpression, item LogItem) (bool, error) {
	expectedRaw, ok := expr.Value.Value.(string)
	if !ok {
		return false, fmt.Errorf(
			"field %q requires an RFC3339 timestamp string",
			expr.Field.Name,
		)
	}

	expected, err := time.Parse(
		time.RFC3339Nano,
		expectedRaw,
	)
	if err != nil {
		return false, fmt.Errorf(
			"invalid timestamp %q for field %q: %w",
			expectedRaw,
			expr.Field.Name,
			err,
		)
	}

	actual, err := time.Parse(
		time.RFC3339Nano,
		item.Job.Timestamp,
	)
	if err != nil {
		return false, fmt.Errorf(
			"invalid log timestamp %q: %w",
			item.Job.Timestamp,
			err,
		)
	}

	switch expr.Operator.Type {
	case OperatorTypeEqual:
		return actual.Equal(expected), nil

	case OperatorTypeNotEqual:
		return !actual.Equal(expected), nil

	case OperatorTypeGreater:
		return actual.After(expected), nil

	case OperatorTypeGreaterEqual:
		return actual.After(expected) ||
			actual.Equal(expected), nil

	case OperatorTypeLess:
		return actual.Before(expected), nil

	case OperatorTypeLessEqual:
		return actual.Before(expected) ||
			actual.Equal(expected), nil

	default:
		return false, fmt.Errorf(
			"operator %q is not supported for field %q; allowed operators are '=', '!=', '>', '>=', '<', '<='",
			expr.Operator.Name,
			expr.Field.Name,
		)
	}
}

func evaluateFindingsCount(expr ComparisonExpression, item LogItem) (bool, error) {
	expected, ok := expr.Value.Value.(int)
	if !ok {
		return false, fmt.Errorf(
			"field %q requires a numeric value",
			expr.Field.Name,
		)
	}

	actual := len(item.Findings)

	switch expr.Operator.Type {
	case OperatorTypeEqual:
		return actual == expected, nil

	case OperatorTypeNotEqual:
		return actual != expected, nil

	case OperatorTypeGreater:
		return actual > expected, nil

	case OperatorTypeGreaterEqual:
		return actual >= expected, nil

	case OperatorTypeLess:
		return actual < expected, nil

	case OperatorTypeLessEqual:
		return actual <= expected, nil

	default:
		return false, fmt.Errorf(
			"operator %q is not supported for field %q; allowed operators are '=', '!=', '>', '>=', '<', '<='",
			expr.Operator.Name,
			expr.Field.Name,
		)
	}
}
