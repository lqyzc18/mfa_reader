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

// Base32 的数据位数只能是 8n、8n+2、8n+4、8n+5、8n+7；
// 其余长度虽然字符集合法，但无法解码，必须在保存前拦截。
func TestValidateSecretBase32Lengths(t *testing.T) {
	alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	for length := 16; length <= 24; length++ {
		secret := ""
		for i := 0; i < length; i++ {
			secret += string(alphabet[i%32])
		}
		err := ValidateSecret(secret)
		decodeInvalid := length%8 == 1 || length%8 == 3 || length%8 == 6
		if decodeInvalid && err == nil {
			t.Errorf("len=%d: 无法解码的密钥应被拒绝", length)
		}
		if !decodeInvalid && err != nil {
			t.Errorf("len=%d: 合法密钥被误拒: %v", length, err)
		}
	}
}

func TestNormalizeParams(t *testing.T) {
	// 零值补标准默认。
	acc := MFAAccount{Secret: "JBSWY3DPEHPK3PXP"}
	if err := acc.NormalizeParams(); err != nil {
		t.Fatalf("default params: %v", err)
	}
	if acc.Digits != DefaultDigits || acc.Period != DefaultPeriod || acc.Algorithm != DefaultAlgorithm {
		t.Fatalf("defaults not applied: %+v", acc)
	}

	// 小写算法名归一为大写。
	custom := MFAAccount{Secret: "JBSWY3DPEHPK3PXP", Digits: 8, Period: 60, Algorithm: " sha256 "}
	if err := custom.NormalizeParams(); err != nil {
		t.Fatalf("custom params: %v", err)
	}
	if custom.Digits != 8 || custom.Period != 60 || custom.Algorithm != "SHA256" {
		t.Fatalf("custom params not normalized: %+v", custom)
	}

	// 非法取值必须报错，且不改动账号。
	cases := []struct {
		name string
		acc  MFAAccount
	}{
		{name: "位数不支持", acc: MFAAccount{Digits: 7}},
		{name: "周期过小", acc: MFAAccount{Period: -1}},
		{name: "周期过大", acc: MFAAccount{Period: 3600}},
		{name: "算法不支持", acc: MFAAccount{Algorithm: "MD5"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			before := c.acc
			if err := c.acc.NormalizeParams(); err == nil {
				t.Fatal("expected error")
			}
			if c.acc != before {
				t.Fatalf("校验失败时不应改动账号: %+v", c.acc)
			}
		})
	}
}

func TestParamsSummary(t *testing.T) {
	cases := []struct {
		name string
		acc  MFAAccount
		want string
	}{
		{name: "标准参数无摘要", acc: MFAAccount{}, want: ""},
		{name: "八位", acc: MFAAccount{Digits: 8}, want: "8位"},
		{name: "六十秒", acc: MFAAccount{Period: 60}, want: "60s"},
		{name: "算法", acc: MFAAccount{Algorithm: "SHA256"}, want: "SHA256"},
		{name: "组合", acc: MFAAccount{Digits: 8, Period: 60, Algorithm: "SHA512"}, want: "SHA512 / 60s / 8位"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.acc.ParamsSummary(); got != c.want {
				t.Fatalf("ParamsSummary() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestEffectiveParams(t *testing.T) {
	var zero MFAAccount
	if zero.EffectiveDigits() != DefaultDigits || zero.EffectivePeriod() != DefaultPeriod || zero.EffectiveAlgorithm() != DefaultAlgorithm {
		t.Fatalf("zero value should fall back to defaults: %+v", zero)
	}
	if !zero.IsDefaultParams() {
		t.Fatal("zero value should count as default params")
	}

	custom := MFAAccount{Digits: 8, Period: 60, Algorithm: "sha512"}
	if custom.IsDefaultParams() {
		t.Fatal("custom params should not count as default")
	}
	if custom.EffectiveAlgorithm() != "SHA512" {
		t.Fatalf("EffectiveAlgorithm() = %q", custom.EffectiveAlgorithm())
	}
}
