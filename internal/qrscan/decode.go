package qrscan

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/kbinani/screenshot"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	_ "golang.org/x/image/bmp"
)

// FromFile 从图片文件解码二维码文本。
func FromFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return "", fmt.Errorf("无法解码图片: %w", err)
	}
	return FromImage(img)
}

// FromImage 从内存图像解码二维码。
func FromImage(img image.Image) (string, error) {
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return "", err
	}
	hints := map[gozxing.DecodeHintType]interface{}{
		gozxing.DecodeHintType_TRY_HARDER: true,
	}
	result, err := qrcode.NewQRCodeReader().Decode(bmp, hints)
	if err != nil {
		return "", fmt.Errorf("未识别到二维码")
	}
	return result.GetText(), nil
}

// FromDisplays 截取全部显示器并尝试解码二维码。
func FromDisplays() (string, error) {
	n := screenshot.NumActiveDisplays()
	if n <= 0 {
		return "", fmt.Errorf("没有可用的显示器")
	}
	var last error
	for i := 0; i < n; i++ {
		bounds := screenshot.GetDisplayBounds(i)
		img, err := screenshot.CaptureRect(bounds)
		if err != nil {
			last = err
			continue
		}
		text, err := FromImage(img)
		if err == nil {
			return text, nil
		}
		last = err
	}
	if last == nil {
		last = fmt.Errorf("未识别到二维码")
	}
	return "", last
}
