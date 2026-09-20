package storage

import (
	"log"
	"os"
	"path/filepath"
	"strings"
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

// writeFileAtomic 原子写入：先写同目录临时文件并 fsync，再 rename 替换目标。
func writeFileAtomic(filePath string, data []byte) error {
	tmpPath := filePath + ".tmp"

	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, filePath); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}
