package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/justblue/samsa/config"
)

var (
	ErrInvalidToken    = errors.New("invalid token")
	ErrExpiredToken    = errors.New("token has expired")
	ErrInvalidClaims   = errors.New("invalid token claims")
	ErrInvalidProvider = errors.New("invalid provider")
	ErrInvalidType     = errors.New("invalid type")
)

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

type JwtInfos struct {
	UserID       string          `json:"user_id"`
	AuthProvider *string         `json:"provider"`
	ExType       *string         `json:"extra_type"`
	Metadata     *map[string]any `json:"metadata"`
}

type Claims struct {
	JwtInfos
	jwt.RegisteredClaims
}

func Encode(c *config.Config, tokenType TokenType, infos JwtInfos) (string, error) {
	var (
		now       = time.Now()
		expiresAt time.Time
	)

	switch tokenType {
	case AccessToken:
		expiresAt = now.Add(c.Jwt.AccessTokenTTL)
	case RefreshToken:
		expiresAt = now.Add(c.Jwt.RefreshTokenTTL)
	}

	claims := Claims{
		JwtInfos: infos,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    c.Jwt.Issuer,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(c.SecretKey))
}

func UnsafeDecode(c *config.Config, tokenString string) (*JwtInfos, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(c.SecretKey), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}
	return &claims.JwtInfos, nil
}

func Decode(c *config.Config, tokenString string, authProv string, exType string) (*JwtInfos, error) {
	infos, err := UnsafeDecode(c, tokenString)
	if infos == nil {
		return nil, err
	}
	if infos.AuthProvider != nil && *infos.AuthProvider != authProv {
		return nil, ErrInvalidProvider
	}

	if infos.ExType != nil && *infos.ExType != exType {
		return nil, ErrInvalidType
	}

	return infos, nil
}
