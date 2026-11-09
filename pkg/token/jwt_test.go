package token

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestJWTManager_GenerateAndValidate(t *testing.T) {
	secret := "test-secret-key-must-be-long-enough-32-chars"
	accessTTL := 5 * time.Minute
	refreshTTL := 24 * time.Hour

	mgr := NewJWTManager(secret, accessTTL, refreshTTL)
	userID := uuid.New()
	email := "user@example.com"

	// 1. Generate Token Pair
	pair, err := mgr.GenerateTokenPair(userID, email)
	if err != nil {
		t.Fatalf("unexpected error generating token pair: %v", err)
	}

	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("expected non-empty tokens in pair, got %+v", pair)
	}

	// 2. Validate Access Token
	claims, err := mgr.ValidateToken(pair.AccessToken, AccessToken)
	if err != nil {
		t.Fatalf("failed to validate access token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected userID %s, got %s", userID, claims.UserID)
	}
	if claims.Email != email {
		t.Errorf("expected email %s, got %s", email, claims.Email)
	}
	if claims.Type != AccessToken {
		t.Errorf("expected type ACCESS, got %s", claims.Type)
	}

	// 3. Validate Token Type Mismatch
	_, err = mgr.ValidateToken(pair.AccessToken, RefreshToken)
	if err != ErrInvalidType {
		t.Errorf("expected ErrInvalidType, got %v", err)
	}

	// 4. Validate Refresh Token
	refClaims, err := mgr.ValidateToken(pair.RefreshToken, RefreshToken)
	if err != nil {
		t.Fatalf("failed to validate refresh token: %v", err)
	}
	if refClaims.UserID != userID {
		t.Errorf("expected userID %s, got %s", userID, refClaims.UserID)
	}

	// 5. Validate with empty expected type (any type allowed)
	anyClaims, err := mgr.ValidateToken(pair.AccessToken, "")
	if err != nil {
		t.Fatalf("failed to validate without expected type: %v", err)
	}
	if anyClaims.UserID != userID {
		t.Errorf("expected user ID match, got %s", anyClaims.UserID)
	}

	// 6. Validate with completely malformed token string
	_, err = mgr.ValidateToken("invalid.token.string", AccessToken)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken for garbage string, got %v", err)
	}

	// 7. Token signed with wrong algorithm (e.g. None or RSA)
	unsecuredToken := jwt.NewWithClaims(jwt.SigningMethodNone, &Claims{
		UserID: userID,
		Email:  email,
		Type:   AccessToken,
	})
	unsecuredStr, _ := unsecuredToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	_, err = mgr.ValidateToken(unsecuredStr, AccessToken)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken for invalid alg, got %v", err)
	}
}

func TestJWTManager_ExpiredToken(t *testing.T) {
	secret := "test-secret-key-must-be-long-enough-32-chars"
	mgr := NewJWTManager(secret, -1*time.Minute, -1*time.Minute)

	tokenStr, _, err := mgr.GenerateAccessToken(uuid.New(), "expired@example.com")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = mgr.ValidateToken(tokenStr, AccessToken)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken for expired token, got %v", err)
	}
}

func TestJWTManager_WrongSecretAndClaims(t *testing.T) {
	mgr1 := NewJWTManager("secret-key-1-abcdefghijklmnopqrstuvwxyz", 10*time.Minute, 10*time.Minute)
	mgr2 := NewJWTManager("secret-key-2-abcdefghijklmnopqrstuvwxyz", 10*time.Minute, 10*time.Minute)

	tokenStr, _, err := mgr1.GenerateAccessToken(uuid.New(), "test@example.com")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	// Validate with wrong secret manager
	_, err = mgr2.ValidateToken(tokenStr, AccessToken)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken for wrong secret, got %v", err)
	}

	// Token with standard map claims instead of custom Claims
	mapClaimsToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": uuid.New().String(),
		"exp": time.Now().Add(10 * time.Minute).Unix(),
	})
	mapClaimsStr, err := mapClaimsToken.SignedString([]byte("secret-key-1-abcdefghijklmnopqrstuvwxyz"))
	if err != nil {
		t.Fatalf("unexpected sign err: %v", err)
	}
	_, err = mgr1.ValidateToken(mapClaimsStr, AccessToken)
	if err != ErrInvalidType && err != ErrInvalidToken {
		t.Errorf("expected error for map claims, got %v", err)
	}
}
