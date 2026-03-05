package jwt

import (
	"testing"
	"time"

	"github.com/justblue/samsa/config"
	"github.com/stretchr/testify/assert"
)

func setupTestConfig() *config.Config {
	cfg := config.Config{}
	cfg.SecretKey = "test-secret"
	cfg.Jwt.Issuer = "test-issuer"
	cfg.Jwt.AccessTokenTTL = 1 * time.Hour
	cfg.Jwt.RefreshTokenTTL = 24 * time.Hour
	return &cfg
}

func TestJWT(t *testing.T) {
	cfg := setupTestConfig()
	userID := "user-123"
	provider := "local"
	exType := "login"

	infos := JwtInfos{
		UserID:       userID,
		AuthProvider: &provider,
		ExType:       &exType,
	}

	t.Run("Encode and Decode Access Token", func(t *testing.T) {
		token, err := Encode(cfg, AccessToken, infos)
		assert.NoError(t, err)

		decodedInfos, err := Decode(cfg, token, provider, exType)
		assert.NoError(t, err)

		assert.Equal(t, userID, decodedInfos.UserID)
		assert.Equal(t, provider, *decodedInfos.AuthProvider)
		assert.NotNil(t, decodedInfos.ExType)
		assert.Equal(t, exType, *decodedInfos.ExType)
	})

	t.Run("Decode with validation", func(t *testing.T) {
		token, _ := Encode(cfg, AccessToken, infos)

		// Success
		_, err := Decode(cfg, token, provider, exType)
		assert.NoError(t, err)

		// Invalid provider
		_, err = Decode(cfg, token, "other-provider", exType)
		assert.ErrorIs(t, err, ErrInvalidProvider)

		// Invalid type
		_, err = Decode(cfg, token, provider, "other-type")
		assert.ErrorIs(t, err, ErrInvalidType)

	})

	t.Run("Expired Token", func(t *testing.T) {
		shortCfg := setupTestConfig()
		shortCfg.Jwt.AccessTokenTTL = -1 * time.Second // Already expired

		token, _ := Encode(shortCfg, AccessToken, infos)
		_, err := UnsafeDecode(cfg, token)
		assert.ErrorIs(t, err, ErrExpiredToken)
	})

	t.Run("Invalid Token", func(t *testing.T) {
		_, err := UnsafeDecode(cfg, "invalid.token.string")
		assert.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("Tampered Token", func(t *testing.T) {
		token, _ := Encode(cfg, AccessToken, infos)
		tamperedToken := token + "tamper"

		_, err := UnsafeDecode(cfg, tamperedToken)
		assert.ErrorIs(t, err, ErrInvalidToken)
	})
}
