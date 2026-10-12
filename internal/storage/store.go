package storage

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"sync"

	"mfa_reader/internal/cryptutil"
	"mfa_reader/internal/model"
)

// ErrReadOnly 表示只读模式拒绝写入：数据文件读取失败时用它保护原文件不被覆盖。
var ErrReadOnly = errors.New("当前为只读模式，不会写入数据文件")

// Store 持有内存中的账号列表，并以明文 JSON 持久化。
type Store struct {
	mu       sync.RWMutex
	accounts []model.MFAAccount
	// readOnly 为 true 时拒绝一切落盘，仅在数据文件读取失败的降级启动中使用。
	readOnly bool
}

func NewEmpty() *Store {
	return &Store{accounts: []model.MFAAccount{}}
}

func (s *Store) FilePath() string {
	return dataFilePath()
}

// SetReadOnly 切换只读模式；降级启动时开启，避免把读取不到的数据文件覆盖成空表。
func (s *Store) SetReadOnly(on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readOnly = on
}

// ReadOnly 报告当前是否处于只读模式。
func (s *Store) ReadOnly() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.readOnly
}

// Open 读取明文 mfa.json；磁盘上是加密信封格式时会解密并把明文写回。
func Open() (*Store, error) {
	s := NewEmpty()
	raw, err := os.ReadFile(dataFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	trim := bytes.TrimSpace(trimBOM(raw))
	if len(trim) == 0 {
		return s, nil
	}

	accs, err := decodeAccounts(trim)
	if err != nil {
		return nil, err
	}
	model.SortPinned(accs)
	s.accounts = accs

	// 磁盘上是加密信封格式时写入明文，保证下次读取不再依赖解密。
	if trim[0] == '{' {
		if err := s.persistLocked(); err != nil {
			log.Printf("[storage] 数据文件转为明文失败: %v", err)
		} else {
			log.Printf("[storage] 已将加密信封格式的 mfa.json 转为明文 JSON")
		}
	}
	return s, nil
}

// decodeAccounts 解析账号数组，并把 OTP 参数补全为可直接出码的取值。
func decodeAccounts(trim []byte) ([]model.MFAAccount, error) {
	accs, err := parseAccounts(trim)
	if err != nil {
		return nil, err
	}
	repairParams(accs)
	return accs, nil
}

// parseAccounts 识别两种数据格式：明文账号数组，以及带 data 字段的加密信封。
func parseAccounts(trim []byte) ([]model.MFAAccount, error) {
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
		return nil, fmt.Errorf("该数据文件使用主密码加密，无法自动解锁，请改用明文 JSON 数据文件")
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

// repairParams 兼容历史数据：OTP 参数缺省或非法时回落到标准参数，
// 避免单个账号的字段异常导致整份数据文件读取失败。
func repairParams(accs []model.MFAAccount) {
	for i := range accs {
		if err := accs[i].NormalizeParams(); err != nil {
			log.Printf("[storage] 账号「%s」的 OTP 参数无效（%v），已回落为标准参数", accs[i].AccountName, err)
			accs[i].Digits = model.DefaultDigits
			accs[i].Period = model.DefaultPeriod
			accs[i].Algorithm = model.DefaultAlgorithm
		}
	}
}

func (s *Store) Snapshot() []model.MFAAccount {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.MFAAccount, len(s.accounts))
	copy(out, s.accounts)
	return out
}

// applyLocked 在持有写锁的前提下执行内存变更并落盘。
// 落盘失败时把内存还原为变更前的快照，避免「界面显示已添加/已删除，但磁盘上没有」的不一致。
func (s *Store) applyLocked(mutate func()) error {
	backup := slices.Clone(s.accounts)
	mutate()
	if err := s.persistLocked(); err != nil {
		s.accounts = backup
		return err
	}
	return nil
}

func (s *Store) Add(acc model.MFAAccount) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.applyLocked(func() {
		s.accounts = append(s.accounts, acc)
		model.SortPinned(s.accounts)
	})
}

func (s *Store) Update(oldName, oldSecret string, acc model.MFAAccount) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := model.FindAccount(s.accounts, oldName, oldSecret)
	if i < 0 {
		return fmt.Errorf("账号不存在")
	}
	return s.applyLocked(func() {
		s.accounts[i] = acc
		model.SortPinned(s.accounts)
	})
}

func (s *Store) Delete(name, secret string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := model.FindAccount(s.accounts, name, secret)
	if i < 0 {
		return nil
	}
	return s.applyLocked(func() {
		s.accounts = append(s.accounts[:i], s.accounts[i+1:]...)
	})
}

func (s *Store) TogglePin(name, secret string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := model.FindAccount(s.accounts, name, secret)
	if i < 0 {
		return fmt.Errorf("账号不存在")
	}
	return s.applyLocked(func() {
		s.accounts[i].Pinned = !s.accounts[i].Pinned
		model.SortPinned(s.accounts)
	})
}

// Move 在同一置顶分组内上下移动账号。
// 返回是否真的发生了移动：已在分组边界或跨分组时为 false。
func (s *Store) Move(name, secret string, delta int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := model.FindAccount(s.accounts, name, secret)
	if i < 0 {
		return false, fmt.Errorf("账号不存在")
	}

	moved := false
	err := s.applyLocked(func() {
		s.accounts, moved = model.MoveAccount(s.accounts, i, delta)
	})
	return moved, err
}

// Import 导入账号：只按密钥去重（同密钥视为同一账号），
// 名称重复时自动追加序号，保证每个导入的账号都能被搜索与定位。
func (s *Store) Import(accs []model.MFAAccount) (added, skipped int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	usedNames := make(map[string]bool, len(s.accounts)+len(accs))
	usedSecrets := make(map[string]bool, len(s.accounts)+len(accs))
	for _, a := range s.accounts {
		usedNames[strings.ToLower(a.AccountName)] = true
		usedSecrets[a.Secret] = true
	}

	var pending []model.MFAAccount
	for _, acc := range accs {
		if usedSecrets[acc.Secret] {
			skipped++
			continue
		}
		usedSecrets[acc.Secret] = true
		acc.AccountName = uniqueAccountName(acc.AccountName, usedNames)
		pending = append(pending, acc)
	}
	if len(pending) == 0 {
		return 0, skipped, nil
	}

	added = len(pending)
	if err := s.applyLocked(func() {
		s.accounts = append(s.accounts, pending...)
		model.SortPinned(s.accounts)
	}); err != nil {
		// 落盘失败已回滚内存，不能把 added 报给调用方弹「已导入 N 个」的成功提示。
		return 0, skipped, err
	}
	return added, skipped, nil
}

// uniqueAccountName 在名称冲突时追加序号（「扫码导入」→「扫码导入 2」），并把结果计入已用名称。
func uniqueAccountName(name string, used map[string]bool) string {
	key := strings.ToLower(name)
	if !used[key] {
		used[key] = true
		return name
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s %d", name, i)
		key = strings.ToLower(candidate)
		if !used[key] {
			used[key] = true
			return candidate
		}
	}
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
	trim := bytes.TrimSpace(trimBOM(raw))
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
	if s.readOnly {
		return ErrReadOnly
	}
	data, err := json.MarshalIndent(s.accounts, "", "\t")
	if err != nil {
		return err
	}
	return writeFileAtomic(dataFilePath(), data)
}
