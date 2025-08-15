package server

var REASON_JSON_SYNTAX_ERROR = Reason("json syntax error in body")
var REASON_INTERNAL_ERROR = Reason("internal server error")
var REASON_INVALID_CREDENTIALS = Reason("invalid username/password")

// Error reason
type ErrorReason struct {
	Reason string `json:"reason" example:"reason"`
}

func Reason(err string) ErrorReason {
	return ErrorReason{err}
}
