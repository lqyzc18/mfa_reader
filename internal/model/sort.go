package model

import "sort"

// SortPinned 稳定排序：置顶账号始终排在未置顶之前，组内相对顺序不变。
func SortPinned(accounts []MFAAccount) {
	sort.SliceStable(accounts, func(i, j int) bool {
		if accounts[i].Pinned == accounts[j].Pinned {
			return false
		}
		return accounts[i].Pinned
	})
}

// MoveAccount 在同一置顶分组内上下移动一项；越界或跨组则原样返回。
func MoveAccount(accounts []MFAAccount, index, delta int) []MFAAccount {
	j := index + delta
	if index < 0 || index >= len(accounts) || j < 0 || j >= len(accounts) {
		return accounts
	}
	if accounts[index].Pinned != accounts[j].Pinned {
		return accounts
	}
	accounts[index], accounts[j] = accounts[j], accounts[index]
	return accounts
}

// FindAccount 按名称+密钥定位账号，找不到返回 -1。
func FindAccount(accounts []MFAAccount, name, secret string) int {
	for i, a := range accounts {
		if a.AccountName == name && a.Secret == secret {
			return i
		}
	}
	return -1
}
