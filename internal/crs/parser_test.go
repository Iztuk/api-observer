package crs

import (
	"reflect"
	"testing"
)

func TestParseSecRule(t *testing.T) {
	raw := `SecRule REQUEST_COOKIES|REQUEST_COOKIES_NAMES|ARGS_NAMES|ARGS|XML:/*|XML://@* "@rx (?i)union.*?select.*?from" \
    "id:942270,\
    phase:2,\
    block,\
    capture,\
    t:none,t:urlDecodeUni,\
    msg:'Looking for basic sql injection',\
    logdata:'Matched Data: %{TX.0} found within %{MATCHED_VAR_NAME}: %{MATCHED_VAR}',\
    tag:'application-multi',\
    tag:'attack-sqli',\
    tag:'OWASP_CRS',\
    ver:'OWASP_CRS/4.30.0-dev',\
    severity:'CRITICAL',\
    setvar:'tx.sql_injection_score=+%{tx.critical_anomaly_score}'"`

	got, err := ParseSecRule(raw)
	if err != nil {
		t.Fatalf("ParseSecRule() error = %v", err)
	}

	want := CRSRule{
		Targets: []Target{
			{
				Variable: RequestCookies,
			},
			{
				Variable: RequestCookiesNames,
			},
			{
				Variable: ArgsNames,
			},
			{
				Variable: Args,
			},
			{
				Variable: XML,
				Selector: "/*",
			},
			{
				Variable: XML,
				Selector: "//@*",
			},
		},

		Operator: Operator{
			Type:  OperatorRegex,
			Value: `(?i)union.*?select.*?from`,
		},

		Actions: Actions{
			ID:    "942270",
			Phase: 2,

			Transforms: []Transformation{
				TransformNone,
				TransformURLDecodeUni,
			},

			Message: "Looking for basic sql injection",
			LogData: "Matched Data: %{TX.0} found within " +
				"%{MATCHED_VAR_NAME}: %{MATCHED_VAR}",

			Severity: SeverityCritical,

			Tags: []string{
				"application-multi",
				"attack-sqli",
				"OWASP_CRS",
			},

			Version: "OWASP_CRS/4.30.0-dev",

			Block:   true,
			Capture: true,

			SetVars: []SetVar{
				{
					Collection: "tx",
					Name:       "sql_injection_score",
					Operation:  SetVarIncrement,
					Value:      "%{tx.critical_anomaly_score}",
				},
			},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf(
			"ParseSecRule() mismatch\n\ngot:\n%+v\n\nwant:\n%+v",
			got,
			want,
		)
	}
}
