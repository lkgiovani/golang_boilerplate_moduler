package errs

// Code is a machine-readable identifier for a domain error category.
// Using a typed string instead of raw strings gives us compile-time
// protection against passing arbitrary strings where a Code is expected.
type Code string

const (
	EBADREQUEST     Code = "bad_request"
	ECONFLICT       Code = "conflict"
	EDUPLICATION    Code = "duplication"
	EEXPIRED        Code = "expired"
	EFORBIDDEN      Code = "forbidden"
	EINTERNAL       Code = "internal"
	EINVALID        Code = "invalid"
	ENOTFOUND       Code = "not_found"
	ENOTIMPLEMENTED Code = "not_implemented"
	EPRECONDITION   Code = "precondition_failed"
	ERATELIMIT      Code = "rate_limit"
	ETIMEOUT        Code = "timeout"
	EUNAUTHORIZED   Code = "unauthorized"
	EUNAVAILABLE    Code = "service_unavailable"
)
