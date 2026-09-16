package interfaces

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

// TestScaleImage 缩放变体（issue #17：测评报告 #11 原图直出）。
func TestScaleImage(t *testing.T) {
	// 生成 800x500 测试 JPEG（纯色块 + 渐变）。
	src := image.NewRGBA(image.Rect(0, 0, 800, 500))
	for y := 0; y < 500; y++ {
		for x := 0; x < 800; x++ {
			src.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 0x7f, A: 0xff})
		}
	}
	var srcBuf bytes.Buffer
	if err := jpeg.Encode(&srcBuf, src, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode src: %v", err)
	}

	// 缩到 400 宽：尺寸正确、体积显著下降。
	out, ok := scaleImage(srcBuf.Bytes(), "image/jpeg", 400)
	if !ok {
		t.Fatal("scaleImage 应成功")
	}
	decoded, err := jpeg.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("变体应是合法 JPEG: %v", err)
	}
	if got := decoded.Bounds().Dx(); got != 400 {
		t.Fatalf("目标宽应 400，得 %d", got)
	}
	if len(out) >= srcBuf.Len() {
		t.Fatalf("变体应更小：out=%d src=%d", len(out), srcBuf.Len())
	}

	// 目标宽大于原图：不缩放（回退原图）。
	if _, ok := scaleImage(srcBuf.Bytes(), "image/jpeg", 1600); ok {
		t.Fatal("原图小于目标宽应返回 false")
	}

	// 不支持的格式。
	if _, ok := scaleImage(srcBuf.Bytes(), "image/webp", 400); ok {
		t.Fatal("webp 应不支持（编码器留待框架需求）")
	}
	// 损坏数据。
	if _, ok := scaleImage([]byte("garbage"), "image/jpeg", 400); ok {
		t.Fatal("损坏数据应返回 false")
	}
}

// TestVariantCache FIFO 淘汰与命中。
func TestVariantCache(t *testing.T) {
	for i := 0; i < variantCacheLimit+10; i++ {
		cacheVariant(int64(i), 400, []byte{byte(i)})
	}
	if _, ok := cachedVariant(0, 400); ok {
		t.Fatal("最早写入的条目应被淘汰")
	}
	if _, ok := cachedVariant(int64(variantCacheLimit+9), 400); !ok {
		t.Fatal("最新条目应可命中")
	}
}
