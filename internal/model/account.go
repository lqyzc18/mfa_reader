package model

type MFAAccount struct {
	AccountName string `json:"accountName"`
	Time        int64  `json:"time"`
	Secret      string `json:"secret"`
}