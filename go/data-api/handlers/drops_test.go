package handlers

import (
	"regexp"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestGenerateSlug_Length(t *testing.T) {
	slug := generateSlug()
	if len(slug) != 8 {
		t.Errorf("expected slug length 8, got %d: %q", len(slug), slug)
	}
}

func TestGenerateSlug_Charset(t *testing.T) {
	for i := 0; i < 50; i++ {
		slug := generateSlug()
		matched, _ := regexp.MatchString(`^[a-z0-9]+$`, slug)
		if !matched {
			t.Errorf("slug contains invalid characters: %q", slug)
		}
	}
}

func TestGenerateSlug_Uniqueness(t *testing.T) {
	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		s := generateSlug()
		if seen[s] {
			t.Errorf("slug collision detected: %q", s)
		}
		seen[s] = true
	}
}

func TestGenerateSessionToken_RawLength(t *testing.T) {
	raw, _, err := generateSessionToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(raw) != 64 {
		t.Errorf("expected raw token length 64, got %d", len(raw))
	}
}

func TestGenerateSessionToken_RawIsHex(t *testing.T) {
	raw, _, err := generateSessionToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	matched, _ := regexp.MatchString(`^[0-9a-f]+$`, raw)
	if !matched {
		t.Errorf("raw token is not lowercase hex: %q", raw)
	}
}

func TestGenerateSessionToken_HashMatchesRaw(t *testing.T) {
	raw, hashed, err := generateSessionToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(raw)); err != nil {
		t.Errorf("hashed token does not verify against raw: %v", err)
	}
}

func TestGenerateSessionToken_Unique(t *testing.T) {
	raw1, _, _ := generateSessionToken()
	raw2, _, _ := generateSessionToken()
	if raw1 == raw2 {
		t.Error("two consecutive tokens are identical")
	}
}
