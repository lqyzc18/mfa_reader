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
