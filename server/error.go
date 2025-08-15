package server

var (
	REASON_JSON_SYNTAX_ERROR   = Reason("json syntax error in body")
	REASON_INTERNAL_ERROR      = Reason("internal server error")
	REASON_INVALID_CREDENTIALS = Reason("invalid username/password")
	REASON_INVALID_AUTH_HEADER = Reason("invalid Authorization header")
)


// Error reason
type ErrorReason struct {
	Reason string `json:"reason" example:"reason"`
}

func Reason(err string) ErrorReason {
	return ErrorReason{err}
}
