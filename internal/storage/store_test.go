package storage

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"mfa_reader/internal/cryptutil"
	"mfa_reader/internal/model"
)

func useTempFile(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	orig := dataFilePath
	dataFilePath = func() string { return filepath.Join(tmp, "mfa.json") }
	t.Cleanup(func() { dataFilePath = orig })
}

func TestStoreSaveLoadPlaintext(t *testing.T) {
	useTempFile(t)
	s := NewEmpty()
	accs := []model.MFAAccount{
		{AccountName: "Google", Secret: "JBSWY3DPEHPK3PXP"},
		{AccountName: "GitHub", Secret: "234567ABCDEF2345", Pinned: true},
	}
	for _, a := range accs {
		if err := s.Add(a); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}

	raw, err := os.ReadFile(dataFilePath())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte("JBSWY3DPEHPK3PXP")) {
		t.Fatal("expected plaintext secret on disk")
	}

	loaded, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	got := loaded.Snapshot()
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if !got[0].Pinned || got[0].AccountName != "GitHub" {
		t.Fatalf("pinned account should be first: %+v", got[0])
	}
}

func TestStoreImportDuplicate(t *testing.T) {
	useTempFile(t)
	s := NewEmpty()
	_ = s.Add(model.MFAAccount{AccountName: "Google", Secret: "JBSWY3DPEHPK3PXP"})
	added, skipped, err := s.Import([]model.MFAAccount{
		{AccountName: "Google", Secret: "JBSWY3DPEHPK3PXP"},
		{AccountName: "GitHub", Secret: "234567ABCDEF2345"},
	})
	if err != nil || added != 1 || skipped != 1 {
		t.Fatalf("added=%d skipped=%d err=%v", added, skipped, err)
	}
}

func TestLoadBackup(t *testing.T) {
	useTempFile(t)
	s := NewEmpty()
	_ = s.Add(model.MFAAccount{AccountName: "Google", Secret: "JBSWY3DPEHPK3PXP"})
	accs, err := LoadBackup(dataFilePath())
	if err != nil || len(accs) != 1 {
		t.Fatalf("n=%d err=%v", len(accs), err)
	}
}

func TestOpenMissingAndInvalid(t *testing.T) {
	useTempFile(t)
	s, err := Open()
	if err != nil || len(s.Snapshot()) != 0 {
		t.Fatalf("missing file: %+v %v", s, err)
	}

	if err := os.WriteFile(dataFilePath(), []byte("not-json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(); err == nil {
		t.Fatal("invalid json should fail")
	}
}

func TestOpenRewritesLeftoverEncryptedFile(t *testing.T) {
	useTempFile(t)
	plain, _ := json.Marshal([]model.MFAAccount{{AccountName: "Google", Secret: "JBSWY3DPEHPK3PXP"}})
	blob, err := cryptutil.Protect(plain)
	if err != nil {
		t.Fatal(err)
	}
	env, _ := json.Marshal(map[string]any{
		"version": 2,
		"mode":    "dpapi",
		"data":    base64.StdEncoding.EncodeToString(blob),
	})
	if err := os.WriteFile(dataFilePath(), env, 0600); err != nil {
		t.Fatal(err)
	}

	s, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Snapshot()) != 1 || s.Snapshot()[0].AccountName != "Google" {
		t.Fatalf("%+v", s.Snapshot())
	}
	raw, _ := os.ReadFile(dataFilePath())
	if !bytes.Contains(raw, []byte("JBSWY3DPEHPK3PXP")) {
		t.Fatal("encrypted file should be rewritten as plaintext")
	}
}

func TestSaveEmptySlice(t *testing.T) {
	useTempFile(t)
	s := NewEmpty()
	if err := s.persistLocked(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(dataFilePath())
	if err != nil {
		t.Fatal(err)
	}
	var accs []model.MFAAccount
	if err := json.Unmarshal(raw, &accs); err != nil {
		t.Fatal(err)
	}
	if len(accs) != 0 {
		t.Fatalf("len=%d", len(accs))
	}
}
