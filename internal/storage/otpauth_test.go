package storage

import (
	"strings"
	"testing"

	"mfa_reader/internal/model"
)

func TestParseOTPAuth(t *testing.T) {
	acc, err := ParseOTPAuth("otpauth://totp/GitHub:octocat?secret=JBSWY3DPEHPK3PXP&issuer=GitHub")
	if err != nil {
		t.Fatal(err)
	}
	if acc.Secret != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("secret=%s", acc.Secret)
	}
	if !strings.Contains(acc.AccountName, "octocat") {
		t.Fatalf("name=%s", acc.AccountName)
	}

	if _, err := ParseOTPAuth("otpauth://hotp/x?secret=JBSWY3DPEHPK3PXP"); err == nil {
		t.Fatal("hotp should be rejected")
	}
}

func TestParseImportTextJSONAndLines(t *testing.T) {
	accs, err := ParseImportText(`[{"accountName":"A","secret":"JBSWY3DPEHPK3PXP"}]`)
	if err != nil || len(accs) != 1 || accs[0].AccountName != "A" {
		t.Fatalf("%v %v", accs, err)
	}

	accs, err = ParseImportText("otpauth://totp/B?secret=234567ABCDEF2345\notpauth://totp/C?secret=JBSWY3DPEHPK3PXP")
	if err != nil || len(accs) != 2 {
		t.Fatalf("%v %v", accs, err)
	}
}

func TestParseQRTextSecret(t *testing.T) {
	accs, err := ParseQRText("jbsw y3dp ehpk 3pxp")
	if err != nil || len(accs) != 1 {
		t.Fatalf("%v %v", accs, err)
	}
	if accs[0].Secret != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("secret=%s", accs[0].Secret)
	}
}

func TestFormatOTPAuthRoundTrip(t *testing.T) {
	src := model.MFAAccount{AccountName: "Example", Secret: "JBSWY3DPEHPK3PXP"}
	uri := FormatOTPAuth(src)
	got, err := ParseOTPAuth(uri)
	if err != nil {
		t.Fatal(err)
	}
	if got.Secret != src.Secret {
		t.Fatalf("got %+v", got)
	}
}

// 标准参数不写出冗余字段，导入后仍回落为默认值。
func TestFormatOTPAuthOmitsDefaultParams(t *testing.T) {
	uri := FormatOTPAuth(model.MFAAccount{AccountName: "Example", Secret: "JBSWY3DPEHPK3PXP"})
	for _, param := range []string{"digits=", "period=", "algorithm="} {
		if strings.Contains(uri, param) {
			t.Fatalf("uri should omit default params: %s", uri)
		}
	}
	got, err := ParseOTPAuth(uri)
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsDefaultParams() {
		t.Fatalf("params should fall back to defaults: %+v", got)
	}
}

// otpauth 链接中的位数、周期与算法必须被保留并在导出时回写。
func TestParseOTPAuthKeepsCustomParams(t *testing.T) {
	uri := "otpauth://totp/Test:user?secret=JBSWY3DPEHPK3PXP&issuer=Test&digits=8&period=60&algorithm=SHA256"
	acc, err := ParseOTPAuth(uri)
	if err != nil {
		t.Fatal(err)
	}
	if acc.Digits != 8 || acc.Period != 60 || acc.Algorithm != "SHA256" {
		t.Fatalf("custom params lost: %+v", acc)
	}

	out := FormatOTPAuth(acc)
	for _, param := range []string{"digits=8", "period=60", "algorithm=sha256"} {
		if !strings.Contains(out, param) {
			t.Fatalf("uri should keep %s: %s", param, out)
		}
	}
	round, err := ParseOTPAuth(out)
	if err != nil {
		t.Fatal(err)
	}
	if round.Digits != acc.Digits || round.Period != acc.Period || round.Algorithm != acc.Algorithm {
		t.Fatalf("round trip changed params: %+v -> %+v", acc, round)
	}
}

// 库只支持 6/8 位，非法取值必须在导入时明确报错，而不是生成错误验证码。
func TestParseOTPAuthRejectsUnsupportedDigits(t *testing.T) {
	uri := "otpauth://totp/Test:user?secret=JBSWY3DPEHPK3PXP&digits=7"
	if _, err := ParseOTPAuth(uri); err == nil {
		t.Fatal("digits=7 should be rejected")
	}
}

// 无法解码的密钥长度应在导入阶段被拦截。
func TestParseImportTextRejectsUndecodableSecret(t *testing.T) {
	if _, err := ParseImportText(`[{"accountName":"A","secret":"JBSWY3DPEHPK3PXPX"}]`); err == nil {
		t.Fatal("undecodable base32 length should be rejected")
	}
}
