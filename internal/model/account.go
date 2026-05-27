package model

import "strings"

type MFAAccount struct {
	AccountName string `json:"accountName"`
	Secret      string `json:"secret"`
}

func (a *MFAAccount) NormalizeSecret() string {
	s := strings.ToUpper(strings.TrimSpace(a.Secret))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	return strings.TrimRight(s, "=")
}
