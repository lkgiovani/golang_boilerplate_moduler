package authproviders

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"golang_boilerplate_module/internal/config"
	"golang_boilerplate_module/internal/modules/auth/authdomain"
	"golang_boilerplate_module/internal/modules/auth/authdomain/authprovider"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// accessClaims represents the JWT claims for an access token.
type accessClaims struct {
	UserID        int64  `json:"user_id"`
	Email         string `json:"email"`
	Admin         bool   `json:"admin"`
	EmailVerified bool   `json:"email_verified"`
	jwt.RegisteredClaims
}

// refreshClaims represents the JWT claims for a refresh token.
type refreshClaims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

// JWTTokenProvider implements authprovider.TokenProvider using golang-jwt/v5.
type JWTTokenProvider struct {
	cfg config.JWTConfig
}

// NewJWTTokenProvider creates a new JWT-based token provider.
func NewJWTTokenProvider(cfg *config.Config) authprovider.TokenProvider {
	return &JWTTokenProvider{cfg: cfg.JWT}
}

// GenerateTokenPair creates an access token (short-lived) and a refresh token
// (long-lived). The access token contains full user claims; the refresh token
// contains only user_id and jti for rotation.
func (p *JWTTokenProvider) GenerateTokenPair(claims authdomain.TokenClaims) (*authdomain.TokenPair, error) {
	now := time.Now()
	accessTokenID := uuid.New().String()
	refreshTokenID := uuid.New().String()

	// Build access token
	accessExpiry := now.Add(time.Duration(p.cfg.AccessExpiry) * time.Minute)
	ac := accessClaims{
		UserID:        claims.UserID,
		Email:         claims.Email,
		Admin:         claims.Admin,
		EmailVerified: claims.EmailVerified,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    p.cfg.Issuer,
			ID:        accessTokenID,
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, ac)
	accessTokenStr, err := accessToken.SignedString([]byte(p.cfg.AccessSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Build refresh token
	refreshExpiry := now.Add(time.Duration(p.cfg.RefreshExpiry) * 24 * time.Hour)
	rc := refreshClaims{
		UserID: claims.UserID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    p.cfg.Issuer,
			ID:        refreshTokenID,
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, rc)
	refreshTokenStr, err := refreshToken.SignedString([]byte(p.cfg.RefreshSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &authdomain.TokenPair{
		AccessToken:  accessTokenStr,
		RefreshToken: refreshTokenStr,
		ExpiresIn:    int64(p.cfg.AccessExpiry * 60), // seconds
	}, nil
}

// ValidateAccessToken parses and validates an access token, returning the
// embedded claims. It enforces HS256 signing method, expiry, and issuer.
func (p *JWTTokenProvider) ValidateAccessToken(tokenString string) (*authdomain.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &accessClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(p.cfg.AccessSecret), nil
	},
		jwt.WithIssuer(p.cfg.Issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid access token: %w", err)
	}

	claims, ok := token.Claims.(*accessClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid access token claims")
	}

	return &authdomain.TokenClaims{
		UserID:        claims.UserID,
		Email:         claims.Email,
		Admin:         claims.Admin,
		TokenID:       claims.ID,
		EmailVerified: claims.EmailVerified,
	}, nil
}

// ValidateRefreshToken parses and validates a refresh token, returning the
// user ID and token ID (jti). It enforces HS256 signing, expiry, and issuer.
func (p *JWTTokenProvider) ValidateRefreshToken(tokenString string) (int64, string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &refreshClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(p.cfg.RefreshSecret), nil
	},
		jwt.WithIssuer(p.cfg.Issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return 0, "", fmt.Errorf("invalid refresh token: %w", err)
	}

	claims, ok := token.Claims.(*refreshClaims)
	if !ok || !token.Valid {
		return 0, "", fmt.Errorf("invalid refresh token claims")
	}

	return claims.UserID, claims.ID, nil
}

// HashToken produces a SHA-256 hex digest of a raw token string.
// Used by the refresh-token rotation flow to store token_hash in PostgreSQL
// instead of persisting the raw JWT.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
