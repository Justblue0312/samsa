package common

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"

	"github.com/gosimple/slug"
)

func GenerateRandomColor() (string, error) {
	const letters = "0123456789ABCDEF"

	b := make([]byte, 7)
	b[0] = '#'

	max := big.NewInt(int64(len(letters)))

	for i := 1; i < 7; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = letters[n.Int64()]
	}

	return string(b), nil
}

func GenerateCode(length int) (string, error) {
	if length <= 0 {
		return "", nil
	}

	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, length)
	max := big.NewInt(int64(len(charset)))

	for i := range length {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = charset[n.Int64()]
	}

	return string(b), nil
}

func GenerateNextVersionName(currentVersions []string) string {
	maxVersion := 0
	for _, v := range currentVersions {
		var num int
		if len(v) > 0 && v[0] == 'v' {
			_, err := fmt.Sscanf(v, "v%d", &num)
			if err == nil && num > maxVersion {
				maxVersion = num
			}
		}
	}
	return fmt.Sprintf("v%d", maxVersion+1)
}

func Slugify(s string) string {
	return slug.Make(s)
}

func TokenURLSafe(n int) (string, error) {
	// n = number of random bytes
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
