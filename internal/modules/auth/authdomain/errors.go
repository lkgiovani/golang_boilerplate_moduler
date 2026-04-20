package authdomain

import "golang_boilerplate_module/internal/shared/domain/errs"

// Validation / input errors (EBADREQUEST = 400)

func MissingCredentials() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "Email and password are required")
}

func MissingRegistrationFields() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "Name, email and password are required")
}

func PasswordTooShort() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "Password must be at least 8 characters")
}

func MissingRefreshToken() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "Refresh token is required")
}

func MissingLogoutFields() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "User ID and token ID are required")
}

func MissingPasswordFields() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "Current and new password are required")
}

func NewPasswordTooShort() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "New password must be at least 8 characters")
}

func MissingResetFields() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "Token and new password are required")
}

func MissingToken() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "Token is required")
}

func MissingForgotEmail() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "Email is required")
}

func InvalidRequestBody() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "Invalid request body")
}

// Authentication errors (EUNAUTHORIZED = 401)

func InvalidCredentials() *errs.Error {
	return errs.Errorf(errs.EUNAUTHORIZED, "invalid credentials")
}

func AccountDisabled() *errs.Error {
	return errs.Errorf(errs.EUNAUTHORIZED, "account disabled")
}

func CurrentPasswordIncorrect() *errs.Error {
	return errs.Errorf(errs.EUNAUTHORIZED, "Current password is incorrect")
}

func InvalidRefreshToken() *errs.Error {
	return errs.Errorf(errs.EUNAUTHORIZED, "invalid refresh token")
}

func RefreshTokenNotFound() *errs.Error {
	return errs.Errorf(errs.EUNAUTHORIZED, "refresh token not found")
}

func RefreshTokenRevoked() *errs.Error {
	return errs.Errorf(errs.EUNAUTHORIZED, "refresh token has been revoked")
}

func TokenReuseDetected() *errs.Error {
	return errs.Errorf(errs.EUNAUTHORIZED, "token reuse detected")
}

func UserNotFoundForAuth() *errs.Error {
	return errs.Errorf(errs.EUNAUTHORIZED, "user not found")
}

func MissingAuthToken() *errs.Error {
	return errs.Errorf(errs.EUNAUTHORIZED, "missing authentication token")
}

func InvalidOrExpiredAuthToken() *errs.Error {
	return errs.Errorf(errs.EUNAUTHORIZED, "invalid or expired token")
}

func TokenRevoked() *errs.Error {
	return errs.Errorf(errs.EUNAUTHORIZED, "token has been revoked")
}

func MissingUserIdentity() *errs.Error {
	return errs.Errorf(errs.EUNAUTHORIZED, "missing user identity")
}

func MissingTokenIdentity() *errs.Error {
	return errs.Errorf(errs.EUNAUTHORIZED, "missing token identity")
}

// Not-found (ENOTFOUND = 404)

func UserNotFound() *errs.Error {
	return errs.Errorf(errs.ENOTFOUND, "User not found")
}

func InvalidOrExpiredVerificationToken() *errs.Error {
	return errs.Errorf(errs.ENOTFOUND, "Invalid or expired token")
}

func RefreshTokenStorageNotFound() *errs.Error {
	return errs.Errorf(errs.ENOTFOUND, "Refresh token not found")
}

func EmailVerificationTokenNotFound() *errs.Error {
	return errs.Errorf(errs.ENOTFOUND, "Email verification token not found")
}

func PasswordResetTokenNotFound() *errs.Error {
	return errs.Errorf(errs.ENOTFOUND, "Password reset token not found")
}

// Token lifecycle conflicts (per RESEARCH F-05 semantic mapping)

// TokenAlreadyUsed maps to ECONFLICT (409).
func TokenAlreadyUsed() *errs.Error {
	return errs.Errorf(errs.ECONFLICT, "Token already used")
}

// TokenExpired maps to EEXPIRED (410).
func TokenExpired() *errs.Error {
	return errs.Errorf(errs.EEXPIRED, "Token has expired")
}

// NoLocalPassword maps to EPRECONDITION (412) — the user exists but their
// account shape makes the requested operation impossible.
func NoLocalPassword() *errs.Error {
	return errs.Errorf(errs.EPRECONDITION, "User has no local password")
}

// Internal errors (EINTERNAL = 500) — all Reportable=true per
// RESEARCH Open Question 3.

func reportable(e *errs.Error) *errs.Error {
	e.Reportable = true
	return e
}

func FailedToHashPassword() *errs.Error {
	return reportable(errs.Errorf(errs.EINTERNAL, "failed to hash password"))
}

func FailedToGenerateTokens() *errs.Error {
	return reportable(errs.Errorf(errs.EINTERNAL, "failed to generate tokens"))
}

func FailedToExtractTokenClaims() *errs.Error {
	return reportable(errs.Errorf(errs.EINTERNAL, "failed to extract token claims"))
}

func FailedToRotateToken() *errs.Error {
	return reportable(errs.Errorf(errs.EINTERNAL, "failed to rotate token"))
}

func FailedToBlacklistToken() *errs.Error {
	return reportable(errs.Errorf(errs.EINTERNAL, "failed to blacklist access token"))
}

func FailedToCheckBlacklist() *errs.Error {
	return reportable(errs.Errorf(errs.EINTERNAL, "failed to check token blacklist"))
}

func FailedToGenerateVerificationToken() *errs.Error {
	return reportable(errs.Errorf(errs.EINTERNAL, "failed to generate token"))
}

func FailedToSendVerificationEmail() *errs.Error {
	return reportable(errs.Errorf(errs.EINTERNAL, "failed to send confirmation email"))
}

func FailedToGenerateResetToken() *errs.Error {
	return reportable(errs.Errorf(errs.EINTERNAL, "failed to generate token"))
}

func FailedToSendResetEmail() *errs.Error {
	return reportable(errs.Errorf(errs.EINTERNAL, "failed to send reset email"))
}
