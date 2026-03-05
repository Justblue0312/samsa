package pwd

import (
	"testing"
)

func TestHash(t *testing.T) {
	password := "password123"
	hashedPassword, err := Hash(password)
	if err != nil {
		t.Errorf("failed to hash password: %v", err)
	}

	if len(hashedPassword) == 0 {
		t.Errorf("hashed password is empty")
	}
}

func TestVerify(t *testing.T) {
	password := "password123"
	hashedPassword, err := Hash(password)
	if err != nil {
		t.Errorf("failed to hash password: %v", err)
	}

	err = Verify(hashedPassword, password)
	if err != nil {
		t.Errorf("failed to verify password: %v", err)
	}

	err = Verify(hashedPassword, "wrongpassword")
	if err == nil {
		t.Errorf("expected error for wrong password")
	}
}
