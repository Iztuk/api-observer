package audit

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type RuleSet struct {
	Rules map[string]*Rule `json:"rules" yaml:"rules"`
}

type Rule struct {
	Enabled     *bool       `json:"enabled" yaml:"enabled"`
	Scope       []RuleScope `json:"scope" yaml:"scope"`
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`

	Selector RuleSelector `json:"selector" yaml:"selector"`
	Match    RuleMatch    `json:"match" yaml:"match"`
	Finding  RuleFinding  `json:"finding" yaml:"finding"`

	Chain       string `json:"chain,omitempty" yaml:"chain,omitempty"` // HostRule name will be the reference to the chained HostRule
	ChainedRule *Rule  `json:"-" yaml:"-"`
}

type RuleScope string

const (
	RuleScopeRequest  RuleScope = "request"
	RuleScopeResponse RuleScope = "response"
)

type RuleSelector struct {
	Hosts []string `json:"hosts,omitempty" yaml:"hosts,omitempty"`
}

type RuleMatch struct {
	Mode       MatchMode        `json:"mode,omitempty" yaml:"mode,omitempty"`
	Conditions []MatchCondition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

type MatchMode string

const (
	MatchModeAll MatchMode = "all"
	MatchModeAny MatchMode = "any"
)

type MatchCondition struct {
	Target RuleTarget `json:"target" yaml:"target"`
	Key    string     `json:"key,omitempty" yaml:"key,omitempty"` // Only applies to key value collections

	Operator MatchOperator `json:"operator" yaml:"operator"`
	Value    string        `json:"value,omitempty" yaml:"value,omitempty"`
	Negated  bool          `json:"negated,omitempty" yaml:"negated,omitempty"`

	Regex *regexp.Regexp `json:"-" yaml:"-"`
}

type RuleTarget string

const (
	RuleTargetPath       RuleTarget = "path"
	RuleTargetQuery      RuleTarget = "query"
	RuleTargetHeader     RuleTarget = "header"
	RuleTargetBody       RuleTarget = "body"
	RuleTargetBodyLength RuleTarget = "body_length"
	RuleTargetMethod     RuleTarget = "method"
	RuleTargetCookie     RuleTarget = "cookie"
	RuleTargetCookieName RuleTarget = "cookie_name"
)

type MatchOperator string

const (
	MatchOperatorRegex       MatchOperator = "regex"
	MatchOperatorStringEqual MatchOperator = "string_equal"
	MatchOperatorLessThan    MatchOperator = "less_than"
	MatchOperatorDetectSQLi  MatchOperator = "detect_sqli"
)

type RuleFinding struct {
	Title    string   `json:"title" yaml:"title"`
	Message  string   `json:"message" yaml:"message"`
	Severity string   `json:"severity,omitempty" yaml:"severity,omitempty"`
	Tags     []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

func ParseRuleSet(content string) (*RuleSet, error) {
	if strings.TrimSpace(content) == "" {
		return NewRuleSet(), nil
	}

	var ruleset RuleSet

	if err := yaml.Unmarshal([]byte(content), &ruleset); err != nil {
		return nil, fmt.Errorf("failed to parse ruleset: %w", err)
	}

	if ruleset.Rules == nil {
		return NewRuleSet(), nil
	}
}

func NewRuleSet() *RuleSet {
	return &RuleSet{
		Rules: make(map[string]*Rule),
	}
}

func (doc *RuleSet) CompilePatterns() error {
	for ruleID, rule := range doc.Rules {
		for i := range rule.Match.Conditions {
			pattern := &rule.Match.Conditions[i]

			if pattern.Operator != MatchOperatorRegex {
				continue
			}

			re, err := regexp.Compile(pattern.Value)
			if err != nil {
				return fmt.Errorf(
					"host rule %q has invalid regex pattern %q: %w",
					ruleID,
					pattern.Value,
					err,
				)
			}

			pattern.Regex = re
		}
	}

	return nil
}

func ParseRuleSet(content string) (*RuleSet, error) {
	if strings.TrimSpace(content) == "" {
		return nil, nil
	}

	var doc HostRulesDoc

	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return nil, fmt.Errorf("failed to parse host rules: %w", err)
	}

	if doc.Rules == nil {
		doc.Rules = make(map[string]*HostRule)
	} else {
		if err := doc.chainRules(); err != nil {
			return nil, err
		}

		if err := doc.validateChains(); err != nil {
			return nil, err
		}

		for _, rule := range doc.Rules {
			if rule.Enabled == nil {
				v := true
				rule.Enabled = &v
			}
		}
	}

	if err := doc.CompilePatterns(); err != nil {
		return nil, err
	}

	return &doc, nil
}

func (doc *HostRulesDoc) chainRules() error {
	for id, rule := range doc.Rules {
		if rule.Chain == "" {
			continue
		}

		chainedRule, ok := doc.Rules[rule.Chain]
		if !ok {
			return fmt.Errorf("failed to chain rule '%s' to rule '%s'", id, rule.Chain)
		}

		rule.ChainedRule = chainedRule

	}

	return nil
}

func (doc *HostRulesDoc) validateChains() error {
	for id, rule := range doc.Rules {
		visited := make(map[*HostRule]bool)

		current := rule

		for current != nil {
			if visited[current] {
				return fmt.Errorf(
					"cycle detected in rule chain starting at %q",
					id,
				)
			}

			visited[current] = true
			current = current.ChainedRule
		}
	}

	return nil
}
