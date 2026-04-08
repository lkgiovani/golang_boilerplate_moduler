package authprovider

import (
	"golang_boilerplate_module/internal/modules/auth/authdomain"
)

// TokenProvider defines the contract for JWT token generation and validation.
type TokenProvider interface {
	GenerateTokenPair(claims authdomain.TokenClaims) (*authdomain.TokenPair, error)
	ValidateAccessToken(tokenString string) (*authdomain.TokenClaims, error)
	ValidateRefreshToken(tokenString string) (int64, string, error) // userID, tokenID, error
}
