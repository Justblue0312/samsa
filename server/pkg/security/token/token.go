package token

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"hash/crc32"
	"math/big"
)

const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// crc32ToBase62 converts a CRC32 value to a zero-padded base62 string (length 6).
func crc32ToBase62(number uint32) string {
	base := uint32(len(base62Chars))
	encoded := make([]byte, 0)

	for number > 0 {
		remainder := number % base
		number /= base
		encoded = append([]byte{base62Chars[remainder]}, encoded...)
	}

	// Zero-pad to 6 characters
	for len(encoded) < 6 {
		encoded = append([]byte{'0'}, encoded...)
	}

	return string(encoded)
}

// GetTokenHash returns the HMAC-SHA256 hex digest of the token using the secret.
func GetTokenHash(token string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}

// GenerateToken generates a high-entropy token with an optional prefix.
// Format: <prefix><random 37 chars><crc32 base62 checksum>
func GenerateToken(prefix string) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const tokenLength = 37

	tokenBytes := make([]byte, tokenLength)
	for i := range tokenLength {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		tokenBytes[i] = charset[n.Int64()]
	}

	token := string(tokenBytes)

	// Calculate CRC32 checksum
	checksum := crc32.ChecksumIEEE([]byte(token))
	checksumBase62 := crc32ToBase62(checksum)

	return prefix + token + checksumBase62, nil
}

// GenerateTokenHashPair generates a token and its HMAC-SHA256 hash.
// Only the hash should be stored.
func GenerateTokenHashPair(secret string, prefix string) (token string, hash string, err error) {
	token, err = GenerateToken(prefix)
	if err != nil {
		return "", "", err
	}

	hash = GetTokenHash(token, secret)
	return token, hash, nil
}
