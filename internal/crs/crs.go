// Package crs supports the OWASP Core Rule Set (CRS).
package crs

type CRSRule struct {
	Targets  []Target
	Operator Operator
	Actions  Actions
}

type Variable string

const (
	Args      Variable = "ARGS"
	ArgsNames Variable = "ARGS_NAMES"

	RequestBody       Variable = "REQUEST_BODY"
	RequestBodyLength Variable = "REQUEST_BODY_LENGTH"

	RequestCookies      Variable = "REQUEST_COOKIES"
	RequestCookiesNames Variable = "REQUEST_COOKIES_NAMES"

	RequestHeaders      Variable = "REQUEST_HEADERS"
	RequestHeadersNames Variable = "REQUEST_HEADERS_NAMES"

	RequestMethod   Variable = "REQUEST_METHOD"
	RequestFilename Variable = "REQUEST_FILENAME"
	RequestBasename Variable = "REQUEST_BASENAME"

	XML Variable = "XML"

	TX               Variable = "TX"
	MatchedVar       Variable = "MATCHED_VAR"
	MatchedVarName   Variable = "MATCHED_VAR_NAME"
	MatchedVars      Variable = "MATCHED_VARS"
	MatchedVarsNames Variable = "MATCHED_VARS_NAMES"
)

func (v Variable) IsValid() bool {
	switch v {
	case Args, ArgsNames, RequestBody, RequestBodyLength,
		RequestCookies, RequestCookiesNames, RequestHeaders,
		RequestHeadersNames, RequestMethod, RequestFilename,
		RequestBasename, XML, TX, MatchedVar, MatchedVarName,
		MatchedVars, MatchedVarsNames:
		return true
	default:
		return false
	}
}

type Target struct {
	Variable Variable
	Selector string // Field to check
	Exclude  bool
}

type OperatorType string

const (
	OperatorRegex      OperatorType = "rx"
	OperatorDetectSQLi OperatorType = "detectSQLi"
	OperatorStringEq   OperatorType = "streq"
	OperatorLessThan   OperatorType = "lt"
)

func (ot OperatorType) IsValid() bool {
	switch ot {
	case OperatorRegex, OperatorDetectSQLi, OperatorStringEq, OperatorLessThan:
		return true
	default:
		return false
	}
}

type Operator struct {
	Type    OperatorType
	Value   string
	Negated bool
}

type Actions struct {
	ID         string
	Phase      int
	Transforms []Transformation

	Message  string
	LogData  string
	Severity Severity
	Tags     []string
	Version  string

	// Standalone actions
	Block      bool
	Pass       bool
	NoLog      bool
	Capture    bool
	MultiMatch bool

	SkipAfter string
	SetVars   []SetVar

	ChainedRule *CRSRule
}

type Transformation string

const (
	TransformNone               Transformation = "none"
	TransformUTF8ToUnicode      Transformation = "utf8toUnicode"
	TransformURLDecodeUni       Transformation = "urlDecodeUni"
	TransformRemoveNulls        Transformation = "removeNulls"
	TransformReplaceNulls       Transformation = "replaceNulls"
	TransformReplaceComments    Transformation = "replaceComments"
	TransformRemoveComments     Transformation = "removeComments"
	TransformRemoveCommentsChar Transformation = "removeCommentsChar"
	TransformRemoveWhitespace   Transformation = "removeWhitespace"
)

type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityError    Severity = "ERROR"
	SeverityWarning  Severity = "WARNING"
	SeverityNotice   Severity = "NOTICE"
)

type SetVarOperation string

const (
	SetVarAssign    SetVarOperation = "="
	SetVarIncrement SetVarOperation = "+="
	SetVarDecrement SetVarOperation = "-="
)

type SetVar struct {
	Collection string
	Name       string
	Operation  SetVarOperation
	Value      string
}
