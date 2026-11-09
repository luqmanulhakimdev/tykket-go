package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
	ErrInvalidType  = errors.New("invalid token type")
)

type TokenType string

const (
	AccessToken  TokenType = "ACCESS"
	RefreshToken TokenType = "REFRESH"
)

// Claims represents custom JWT claims.
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Type   TokenType `json:"type"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type TokenManager interface {
	GenerateAccessToken(userID uuid.UUID, email string) (string, time.Time, error)
	GenerateRefreshToken(userID uuid.UUID, email string) (string, time.Time, error)
	GenerateTokenPair(userID uuid.UUID, email string) (*TokenPair, error)
	ValidateToken(tokenString string, expectedType TokenType) (*Claims, error)
}

type jwtManager struct {
	secret        []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewJWTManager(secret string, accessExpiry, refreshExpiry time.Duration) TokenManager {
	return &jwtManager{
		secret:        []byte(secret),
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

func (m *jwtManager) GenerateAccessToken(userID uuid.UUID, email string) (string, time.Time, error) {
	now := time.Now().UTC()
	exp := now.Add(m.accessExpiry)

	claims := &Claims{
		UserID: userID,
		Email:  email,
		Type:   AccessToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign access token: %w", err)
	}

	return tokenStr, exp, nil
}

func (m *jwtManager) GenerateRefreshToken(userID uuid.UUID, email string) (string, time.Time, error) {
	now := time.Now().UTC()
	exp := now.Add(m.refreshExpiry)

	claims := &Claims{
		UserID: userID,
		Email:  email,
		Type:   RefreshToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return tokenStr, exp, nil
}

func (m *jwtManager) GenerateTokenPair(userID uuid.UUID, email string) (*TokenPair, error) {
	accessToken, accessExp, err := m.GenerateAccessToken(userID, email)
	if err != nil {
		return nil, err
	}

	refreshToken, _, err := m.GenerateRefreshToken(userID, email)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExp,
	}, nil
}

func (m *jwtManager) ValidateToken(tokenString string, expectedType TokenType) (*Claims, error) {
	parsedToken, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})

	if err != nil || !parsedToken.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := parsedToken.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}

	if expectedType != "" && claims.Type != expectedType {
		return nil, ErrInvalidType
	}

	return claims, nil
}

