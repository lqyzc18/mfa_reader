//go:build !windows

package winpos

func Get(string) (int, int, bool) { return 0, 0, false }
func Set(string, int, int)        {}
