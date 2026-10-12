//go:build windows

package singleinstance

import (
	"errors"
	"testing"
)

// 互斥量必须真正互斥：第二次获取返回 ErrAlreadyRunning，释放后可再次获取。
func TestAcquireIsExclusive(t *testing.T) {
	const name = `Local\mfa_reader_test_singleinstance`

	release, err := Acquire(name)
	if err != nil {
		t.Fatalf("首次获取失败: %v", err)
	}

	if _, err := Acquire(name); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("第二次获取应返回 ErrAlreadyRunning，实际: %v", err)
	}

	release()

	release2, err := Acquire(name)
	if err != nil {
		t.Fatalf("释放后应能重新获取: %v", err)
	}
	release2()

	// 重复调用释放函数不应 panic。
	release()
}

// 不同名称互不影响，避免误判为「已在运行」。
func TestAcquireDifferentNames(t *testing.T) {
	releaseA, err := Acquire(`Local\mfa_reader_test_a`)
	if err != nil {
		t.Fatal(err)
	}
	defer releaseA()

	releaseB, err := Acquire(`Local\mfa_reader_test_b`)
	if err != nil {
		t.Fatalf("不同名称不应互斥: %v", err)
	}
	releaseB()
}
