package pbkdf2

import (
	"bytes"
	"testing"
)

func TestHashPasswordAndMatch(t *testing.T) {
	ph := HashPassword("correct-horse-battery-staple")

	if !MatchPassword("correct-horse-battery-staple", ph) {
		t.Error("MatchPassword should return true for the correct password")
	}
	if MatchPassword("wrong-password", ph) {
		t.Error("MatchPassword should return false for an incorrect password")
	}
}

func TestHashPasswordWithSaltIsDeterministic(t *testing.T) {
	salt := []byte("fixed-salt-value")

	ph1 := HashPasswordWithSalt("mypassword", salt)
	ph2 := HashPasswordWithSalt("mypassword", salt)

	if !bytes.Equal(ph1.Hash, ph2.Hash) {
		t.Error("hashing the same password with the same salt should produce the same hash")
	}
	if !bytes.Equal(ph1.Salt, salt) || !bytes.Equal(ph2.Salt, salt) {
		t.Error("PasswordHash.Salt should equal the salt that was passed in")
	}
}

func TestHashPasswordUsesRandomSalt(t *testing.T) {
	ph1 := HashPassword("samepassword")
	ph2 := HashPassword("samepassword")

	if bytes.Equal(ph1.Salt, ph2.Salt) {
		t.Error("two calls to HashPassword should generate different random salts")
	}
	if bytes.Equal(ph1.Hash, ph2.Hash) {
		t.Error("hashes should differ when salts differ, even for the same password")
	}
}

func TestMatchPasswordRejectsDifferentHashLength(t *testing.T) {
	ph := HashPassword("somepassword")
	shortHash := &PasswordHash{Hash: ph.Hash[:len(ph.Hash)-1], Salt: ph.Salt}

	if MatchPassword("somepassword", shortHash) {
		t.Error("MatchPassword should return false when the stored hash length does not match")
	}
}
