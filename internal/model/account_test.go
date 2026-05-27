package model

import "testing"

func TestNormalizeSecret(t *testing.T) {
	tests := []struct {
		name     string
		secret   string
		expected string
	}{
		{
			name:     "标准密钥",
			secret:   "JBSWY3DPEHPK3PXP",
			expected: "JBSWY3DPEHPK3PXP",
		},
		{
			name:     "带空格的密钥",
			secret:   "JBSW Y3DP EHPK 3PXP",
			expected: "JBSWY3DPEHPK3PXP",
		},
		{
			name:     "带横线的密钥",
			secret:   "JBSW-Y3DP-EHPK-3PXP",
			expected: "JBSWY3DPEHPK3PXP",
		},
		{
			name:     "小写密钥",
			secret:   "jbswy3dpehpk3pxp",
			expected: "JBSWY3DPEHPK3PXP",
		},
		{
			name:     "带填充符的密钥",
			secret:   "JBSWY3DPEHPK3PXP===",
			expected: "JBSWY3DPEHPK3PXP",
		},
		{
			name:     "混合格式",
			secret:   " jbsw-y3dp-ehpk-3pxp ",
			expected: "JBSWY3DPEHPK3PXP",
		},
		{
			name:     "带数字2-7的密钥",
			secret:   "234567ABCDEF2345",
			expected: "234567ABCDEF2345",
		},
		{
			name:     "空密钥",
			secret:   "",
			expected: "",
		},
		{
			name:     "只有空格和横线",
			secret:   " - - ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acc := MFAAccount{
				AccountName: "test",
				Secret:      tt.secret,
			}
			result := acc.NormalizeSecret()
			if result != tt.expected {
				t.Errorf("NormalizeSecret() = %v, want %v", result, tt.expected)
			}
		})
	}
}
