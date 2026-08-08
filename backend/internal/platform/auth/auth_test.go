package auth

import (
	"testing"
	"time"
)

func TestHashAndVerify(t *testing.T) {
	hash, err := Hash("s3cret-password")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	ok, err := Verify("s3cret-password", hash)
	if err != nil || !ok {
		t.Fatalf("verify correct password: ok=%v err=%v", ok, err)
	}

	ok, _ = Verify("wrong-password", hash)
	if ok {
		t.Fatal("verify should fail for the wrong password")
	}

	hash2, _ := Hash("s3cret-password")
	if hash == hash2 {
		t.Fatal("two hashes of the same password should differ (random salt)")
	}
}

func TestIssueAndParse(t *testing.T) {
	issuer := NewTokenIssuer("test-secret", time.Hour)

	token, err := issuer.Issue("customer-123")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	sub, err := issuer.Parse(token)
	if err != nil || sub != "customer-123" {
		t.Fatalf("parse: sub=%q err=%v", sub, err)
	}
}
