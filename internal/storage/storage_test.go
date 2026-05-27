package storage

import (
	"os"
	"path/filepath"
	"testing"

	"mfa_reader/internal/model"
)

func TestSaveAndLoadMFAAccounts(t *testing.T) {
	tmpDir := t.TempDir()
	originalDataFilePath := dataFilePath
	defer func() { dataFilePath = originalDataFilePath }()

	dataFilePath = func() string {
		return filepath.Join(tmpDir, "mfa.json")
	}

	accounts := []model.MFAAccount{
		{AccountName: "Google", Secret: "JBSWY3DPEHPK3PXP"},
		{AccountName: "GitHub", Secret: "234567ABCDEF2345"},
	}

	err := SaveMFAAccounts(accounts)
	if err != nil {
		t.Fatalf("SaveMFAAccounts() error = %v", err)
	}

	loaded := LoadMFAAccounts()
	if len(loaded) != len(accounts) {
		t.Fatalf("LoadMFAAccounts() returned %d accounts, want %d", len(loaded), len(accounts))
	}

	for i, acc := range loaded {
		if acc.AccountName != accounts[i].AccountName {
			t.Errorf("Account[%d].AccountName = %v, want %v", i, acc.AccountName, accounts[i].AccountName)
		}
		if acc.Secret != accounts[i].Secret {
			t.Errorf("Account[%d].Secret = %v, want %v", i, acc.Secret, accounts[i].Secret)
		}
	}
}

func TestLoadMFAAccounts_FileNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	originalDataFilePath := dataFilePath
	defer func() { dataFilePath = originalDataFilePath }()

	dataFilePath = func() string {
		return filepath.Join(tmpDir, "nonexistent.json")
	}

	loaded := LoadMFAAccounts()
	if loaded == nil {
		t.Fatal("LoadMFAAccounts() returned nil for nonexistent file")
	}
	if len(loaded) != 0 {
		t.Errorf("LoadMFAAccounts() returned %d accounts, want 0", len(loaded))
	}
}

func TestLoadMFAAccounts_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	originalDataFilePath := dataFilePath
	defer func() { dataFilePath = originalDataFilePath }()

	filePath := filepath.Join(tmpDir, "mfa.json")
	err := os.WriteFile(filePath, []byte("invalid json"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	dataFilePath = func() string {
		return filePath
	}

	loaded := LoadMFAAccounts()
	if loaded == nil {
		t.Fatal("LoadMFAAccounts() returned nil for invalid JSON")
	}
	if len(loaded) != 0 {
		t.Errorf("LoadMFAAccounts() returned %d accounts, want 0", len(loaded))
	}
}

func TestSaveMFAAccounts_EmptySlice(t *testing.T) {
	tmpDir := t.TempDir()
	originalDataFilePath := dataFilePath
	defer func() { dataFilePath = originalDataFilePath }()

	dataFilePath = func() string {
		return filepath.Join(tmpDir, "mfa.json")
	}

	err := SaveMFAAccounts([]model.MFAAccount{})
	if err != nil {
		t.Fatalf("SaveMFAAccounts() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "mfa.json"))
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	expected := "[]"
	if string(data) != expected {
		t.Errorf("Saved content = %q, want %q", string(data), expected)
	}
}
