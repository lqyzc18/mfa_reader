package qrscan

import (
	"bytes"
	"image"
	"image/png"
	"strings"
	"testing"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
)

// encodeQR 生成一张二维码 PNG，用于验证解码链路。
func encodeQR(t *testing.T, content string, size int) []byte {
	t.Helper()
	code, err := qr.Encode(content, qr.M, qr.Auto)
	if err != nil {
		t.Fatalf("encode qr: %v", err)
	}
	scaled, err := barcode.Scale(code, size, size)
	if err != nil {
		t.Fatalf("scale qr: %v", err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, scaled); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestFromReaderDecodesQR(t *testing.T) {
	const content = "otpauth://totp/Test:user?secret=JBSWY3DPEHPK3PXP&issuer=Test"
	if got, err := FromReader(bytes.NewReader(encodeQR(t, content, 256))); err != nil || got != content {
		t.Fatalf("FromReader = %q, %v; want %q", got, err, content)
	}
}

// 小尺寸二维码（每个模块仅几像素）同样要能识别。
func TestFromImageDecodesSmallQR(t *testing.T) {
	const content = "JBSWY3DPEHPK3PXP"
	img, _, err := image.Decode(bytes.NewReader(encodeQR(t, content, 64)))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := FromImage(img); err != nil || got != content {
		t.Fatalf("FromImage = %q, %v; want %q", got, err, content)
	}
}

func TestFromReaderRejectsNonImage(t *testing.T) {
	if _, err := FromReader(strings.NewReader("not an image")); err == nil {
		t.Fatal("非图片内容应返回错误")
	}
}

func TestUpscaleBounds(t *testing.T) {
	if got := upscale(image.NewRGBA(image.Rect(0, 0, 4000, 4000))); got != nil {
		t.Fatal("放大后规模过大时应跳过")
	}
	got := upscale(image.NewRGBA(image.Rect(0, 0, 100, 80)))
	if got == nil || got.Bounds().Dx() != 200 || got.Bounds().Dy() != 160 {
		t.Fatalf("小图应放大一倍，实际 %v", got.Bounds())
	}
}
