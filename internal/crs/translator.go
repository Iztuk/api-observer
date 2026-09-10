package crs

import "observer/internal/audit"

type TranslationResult struct {
	Rules    []audit.HostRule
	Warnings []string
}

func TranslateRule(rule CRSRule) (TranslationResult, error) {
	var result TranslationResult

	return result, nil
}

func translateTarget(t Target)
