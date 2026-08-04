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

func TestValidateSecret(t *testing.T) {
	tests := []struct {
		name      string
		secret    string
		wantError bool
	}{
		{name: "合法密钥", secret: "JBSWY3DPEHPK3PXP", wantError: false},
		{name: "过短", secret: "JBSWY3DPEHPK3PX", wantError: true},
		{name: "非法字符", secret: "JBSWY3DPEHPK3PX0", wantError: true},
		{name: "空密钥", secret: "", wantError: true},
		{name: "带填充符应先标准化", secret: "JBSWY3DPEHPK3PXP", wantError: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSecret(tt.secret)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateSecret(%q) error = %v, wantError %v", tt.secret, err, tt.wantError)
			}
		})
	}
}

func TestMatchName(t *testing.T) {
	acc := MFAAccount{AccountName: "Google Account"}

	if !acc.MatchName("") {
		t.Error("empty filter should match")
	}
	if !acc.MatchName("google") {
		t.Error("case-insensitive match failed")
	}
	if !acc.MatchName("ACCOUNT") {
		t.Error("substring match failed")
	}
	if acc.MatchName("GitHub") {
		t.Error("non-matching filter should not match")
	}
}
