package cryptutil

import "testing"

func TestAESGCMRoundTrip(t *testing.T) {
	key := make([]byte, aesKeySize)
	for i := range key {
		key[i] = byte(i)
	}
	plain := []byte(`[{"accountName":"Google"}]`)
	blob, err := EncryptAESGCM(key, plain)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecryptAESGCM(key, blob)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(plain) {
		t.Fatalf("got %s", got)
	}
	if _, err := DecryptAESGCM(key, blob[:4]); err == nil {
		t.Fatal("short cipher should fail")
	}
}

func TestDeriveKey(t *testing.T) {
	salt, err := NewSalt()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DeriveKey("123", salt); err != ErrShortPassword {
		t.Fatalf("short password: %v", err)
	}
	k1, err := DeriveKey("password", salt)
	if err != nil {
		t.Fatal(err)
	}
	k2, err := DeriveKey("password", salt)
	if err != nil {
		t.Fatal(err)
	}
	if string(k1) != string(k2) || len(k1) != aesKeySize {
		t.Fatal("derive not deterministic")
	}
}

func TestProtectRoundTrip(t *testing.T) {
	plain := []byte("hello-mfa")
	blob, err := Protect(plain)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Unprotect(blob)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(plain) {
		t.Fatalf("got %s", got)
	}
}
