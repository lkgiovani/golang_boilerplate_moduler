package errs

import "testing"

func TestHTTPStatus_AllCodes(t *testing.T) {
	cases := []struct {
		code Code
		want int
	}{
		{EBADREQUEST, 400},
		{EINVALID, 422},
		{EUNAUTHORIZED, 401},
		{EFORBIDDEN, 403},
		{ENOTFOUND, 404},
		{ECONFLICT, 409},
		{EDUPLICATION, 409},
		{EPRECONDITION, 412},
		{EEXPIRED, 410},
		{ERATELIMIT, 429},
		{ENOTIMPLEMENTED, 501},
		{ETIMEOUT, 504},
		{EUNAVAILABLE, 503},
		{EINTERNAL, 500},
	}

	for _, tc := range cases {
		if got := HTTPStatus(tc.code); got != tc.want {
			t.Errorf("HTTPStatus(%q) = %d, want %d", tc.code, got, tc.want)
		}
	}
}

func TestHTTPStatus_UnknownCode(t *testing.T) {
	if got := HTTPStatus(Code("nonexistent")); got != 500 {
		t.Fatalf("HTTPStatus(unknown) = %d, want 500", got)
	}
}

func TestHTTPStatus_EmptyCode(t *testing.T) {
	if got := HTTPStatus(Code("")); got != 500 {
		t.Fatalf("HTTPStatus(empty) = %d, want 500", got)
	}
}

func TestHTTPStatus_MapHasAllFourteen(t *testing.T) {
	if got := len(codeToHTTP); got != 14 {
		t.Fatalf("codeToHTTP has %d entries, want 14", got)
	}
}
