package audit

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type HostRulesDoc struct {
	Rules map[string]*HostRule `json:"rules" yaml:"rules"`
}

type HostRule struct {
	Enabled     *bool     `json:"enabled" yaml:"enabled"`
	AppliesTo   []JobType `json:"applies_to" yaml:"applies_to"`
	Type        RuleType  `json:"type" yaml:"type"`
	Description string    `json:"description,omitempty" yaml:"description,omitempty"`

	Match   RuleMatch   `json:"match" yaml:"match"`
	Finding RuleFinding `json:"finding" yaml:"finding"`

	Chain       string    `json:"chain,omitempty" yaml:"chain,omitempty"` // HostRule name will be the reference to the chained HostRule
	ChainedRule *HostRule `json:"-" yaml:"-"`
}

type RuleType string

const (
	RuleTypePath      RuleType = "path"
	RuleTypeQuery     RuleType = "query"
	RuleTypeHeader    RuleType = "header"
	RuleTypeBodyField RuleType = "body_field"
)

type RuleMatch struct {
	Paths       []string            `json:"paths,omitempty" yaml:"paths,omitempty"`
	Methods     []string            `json:"methods,omitempty" yaml:"methods,omitempty"`
	Headers     map[string][]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	QueryParams map[string][]string `json:"query_params,omitempty" yaml:"query_params,omitempty"`
	Fields      []string            `json:"fields,omitempty" yaml:"fields,omitempty"`
	Patterns    []RulePattern       `json:"patterns,omitempty" yaml:"patterns,omitempty"`
}

type MatchOperator string

const (
	MatchOperatorRegex       MatchOperator = "regex"
	MatchOperatorStringEqual MatchOperator = "string_equal"
	MatchOperatorLessThan    MatchOperator = "less_than"
	MatchOperatorDetectSQLi  MatchOperator = "detect_sqli"
)

type RulePattern struct {
	Target TargetType `json:"target" yaml:"target"`
	Name   string     `json:"name,omitempty" yaml:"name,omitempty"`

	Operator MatchOperator `json:"operator" yaml:"operator"`
	Value    string        `json:"value,omitempty" yaml:"value,omitempty"`
	Negated  bool          `json:"negated,omitempty" yaml:"negated,omitempty"`

	Regex *regexp.Regexp `json:"-" yaml:"-"`
}

type TargetType string

const (
	TargetTypeQuery  TargetType = "query"
	TargetTypeHeader TargetType = "header"
	TargetTypePath   TargetType = "path"
	TargetTypeField  TargetType = "field"
)

type RuleFinding struct {
	Title    string   `json:"title" yaml:"title"`
	Message  string   `json:"message" yaml:"message"`
	Severity string   `json:"severity,omitempty" yaml:"severity,omitempty"`
	Tags     []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

func (doc *HostRulesDoc) CompilePatterns() error {
	for ruleID, rule := range doc.Rules {
		for i := range rule.Match.Patterns {
			pattern := &rule.Match.Patterns[i]

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

func ParseHostRules(content string) (*HostRulesDoc, error) {
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
