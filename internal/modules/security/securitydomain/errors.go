package securitydomain

import "golang_boilerplate_module/internal/shared/domain/errs"

// NoActiveBlock is returned when a lookup for an active user block finds
// nothing. Maps to ENOTFOUND (404) — callers often treat this as "user
// is not currently blocked" rather than an error.
func NoActiveBlock() *errs.Error {
	return errs.Errorf(errs.ENOTFOUND, "No active block found for user")
}
