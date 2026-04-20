package usersdomain

import "golang_boilerplate_module/internal/shared/domain/errs"

// Validation errors

func MissingNameOrEmail() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "Name and email are required")
}

func NameCannotBeEmpty() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "Name cannot be empty")
}

func NoFieldsToUpdate() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "No fields to update")
}

func InvalidRequestBody() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "Invalid request body")
}

func InvalidUserID() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "Invalid user ID")
}

// Auth-adjacent errors (used by users controllers)

func MissingUserIdentity() *errs.Error {
	return errs.Errorf(errs.EUNAUTHORIZED, "missing user identity")
}

// Duplication (per RESEARCH F-05 — "email already exists" → EDUPLICATION)

func EmailTaken(email string) *errs.Error {
	return errs.Errorf(errs.EDUPLICATION, "email %s already in use", email)
}

// Not found

func UserNotFound() *errs.Error {
	return errs.Errorf(errs.ENOTFOUND, "User not found")
}
