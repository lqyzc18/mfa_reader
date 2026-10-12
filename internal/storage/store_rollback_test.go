package storage

import (
	"os"
	"path/filepath"
	"testing"

	"mfa_reader/internal/model"
)

// readOnlyDir 把数据文件指向一个无法写入的目录，用于模拟落盘失败。
// Windows 上把目录 ACL 改成只读比较麻烦，这里改用一个「路径是目录」的方式：
// mfa.json 指向 tmp/dir/mfa.json，而 tmp/dir 本身是个目录，
// OpenFile 打开 tmp/dir/mfa.json.tmp 仍可写，但 rename 到已存在的目录会失败。
func failPersistDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	blocked := filepath.Join(tmp, "blocked")
	if err := os.Mkdir(blocked, 0o755); err != nil {
		t.Fatal(err)
	}
	return blocked
}

func TestStoreAddRollsBackWhenPersistFails(t *testing.T) {
	tmp := t.TempDir()
	blocked := failPersistDir(t)
	orig := dataFilePath
	// 第一次写入正常，之后把路径切到无法 rename 的位置模拟失败。
	dataFilePath = func() string { return filepath.Join(tmp, "mfa.json") }
	t.Cleanup(func() { dataFilePath = orig })

	s := NewEmpty()
	seed := model.MFAAccount{AccountName: "Seed", Secret: "JBSWY3DPEHPK3PXP"}
	if err := s.Add(seed); err != nil {
		t.Fatalf("seed Add: %v", err)
	}

	dataFilePath = func() string { return blocked }
	if err := s.Add(model.MFAAccount{AccountName: "Ghost", Secret: "234567ABCDEF2345"}); err == nil {
		t.Fatal("expected persist to fail")
	}

	// 内存必须回到失败前的状态，否则界面会显示一个磁盘上并不存在的账号。
	got := s.Snapshot()
	if len(got) != 1 || got[0].AccountName != "Seed" {
		t.Fatalf("in-memory state not rolled back: %+v", got)
	}
}

func TestStoreDeleteRollsBackWhenPersistFails(t *testing.T) {
	tmp := t.TempDir()
	blocked := failPersistDir(t)
	orig := dataFilePath
	dataFilePath = func() string { return filepath.Join(tmp, "mfa.json") }
	t.Cleanup(func() { dataFilePath = orig })

	s := NewEmpty()
	if err := s.Add(model.MFAAccount{AccountName: "Keep", Secret: "JBSWY3DPEHPK3PXP"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	dataFilePath = func() string { return blocked }
	if err := s.Delete("Keep", "JBSWY3DPEHPK3PXP"); err == nil {
		t.Fatal("expected persist to fail")
	}
	if got := s.Snapshot(); len(got) != 1 {
		t.Fatalf("delete must be rolled back, got %+v", got)
	}
}

func TestStoreTogglePinRollsBackWhenPersistFails(t *testing.T) {
	tmp := t.TempDir()
	blocked := failPersistDir(t)
	orig := dataFilePath
	dataFilePath = func() string { return filepath.Join(tmp, "mfa.json") }
	t.Cleanup(func() { dataFilePath = orig })

	s := NewEmpty()
	if err := s.Add(model.MFAAccount{AccountName: "PinMe", Secret: "JBSWY3DPEHPK3PXP"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	dataFilePath = func() string { return blocked }
	if err := s.TogglePin("PinMe", "JBSWY3DPEHPK3PXP"); err == nil {
		t.Fatal("expected persist to fail")
	}
	if got := s.Snapshot(); len(got) != 1 || got[0].Pinned {
		t.Fatalf("pin toggle must be rolled back, got %+v", got)
	}
}

func TestStoreImportReportsZeroAddedOnPersistFailure(t *testing.T) {
	tmp := t.TempDir()
	blocked := failPersistDir(t)
	orig := dataFilePath
	dataFilePath = func() string { return filepath.Join(tmp, "mfa.json") }
	t.Cleanup(func() { dataFilePath = orig })

	s := NewEmpty()
	dataFilePath = func() string { return blocked }

	added, skipped, err := s.Import([]model.MFAAccount{
		{AccountName: "A", Secret: "JBSWY3DPEHPK3PXP"},
	})
	if err == nil {
		t.Fatal("expected persist to fail")
	}
	// added 必须是 0：否则 UI 会弹出「已导入 1 个账号」的成功提示，与磁盘状态矛盾。
	if added != 0 {
		t.Fatalf("added should be 0 on failure, got %d", added)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d", skipped)
	}
	if got := s.Snapshot(); len(got) != 0 {
		t.Fatalf("import must be rolled back, got %+v", got)
	}
}

// 落盘失败不应影响后续操作：路径恢复正常后仍能继续写入。
func TestStoreRecoversAfterPersistFailure(t *testing.T) {
	tmp := t.TempDir()
	blocked := failPersistDir(t)
	orig := dataFilePath
	dataFilePath = func() string { return filepath.Join(tmp, "mfa.json") }
	t.Cleanup(func() { dataFilePath = orig })

	s := NewEmpty()
	dataFilePath = func() string { return blocked }
	_ = s.Add(model.MFAAccount{AccountName: "Ghost", Secret: "JBSWY3DPEHPK3PXP"})

	dataFilePath = func() string { return filepath.Join(tmp, "mfa.json") }
	if err := s.Add(model.MFAAccount{AccountName: "Real", Secret: "234567ABCDEF2345"}); err != nil {
		t.Fatalf("Add after recovery: %v", err)
	}

	got := s.Snapshot()
	if len(got) != 1 || got[0].AccountName != "Real" {
		t.Fatalf("unexpected state after recovery: %+v", got)
	}

	reloaded, err := Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if r := reloaded.Snapshot(); len(r) != 1 || r[0].AccountName != "Real" {
		t.Fatalf("disk should only contain the successfully written account: %+v", r)
	}
}
