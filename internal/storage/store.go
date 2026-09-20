package storage

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"mfa_reader/internal/cryptutil"
	"mfa_reader/internal/model"
)

// Store 持有内存中的账号列表，并以明文 JSON 持久化。
type Store struct {
	mu       sync.RWMutex
	accounts []model.MFAAccount
}

func NewEmpty() *Store {
	return &Store{accounts: []model.MFAAccount{}}
}

func (s *Store) FilePath() string {
	return dataFilePath()
}

// Open 读取明文 mfa.json。若遇到上一版加密信封，会尝试解密并写回明文。
func Open() (*Store, error) {
	s := NewEmpty()
	raw, err := os.ReadFile(dataFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	trim := bytes.TrimSpace(raw)
	if len(trim) == 0 {
		return s, nil
	}

	accs, err := decodeAccounts(trim)
	if err != nil {
		return nil, err
	}
	model.SortPinned(accs)
	s.accounts = accs

	// 若磁盘上仍是加密信封，立刻改回明文，避免下次无法读取。
	if len(trim) > 0 && trim[0] == '{' {
		if err := s.persistLocked(); err != nil {
			log.Printf("[storage] 加密文件改回明文失败: %v", err)
		} else {
			log.Printf("[storage] 已将加密 mfa.json 改回明文 JSON")
		}
	}
	return s, nil
}

func decodeAccounts(trim []byte) ([]model.MFAAccount, error) {
	var accs []model.MFAAccount
	if trim[0] == '[' {
		if err := json.Unmarshal(trim, &accs); err != nil {
			return nil, fmt.Errorf("解析 JSON 失败: %w", err)
		}
		if accs == nil {
			accs = []model.MFAAccount{}
		}
		return accs, nil
	}

	var env struct {
		Version int    `json:"version"`
		Mode    string `json:"mode"`
		Salt    string `json:"salt,omitempty"`
		Data    string `json:"data"`
	}
	if err := json.Unmarshal(trim, &env); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}
	if env.Data == "" {
		return nil, fmt.Errorf("不支持的数据文件格式")
	}
	if env.Mode == "password" {
		return nil, fmt.Errorf("该文件是主密码加密格式，当前版本已改回明文存储，无法自动解锁")
	}
	blob, err := base64.StdEncoding.DecodeString(env.Data)
	if err != nil {
		return nil, fmt.Errorf("密文损坏: %w", err)
	}
	plain, err := cryptutil.Unprotect(blob)
	if err != nil {
		return nil, fmt.Errorf("解密失败: %w", err)
	}
	if err := json.Unmarshal(plain, &accs); err != nil {
		return nil, fmt.Errorf("解密后数据损坏: %w", err)
	}
	if accs == nil {
		accs = []model.MFAAccount{}
	}
	return accs, nil
}

func (s *Store) Snapshot() []model.MFAAccount {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.MFAAccount, len(s.accounts))
	copy(out, s.accounts)
	return out
}

func (s *Store) Add(acc model.MFAAccount) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accounts = append(s.accounts, acc)
	model.SortPinned(s.accounts)
	return s.persistLocked()
}

func (s *Store) Update(oldName, oldSecret string, acc model.MFAAccount) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := model.FindAccount(s.accounts, oldName, oldSecret)
	if i < 0 {
		return fmt.Errorf("账号不存在")
	}
	s.accounts[i] = acc
	model.SortPinned(s.accounts)
	return s.persistLocked()
}

func (s *Store) Delete(name, secret string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := model.FindAccount(s.accounts, name, secret)
	if i < 0 {
		return nil
	}
	s.accounts = append(s.accounts[:i], s.accounts[i+1:]...)
	return s.persistLocked()
}

func (s *Store) TogglePin(name, secret string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := model.FindAccount(s.accounts, name, secret)
	if i < 0 {
		return fmt.Errorf("账号不存在")
	}
	s.accounts[i].Pinned = !s.accounts[i].Pinned
	model.SortPinned(s.accounts)
	return s.persistLocked()
}

func (s *Store) Move(name, secret string, delta int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := model.FindAccount(s.accounts, name, secret)
	if i < 0 {
		return fmt.Errorf("账号不存在")
	}
	s.accounts = model.MoveAccount(s.accounts, i, delta)
	return s.persistLocked()
}

func (s *Store) Import(accs []model.MFAAccount) (added, skipped int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, acc := range accs {
		if dup, _ := s.hasDuplicateLocked(acc.AccountName, acc.Secret, "", ""); dup {
			skipped++
			continue
		}
		s.accounts = append(s.accounts, acc)
		added++
	}
	if added == 0 {
		return added, skipped, nil
	}
	model.SortPinned(s.accounts)
	return added, skipped, s.persistLocked()
}

func (s *Store) HasDuplicate(name, secret, exceptName, exceptSecret string) (bool, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hasDuplicateLocked(name, secret, exceptName, exceptSecret)
}

func (s *Store) hasDuplicateLocked(name, secret, exceptName, exceptSecret string) (bool, string) {
	for _, a := range s.accounts {
		if a.AccountName == exceptName && a.Secret == exceptSecret {
			continue
		}
		if a.Secret == secret {
			return true, "该密钥已存在（账号: " + a.AccountName + "）"
		}
		if strings.EqualFold(a.AccountName, name) {
			return true, "账号名称已存在"
		}
	}
	return false, ""
}

// LoadBackup 读取备份 JSON（账号数组）。
func LoadBackup(path string) ([]model.MFAAccount, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	trim := bytes.TrimSpace(raw)
	if len(trim) == 0 {
		return []model.MFAAccount{}, nil
	}
	accs, err := decodeAccounts(trim)
	if err != nil {
		return nil, err
	}
	return normalizeImported(accs)
}

func (s *Store) persistLocked() error {
	data, err := json.MarshalIndent(s.accounts, "", "\t")
	if err != nil {
		return err
	}
	return writeFileAtomic(dataFilePath(), data)
}
