package storage

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"mfa_reader/internal/model"
)

var dataFilePath = func() string {
	exePath, err := os.Executable()
	if err != nil {
		log.Printf("[storage] 获取可执行文件路径失败: %v，使用当前目录", err)
		return "mfa.json"
	}
	return filepath.Join(filepath.Dir(exePath), "mfa.json")
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
	return os.WriteFile(filePath, data, 0644)
}
