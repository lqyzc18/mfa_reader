package qrscan

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"

	"github.com/kbinani/screenshot"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
)

// 放大重试的规模上限（放大后的像素数）：超过则跳过，避免内存激增。
const upscaleMaxPixels = 16 << 20

// FromFile 从图片文件解码二维码文本。
func FromFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return FromReader(f)
}

// FromReader 从图片流解码二维码文本。
func FromReader(r io.Reader) (string, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return "", fmt.Errorf("无法解码图片: %w", err)
	}
	return FromImage(img)
}

// FromImage 从内存图像解码二维码。
// 首次识别失败时放大一倍再试一次：截图与网页里的二维码常常偏小，放大后识别率明显提升。
func FromImage(img image.Image) (string, error) {
	if text, err := decode(img); err == nil {
		return text, nil
	}
	if enlarged := upscale(img); enlarged != nil {
		if text, err := decode(enlarged); err == nil {
			return text, nil
		}
	}
	return "", fmt.Errorf("未识别到二维码")
}

// decode 单次解码尝试，开启 TRY_HARDER 提升容错。
func decode(img image.Image) (string, error) {
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return "", err
	}
	hints := map[gozxing.DecodeHintType]interface{}{
		gozxing.DecodeHintType_TRY_HARDER: true,
	}
	result, err := qrcode.NewQRCodeReader().Decode(bmp, hints)
	if err != nil {
		return "", err
	}
	return result.GetText(), nil
}

// upscale 把图像放大一倍；放大后规模过大时返回 nil 表示跳过。
func upscale(src image.Image) image.Image {
	b := src.Bounds()
	if b.Dx()*b.Dy()*4 > upscaleMaxPixels {
		return nil
	}
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx()*2, b.Dy()*2))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return dst
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
