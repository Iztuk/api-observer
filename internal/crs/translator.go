package crs

import (
	"api-observer/internal/audit"
	"fmt"
)

type TranslationResult struct {
	Rule     audit.Rule
	Warnings []string
}

func TranslateRule(src CRSRule) (TranslationResult, error) {
	var result TranslationResult

	result.Rule = audit.Rule{
		Scope: make(audit.RuleScopes, 0),

		Description: fmt.Sprintf(
			"Translated from CRS rule %s",
			src.Actions.ID,
		),

		Match: audit.RuleMatch{
			Mode:       audit.MatchModeAny,
			Conditions: make(audit.MatchConditions, 0),
		},
	}

	sc, err := translatePhaseToScope(src.Actions.Phase)
	if err != nil {
		return TranslationResult{}, err
	}
	result.Rule.Scope = append(result.Rule.Scope, sc)

	mo, err := translateOperatorToMatchOperator(src.Operator)
	if err != nil {
		return TranslationResult{}, err
	}

	var errors []error
	result.Rule.Match.Conditions, errors = translateTargetMatchConditions(src.Targets, src.Operator, mo)
	if len(errors) != 0 {
		for _, err := range errors {
			result.Warnings = append(result.Warnings, err.Error())
		}
	}

	result.Rule.Finding = translateActionsToFinding(src)

	if src.ChainedRule != nil {
		result.Rule.Chain = fmt.Sprintf("crs-rule-%s", src.ChainedRule.Actions.ID)
	}

	return result, nil
}

// Only returns one RuleScope based on CRS Rule behavior (can only execute one phase per SecRule)
func translatePhaseToScope(phase int) (audit.RuleScope, error) {
	if phase > 0 && phase <= 2 {
		return audit.RuleScopeRequest, nil
	} else if phase > 2 && phase <= 4 {
		return audit.RuleScopeResponse, nil
	} else {
		return "", fmt.Errorf(
			"unsupported CRS phase: %d",
			phase,
		)
	}
}

func translateOperatorToMatchOperator(o Operator) (audit.MatchOperator, error) {
	switch o.Type {
	case OperatorRegex:
		return audit.MatchOperatorRegex, nil

	case OperatorDetectSQLi:
		return audit.MatchOperatorDetectSQLi, nil

	case OperatorStringEq:
		return audit.MatchOperatorStringEqual, nil

	case OperatorEqual:
		return audit.MatchOperatorEqual, nil

	case OperatorLessThan:
		return audit.MatchOperatorLessThan, nil

	case OperatorGreaterThan:
		return audit.MatchOperatorGreaterThan, nil

	case OperatorLessThanOrEqual:
		return audit.MatchOperatorLessThanOrEqual, nil

	case OperatorGreaterThanOrEqual:
		return audit.MatchOperatorGreaterThanOrEqual, nil

	default:
		return "", fmt.Errorf(
			"unsupported CRS operator %q",
			o.Type,
		)
	}
}

func translateTargetMatchConditions(targets []Target, operator Operator, matchOperator audit.MatchOperator) (audit.MatchConditions, []error) {
	var result audit.MatchConditions
	var errors []error

	for _, t := range targets {
		conditions := make([]audit.MatchCondition, 0)

		switch t.Variable {
		case Args, ArgsNames:
			conditions = append(conditions, argsToMatchCondition(t)...)
		case RequestBody:
			conditions = append(conditions, audit.MatchCondition{
				Target: audit.RuleTargetBody,
			})

		case RequestBodyLength:
			conditions = append(conditions, audit.MatchCondition{
				Target: audit.RuleTargetBodyLength,
			})

		case RequestCookies:
			conditions = append(conditions, audit.MatchCondition{
				Target: audit.RuleTargetCookie,
				Key:    t.Selector,
			})

		case RequestCookiesNames:
			conditions = append(conditions, audit.MatchCondition{
				Target: audit.RuleTargetCookieName,
			})

		case RequestHeaders, RequestHeadersNames:
			conditions = append(conditions, audit.MatchCondition{
				Target: audit.RuleTargetHeader,
				Key:    t.Selector,
			})

		case RequestMethod:
			conditions = append(conditions, audit.MatchCondition{
				Target: audit.RuleTargetMethod,
			})

		default:
			errors = append(errors, fmt.Errorf("unsupported variable type: %s", t.Variable))
			continue
		}

		for _, cond := range conditions {
			newCond := audit.MatchCondition{
				Target:   cond.Target,
				Key:      cond.Key,
				Operator: matchOperator,
				Value:    operator.Value,
				Negated:  operator.Negated,
			}

			result = append(result, newCond)
		}
	}

	return result, errors
}

func argsToMatchCondition(t Target) []audit.MatchCondition {
	return []audit.MatchCondition{
		{
			Target: audit.RuleTargetBody,
		},
		{
			Target: audit.RuleTargetQuery,
			Key:    t.Selector,
		},
	}
}

func translateActionsToFinding(source CRSRule) audit.RuleFinding {
	title := source.Actions.Message

	if title == "" {
		title = fmt.Sprintf(
			"CRS Rule %s",
			source.Actions.ID,
		)
	}

	return audit.RuleFinding{
		Title:    title,
		Message:  title,
		Severity: string(source.Actions.Severity),
		Tags: append(
			[]string(nil),
			source.Actions.Tags...,
		),
	}
}
