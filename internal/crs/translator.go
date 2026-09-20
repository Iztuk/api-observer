package crs

import (
	"api-observer/internal/audit"
	"fmt"
)

type TranslationResults struct {
	Rules map[string]TranslationResult `json:"rules" yaml:"rules"`
}

type TranslationResult struct {
	Rule     *audit.Rule `json:"rule,omitempty" yaml:"rule,omitempty"`
	Warnings []string    `json:"warnings,omitempty" yaml:"warnings,omitempty"`
	Error    string      `json:"error,omitempty" yaml:"error,omitempty"`
}

// TranslateRules translates an entire collection of CRS rules.
// Individual translation failures are recorded in the results
// instead of stopping the entire import.
func TranslateRules(source []CRSRule) (TranslationResults, error) {
	results := TranslationResults{
		Rules: make(map[string]TranslationResult),
	}

	for i, src := range source {
		id := fmt.Sprintf("crs-rule-%s", src.Actions.ID)

		if src.Actions.ID == "" {
			id = fmt.Sprintf("crs-rule-import-%d", i+1)
		}

		if err := translateRuleChain(
			src,
			id,
			src.Actions.Phase,
			&results,
		); err != nil {
			return results, err
		}
	}

	return results, nil
}

// translateRuleChain translates a rule and its chained children.
// A failed child invalidates its parent, but unrelated rules
// can still be translated.
func translateRuleChain(
	src CRSRule,
	id string,
	inheritedPhase int,
	results *TranslationResults,
) error {
	if _, exists := results.Rules[id]; exists {
		return fmt.Errorf("duplicate translated rule ID %q", id)
	}

	// Reserve the ID before processing the rule.
	results.Rules[id] = TranslationResult{}

	// CRS chain children may inherit their parent's phase.
	if src.Actions.Phase == 0 {
		src.Actions.Phase = inheritedPhase
	}

	// Give unnamed rules an identifier for their descriptions.
	if src.Actions.ID == "" {
		src.Actions.ID = id
	}

	translated, translateErr := TranslateRule(src)

	// Process the child even if the parent could not be
	// translated, so its diagnostics are still available.
	var childID string

	if src.ChainedRule != nil {
		child := *src.ChainedRule
		childID = id + "-chain"

		if child.Actions.ID != "" {
			childID = fmt.Sprintf(
				"crs-rule-%s",
				child.Actions.ID,
			)
		}

		if err := translateRuleChain(
			child,
			childID,
			src.Actions.Phase,
			results,
		); err != nil {
			return err
		}
	}

	// Record individual translation errors.
	if translateErr != nil {
		results.Rules[id] = TranslationResult{
			Error: translateErr.Error(),
		}
		return nil
	}

	// A rule must have at least one usable condition.
	if translated.Rule == nil ||
		len(translated.Rule.Match.Conditions) == 0 {

		results.Rules[id] = TranslationResult{
			Warnings: translated.Warnings,
			Error:    "no supported match conditions",
		}

		return nil
	}

	// A parent cannot safely run if its chained child failed.
	if childID != "" {
		childResult := results.Rules[childID]

		if childResult.Rule == nil {
			results.Rules[id] = TranslationResult{
				Warnings: translated.Warnings,
				Error: fmt.Sprintf(
					"chained rule %s could not be translated: %s",
					childID,
					childResult.Error,
				),
			}

			return nil
		}

		// Only assign a chain reference after confirming
		// that the child translated successfully.
		translated.Rule.Chain = childID
	}

	results.Rules[id] = translated

	return nil
}

// TranslateRule translates a single CRS rule into an
// API Observer rule and collects unsupported-target warnings.
func TranslateRule(src CRSRule) (TranslationResult, error) {
	result := TranslationResult{
		Rule: &audit.Rule{
			Scope: make(audit.RuleScopes, 0),

			Description: fmt.Sprintf(
				"Translated from CRS rule %s",
				src.Actions.ID,
			),

			Match: audit.RuleMatch{
				Mode:       audit.MatchModeAny,
				Conditions: make(audit.MatchConditions, 0),
			},
		},
	}

	scope, err := translatePhaseToScope(src.Actions.Phase)
	if err != nil {
		return TranslationResult{}, err
	}

	result.Rule.Scope = append(
		result.Rule.Scope,
		scope,
	)

	matchOperator, err := translateOperatorToMatchOperator(
		src.Operator,
	)
	if err != nil {
		return TranslationResult{}, err
	}

	conditions, errors := translateTargetMatchConditions(
		src.Targets,
		src.Operator,
		matchOperator,
	)

	result.Rule.Match.Conditions = conditions

	for _, err := range errors {
		result.Warnings = append(
			result.Warnings,
			err.Error(),
		)
	}

	result.Rule.Finding = translateActionsToFinding(src)

	// Chain references are assigned by translateRuleChain(),
	// which knows whether the child translated successfully.

	return result, nil
}

// Only returns one RuleScope based on CRS Rule behavior.
// A SecRule executes during a single phase.
func translatePhaseToScope(
	phase int,
) (audit.RuleScope, error) {
	switch {
	case phase >= 1 && phase <= 2:
		return audit.RuleScopeRequest, nil

	case phase >= 3 && phase <= 4:
		return audit.RuleScopeResponse, nil

	default:
		return "", fmt.Errorf(
			"unsupported CRS phase: %d",
			phase,
		)
	}
}

func translateOperatorToMatchOperator(
	o Operator,
) (audit.MatchOperator, error) {
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

func translateTargetMatchConditions(
	targets []Target,
	operator Operator,
	matchOperator audit.MatchOperator,
) (audit.MatchConditions, []error) {
	var result audit.MatchConditions
	var errors []error

	for _, target := range targets {
		conditions := make([]audit.MatchCondition, 0)

		switch target.Variable {
		case Args, ArgsNames:
			conditions = append(
				conditions,
				argsToMatchCondition(target)...,
			)

		case RequestBody:
			conditions = append(
				conditions,
				audit.MatchCondition{
					Target: audit.RuleTargetBody,
				},
			)

		case RequestBodyLength:
			conditions = append(
				conditions,
				audit.MatchCondition{
					Target: audit.RuleTargetBodyLength,
				},
			)

		case RequestCookies:
			conditions = append(
				conditions,
				audit.MatchCondition{
					Target: audit.RuleTargetCookie,
					Key:    target.Selector,
				},
			)

		case RequestCookiesNames:
			conditions = append(
				conditions,
				audit.MatchCondition{
					Target: audit.RuleTargetCookieName,
				},
			)

		case RequestHeaders, RequestHeadersNames:
			conditions = append(
				conditions,
				audit.MatchCondition{
					Target: audit.RuleTargetHeader,
					Key:    target.Selector,
				},
			)

		case RequestMethod:
			conditions = append(
				conditions,
				audit.MatchCondition{
					Target: audit.RuleTargetMethod,
				},
			)

		default:
			errors = append(
				errors,
				fmt.Errorf(
					"unsupported variable type: %s",
					target.Variable,
				),
			)

			continue
		}

		for _, condition := range conditions {
			result = append(
				result,
				audit.MatchCondition{
					Target:   condition.Target,
					Key:      condition.Key,
					Operator: matchOperator,
					Value:    operator.Value,
					Negated:  operator.Negated,
				},
			)
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

func translateActionsToFinding(
	source CRSRule,
) audit.RuleFinding {
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
