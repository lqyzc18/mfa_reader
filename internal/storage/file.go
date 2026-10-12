package storage

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	mfaFileName     = "mfa.json"
	corruptFileTag  = ".corrupt-"
	corruptTimeFmt  = "20060102-150405"
	utf8BOM         = "\xef\xbb\xbf"
	goBuildTempMark = "/go-build"
)

// cachedDataDir 缓存目录解析结果：resolveDataDir 内部要调 os.Executable 与
// EvalSymlinks，而增删改每次落盘都会取一次路径，没必要反复付这两个系统调用。
var cachedDataDir = sync.OnceValue(resolveDataDir)

var dataFilePath = func() string {
	return filepath.Join(cachedDataDir(), mfaFileName)
}

// QuarantineDataFile 把无法解析的数据文件改名留档为 mfa.json.corrupt-<时间戳>，
// 供读取失败时保护原文件不被后续写入覆盖。文件不存在时返回空路径。
func QuarantineDataFile() (string, error) {
	src := dataFilePath()
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	dst := src + corruptFileTag + time.Now().Format(corruptTimeFmt)
	if err := os.Rename(src, dst); err != nil {
		return "", err
	}
	return dst, nil
}

// trimBOM 去掉文件开头的 UTF-8 BOM：记事本等编辑器保存 UTF-8 时会写入 BOM，
// 直接交给 json.Unmarshal 会因首字节非法而解析失败。
func trimBOM(raw []byte) []byte {
	return bytes.TrimPrefix(raw, []byte(utf8BOM))
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
	if strings.Contains(slashDir, goBuildTempMark) {
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
