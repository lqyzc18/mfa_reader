package storage

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"mfa_reader/internal/model"

	"github.com/duke-git/lancet/v2/fileutil"
)

const filePath = "mfa.json"

func LoadMFAAccounts() []model.MFAAccount {
	var accounts []model.MFAAccount

	if !fileutil.IsExist(filePath) {
		return accounts
	}

	content, err := fileutil.ReadFileToString(filePath)
	if err != nil {
		return accounts
	}

	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "[") {
		var jsonAccounts []model.MFAAccount
		if err := json.Unmarshal([]byte(content), &jsonAccounts); err == nil {
			accounts = jsonAccounts
		}
	} else {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.Split(line, ",")
			if len(parts) >= 3 {
				accounts = append(accounts, model.MFAAccount{
					AccountName: parts[0],
					Time:        time.Now().UnixMilli(),
					Secret:      parts[2],
				})
			} else if len(parts) == 2 {
				accounts = append(accounts, model.MFAAccount{
					AccountName: parts[0],
					Time:        time.Now().UnixMilli(),
					Secret:      parts[1],
				})
			}
		}
	}
	return accounts
}

func SaveMFAAccounts(accounts []model.MFAAccount) error {
	data, err := json.MarshalIndent(accounts, "", "\t")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}