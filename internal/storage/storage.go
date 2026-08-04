package storage

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"mfa_reader/internal/model"
)

var dataFilePath = func() string {
	return filepath.Join(resolveDataDir(), "mfa.json")
}

// resolveDataDir 返回数据文件目录：优先可执行文件所在目录。
// `go run` / 临时构建目录下回退到当前工作目录，避免写到系统临时目录。
func resolveDataDir() string {
	exePath, err := os.Executable()
	if err != nil {
		log.Printf("[storage] 获取可执行文件路径失败: %v，使用当前目录", err)
		if wd, err := os.Getwd(); err == nil {
			return wd
		}
		return "."
	}

	if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
		exePath = resolved
	}
	dir := filepath.Dir(exePath)

	slashDir := filepath.ToSlash(dir)
	if strings.Contains(slashDir, "/go-build") {
		if wd, err := os.Getwd(); err == nil {
			return wd
		}
	}
	return dir
}

func LoadMFAAccounts() []model.MFAAccount {
	filePath := dataFilePath()

	data, err := os.ReadFile(filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[storage] 读取文件失败: %v", err)
		}
		return []model.MFAAccount{}
	}

	var accounts []model.MFAAccount
	if err := json.Unmarshal(data, &accounts); err != nil {
		log.Printf("[storage] 解析 JSON 失败: %v", err)
		return []model.MFAAccount{}
	}

	return accounts
}

func SaveMFAAccounts(accounts []model.MFAAccount) error {
	filePath := dataFilePath()

	data, err := json.MarshalIndent(accounts, "", "\t")
	if err != nil {
		return err
	}
	// 0600：密钥文件仅当前用户可读写
	return os.WriteFile(filePath, data, 0600)
}
