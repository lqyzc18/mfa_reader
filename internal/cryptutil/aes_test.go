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
