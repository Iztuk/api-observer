package audit

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type HostRulesDoc struct {
	Rules map[string]HostRule `json:"rules" yaml:"rules"`
}

type HostRule struct {
	Enabled     bool      `json:"enabled" yaml:"enabled"`
	AppliesTo   []JobType `json:"applies_to" yaml:"applies_to"`
	Type        RuleType  `json:"type" yaml:"type"`
	Description string    `json:"description,omitempty" yaml:"description,omitempty"`

	Match   RuleMatch   `json:"match" yaml:"match"`
	Finding RuleFinding `json:"finding" yaml:"finding"`
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

type RulePattern struct {
	Target  TargetType     `json:"target" yaml:"target"`                 // query, header, path, field
	Name    string         `json:"name,omitempty" yaml:"name,omitempty"` // param/header/field name
	Pattern string         `json:"pattern" yaml:"pattern"`               // regex
	Regex   *regexp.Regexp `json:"-" yaml:"-"`
}

type TargetType string

const (
	TargetTypeQuery  TargetType = "query"
	TargetTypeHeader TargetType = "header"
	TargetTypePath   TargetType = "path"
	TargetTypeField  TargetType = "field"
)

type RuleFinding struct {
	Title   string `json:"title" yaml:"title"`
	Message string `json:"message" yaml:"message"`
}

func (doc *HostRulesDoc) CompilePatterns() error {
	for ruleID, rule := range doc.Rules {
		for i := range rule.Match.Patterns {
			pattern := &rule.Match.Patterns[i]

			re, err := regexp.Compile(pattern.Pattern)
			if err != nil {
				return fmt.Errorf(
					"host rule %q has invalid regex pattern %q: %w",
					ruleID,
					pattern.Pattern,
					err,
				)
			}

			pattern.Regex = re
		}

		doc.Rules[ruleID] = rule
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
		doc.Rules = make(map[string]HostRule)
	}

	if err := doc.CompilePatterns(); err != nil {
		return nil, err
	}

	return &doc, nil
}
