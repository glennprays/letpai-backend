package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims represents custom JWT claims
type JWTClaims struct {
	UserID         string `json:"user_id"`
	WhatsAppNumber string `json:"whatsapp_number"`
	Role           string `json:"role,omitempty"` // For admin role
	jwt.RegisteredClaims
}

// JWTService handles JWT token generation and validation
type JWTService struct {
	secretKey      []byte
	expiryDuration time.Duration
}

// NewJWTService creates a new JWT service
func NewJWTService(secretKey string, expiryHours int) *JWTService {
	return &JWTService{
		secretKey:      []byte(secretKey),
		expiryDuration: time.Duration(expiryHours) * time.Hour,
	}
}

// GenerateToken generates a JWT token for a user
func (s *JWTService) GenerateToken(userID, whatsappNumber string) (string, error) {
	return s.GenerateTokenWithRole(userID, whatsappNumber, "")
}

// GenerateTokenWithRole generates a JWT token with role
func (s *JWTService) GenerateTokenWithRole(userID, whatsappNumber, role string) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserID:         userID,
		WhatsAppNumber: whatsappNumber,
		Role:           role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expiryDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "letpai",
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns claims
func (s *JWTService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&JWTClaims{},
		func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return s.secretKey, nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
	)

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ExtractUserID extracts user ID from token string
func (s *JWTService) ExtractUserID(tokenString string) (string, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

// ExtractRole extracts role from token string
func (s *JWTService) ExtractRole(tokenString string) (string, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.Role, nil
}

// RefreshToken generates a new token with extended expiry
func (s *JWTService) RefreshToken(tokenString string) (string, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	return s.GenerateTokenWithRole(claims.UserID, claims.WhatsAppNumber, claims.Role)
}
