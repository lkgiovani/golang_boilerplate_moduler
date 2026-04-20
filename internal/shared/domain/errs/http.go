package errs

// codeToHTTP is the single source of truth for Code → HTTP status mapping.
// PLAN-02 rewrites error_handler.go and span_helpers.go to call HTTPStatus
// instead of maintaining their own copies.
var codeToHTTP = map[Code]int{
	EBADREQUEST:     400,
	EINVALID:        422,
	EUNAUTHORIZED:   401,
	EFORBIDDEN:      403,
	ENOTFOUND:       404,
	ECONFLICT:       409,
	EDUPLICATION:    409,
	EPRECONDITION:   412,
	EEXPIRED:        410,
	ERATELIMIT:      429,
	ENOTIMPLEMENTED: 501,
	ETIMEOUT:        504,
	EUNAVAILABLE:    503,
	EINTERNAL:       500,
}

// HTTPStatus returns the HTTP status code for a given domain Code.
// Unknown codes fall back to 500 (Internal Server Error).
func HTTPStatus(code Code) int {
	if status, ok := codeToHTTP[code]; ok {
		return status
	}
	return 500
}
