package views

import (
	"encoding/json"
	"fmt"
	"net/url"
	"observer/internal/audit"
)

func parseQueryParams(rawQuery string) url.Values {
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return url.Values{}
	}

	return values
}

func formatOpenAPI(contract audit.OpenAPIDoc) string {
	data, err := json.MarshalIndent(contract, "", "  ")
	if err != nil {
		return fmt.Sprintf(
			"unable to display OpenAPI contract: %v",
			err,
		)
	}

	return string(data)
}

func findingRuleHref(
	host string,
	ruleID string,
) string {
	if isOpenAPIContractRule(audit.RuleID(ruleID)) {
		return fmt.Sprintf(
			"/rules#%s-openapi",
			host,
		)
	}

	return fmt.Sprintf(
		"/rules#%s-%s",
		host,
		ruleID,
	)
}

func isOpenAPIContractRule(ruleID audit.RuleID) bool {
	switch ruleID {
	case
		audit.RuleRequestPathDoesNotExist,
		audit.RuleRequestMethodNotAllowed,
		audit.RuleRequestContentTypeNotAllowed,
		audit.RuleRequestBodyMissing,
		audit.RuleRequestBodyNotAllowed,
		audit.RuleRequestInvalidBodyFormat,
		audit.RuleRequestBodySchemaInvalid,
		audit.RuleResponseStatusCodeNotDefined,
		audit.RuleResponseContentTypeNotAllowed,
		audit.RuleResponseBodyMissing,
		audit.RuleResponseBodyNotAllowed,
		audit.RuleResponseInvalidBodyFormat,
		audit.RuleResponseBodySchemaInvalid:
		return true

	default:
		return false
	}
}
