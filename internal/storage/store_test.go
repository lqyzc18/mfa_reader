package storage

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
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

// 名称相同但密钥不同的账号都要导入成功：账号由「名称+密钥」共同定位，
// 导入只按密钥去重，重名时自动追加序号。
func TestImportKeepsDistinctAccountsWithSameName(t *testing.T) {
	useTempFile(t)
	s := NewEmpty()

	added, skipped, err := s.Import([]model.MFAAccount{{AccountName: "扫码导入", Secret: "JBSWY3DPEHPK3PXP"}})
	if err != nil || added != 1 || skipped != 0 {
		t.Fatalf("first import: added=%d skipped=%d err=%v", added, skipped, err)
	}

	added, skipped, err = s.Import([]model.MFAAccount{{AccountName: "扫码导入", Secret: "234567ABCDEF2345"}})
	if err != nil || added != 1 || skipped != 0 {
		t.Fatalf("second import: added=%d skipped=%d err=%v", added, skipped, err)
	}

	got := s.Snapshot()
	if len(got) != 2 {
		t.Fatalf("both accounts should be kept: %+v", got)
	}
	names := []string{got[0].AccountName, got[1].AccountName}
	if names[0] == names[1] {
		t.Fatalf("duplicate names should be suffixed: %v", names)
	}
	if names[1] != "扫码导入 2" {
		t.Fatalf("second account should be renamed: %v", names)
	}
}

// 同一批次内的重复密钥只保留一条。
func TestImportDeduplicatesWithinBatch(t *testing.T) {
	useTempFile(t)
	s := NewEmpty()

	added, skipped, err := s.Import([]model.MFAAccount{
		{AccountName: "A", Secret: "JBSWY3DPEHPK3PXP"},
		{AccountName: "B", Secret: "JBSWY3DPEHPK3PXP"},
	})
	if err != nil || added != 1 || skipped != 1 {
		t.Fatalf("added=%d skipped=%d err=%v", added, skipped, err)
	}
}

// 记事本等编辑器会把 UTF-8 文件写成带 BOM 的形式，读取时必须兼容。
func TestOpenToleratesBOM(t *testing.T) {
	useTempFile(t)
	content := append([]byte{0xEF, 0xBB, 0xBF}, []byte(`[{"accountName":"Google","secret":"JBSWY3DPEHPK3PXP"}]`)...)
	if err := os.WriteFile(dataFilePath(), content, 0600); err != nil {
		t.Fatal(err)
	}

	s, err := Open()
	if err != nil {
		t.Fatalf("Open with BOM: %v", err)
	}
	if got := s.Snapshot(); len(got) != 1 || got[0].AccountName != "Google" {
		t.Fatalf("%+v", got)
	}
}

// 只读模式下所有写操作都必须被拒绝，内存保持不变。
func TestReadOnlyStoreRejectsWrites(t *testing.T) {
	useTempFile(t)
	s := NewEmpty()
	if err := s.Add(model.MFAAccount{AccountName: "Keep", Secret: "JBSWY3DPEHPK3PXP"}); err != nil {
		t.Fatal(err)
	}

	s.SetReadOnly(true)
	if !s.ReadOnly() {
		t.Fatal("store should report read-only")
	}

	if err := s.Add(model.MFAAccount{AccountName: "Ghost", Secret: "234567ABCDEF2345"}); !errors.Is(err, ErrReadOnly) {
		t.Fatalf("Add in read-only mode: %v", err)
	}
	if err := s.Delete("Keep", "JBSWY3DPEHPK3PXP"); !errors.Is(err, ErrReadOnly) {
		t.Fatalf("Delete in read-only mode: %v", err)
	}
	if _, _, err := s.Import([]model.MFAAccount{{AccountName: "Ghost", Secret: "234567ABCDEF2345"}}); !errors.Is(err, ErrReadOnly) {
		t.Fatalf("Import in read-only mode: %v", err)
	}

	if got := s.Snapshot(); len(got) != 1 || got[0].AccountName != "Keep" {
		t.Fatalf("read-only mode must not change memory: %+v", got)
	}

	// 磁盘上的内容同样保持不变。
	raw, err := os.ReadFile(dataFilePath())
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("Ghost")) {
		t.Fatal("read-only mode must not write to disk")
	}
}

// 历史数据里的非法 OTP 参数不应导致整份文件读取失败，而是回落为标准参数。
func TestOpenRepairsInvalidParams(t *testing.T) {
	useTempFile(t)
	content := []byte(`[{"accountName":"Legacy","secret":"JBSWY3DPEHPK3PXP","digits":7,"period":9999,"algorithm":"MD5"}]`)
	if err := os.WriteFile(dataFilePath(), content, 0600); err != nil {
		t.Fatal(err)
	}

	s, err := Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got := s.Snapshot()
	if len(got) != 1 {
		t.Fatalf("%+v", got)
	}
	if !got[0].IsDefaultParams() {
		t.Fatalf("invalid params should fall back to defaults: %+v", got[0])
	}
}

// 边界处的移动返回 false，供界面给出「已在分组顶部/底部」提示。
func TestMoveReportsWhetherItMoved(t *testing.T) {
	useTempFile(t)
	s := NewEmpty()
	if err := s.Add(model.MFAAccount{AccountName: "A", Secret: "JBSWY3DPEHPK3PXP"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(model.MFAAccount{AccountName: "B", Secret: "234567ABCDEF2345"}); err != nil {
		t.Fatal(err)
	}

	moved, err := s.Move("A", "JBSWY3DPEHPK3PXP", -1)
	if err != nil {
		t.Fatal(err)
	}
	if moved {
		t.Fatal("moving the first account up should not move")
	}

	moved, err = s.Move("A", "JBSWY3DPEHPK3PXP", 1)
	if err != nil || !moved {
		t.Fatalf("moved=%v err=%v", moved, err)
	}
	got := s.Snapshot()
	if got[0].AccountName != "B" || got[1].AccountName != "A" {
		t.Fatalf("unexpected order: %+v", got)
	}
}
