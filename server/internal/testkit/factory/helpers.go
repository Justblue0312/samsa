package factory

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// randID returns a short random hex string suitable for unique test identifiers.
func randID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// randEmail returns a unique random email for test data.
func randEmail() string {
	return fmt.Sprintf("test-%s@example.com", randID())
}

// now returns the current time truncated to seconds in UTC.
func now() *time.Time {
	t := time.Now().Truncate(time.Second).UTC()
	return &t
}

// Random generators for test data

// RandomEmail generates a random email address with optional prefix.
func RandomEmail(prefix ...string) string {
	p := ""
	if len(prefix) > 0 {
		p = prefix[0] + "-"
	}
	return fmt.Sprintf("%stest-%s@example.com", p, randID())
}

// RandomString generates a random alphanumeric string of given length.
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// RandomName generates a random name like "Test User XXXX".
func RandomName(prefix ...string) string {
	p := "Test"
	if len(prefix) > 0 {
		p = prefix[0]
	}
	return fmt.Sprintf("%s %s", p, randID())
}

// RandomSlug generates a URL-friendly slug from a name.
func RandomSlug(prefix ...string) string {
	name := RandomName(prefix...)
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}

// RandomPhone generates a random phone number.
func RandomPhone() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000000))
	return fmt.Sprintf("+1234567%03d", n.Int64())
}

// RandomURL generates a random URL.
func RandomURL(path ...string) string {
	p := "/test/" + randID()
	if len(path) > 0 {
		p = path[0]
	}
	return "https://example.com" + p
}

// Time helpers

// TimeAgo returns a time duration before now, truncated to seconds.
func TimeAgo(duration time.Duration) *time.Time {
	t := time.Now().Add(-duration).Truncate(time.Second).UTC()
	return &t
}

// TimeFromNow returns a time duration after now, truncated to seconds.
func TimeFromNow(duration time.Duration) *time.Time {
	t := time.Now().Add(duration).Truncate(time.Second).UTC()
	return &t
}

// TimeAt returns a specific time truncated to seconds in UTC.
func TimeAt(year int, month time.Month, day int, hour, min, sec int) *time.Time {
	t := time.Date(year, month, day, hour, min, sec, 0, time.UTC)
	return &t
}

// TimeAdd adds duration to t and returns a new pointer.
func TimeAdd(t *time.Time, duration time.Duration) *time.Time {
	if t == nil {
		return nil
	}
	result := t.Add(duration).Truncate(time.Second).UTC()
	return &result
}

// TimeDiff returns the duration between two times.
func TimeDiff(t1, t2 *time.Time) time.Duration {
	if t1 == nil || t2 == nil {
		return 0
	}
	return t2.Sub(*t1)
}

// Slice helpers

// StringSlice returns a pointer to a string slice.
func StringSlice(items ...string) *[]string {
	if len(items) == 0 {
		return nil
	}
	return &items
}

// IntPtr returns a pointer to an int.
func IntPtr(i int) *int {
	return &i
}

// Int32Ptr returns a pointer to an int32.
func Int32Ptr(i int32) *int32 {
	return &i
}

// Int64Ptr returns a pointer to an int64.
func Int64Ptr(i int64) *int64 {
	return &i
}

// Float32Ptr returns a pointer to a float32.
func Float32Ptr(f float32) *float32 {
	return &f
}

// BoolPtr returns a pointer to a bool.
func BoolPtr(b bool) *bool {
	return &b
}

// StringPtr returns a pointer to a string.
func StringPtr(s string) *string {
	return &s
}
