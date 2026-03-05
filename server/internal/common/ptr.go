package common

import "time"

// Ptr returns a pointer to the given value.
func Ptr[T any](v T) *T {
	return &v
}

// PtrOr returns the value of the pointer if it is not nil, otherwise returns the fallback value.
func PtrOr[T any](v *T, fallback T) T {
	if v == nil {
		return fallback
	}
	return *v
}

// UnixToPtrTime converts a unix timestamp pointer (int64) to a time.Time pointer.
func UnixToPtrTime(unix *int64) *time.Time {
	if unix == nil {
		return nil
	}
	t := time.Unix(*unix, 0)
	return &t
}

// UnixToPtrTimeWithDelta converts a unix timestamp pointer (int64) to a time.Time pointer and adds a delta.
func UnixToPtrTimeWithDelta(unix *int64, delta time.Duration) *time.Time {
	t := UnixToPtrTime(unix)
	if t == nil {
		return nil
	}
	res := t.Add(delta)
	return &res
}
