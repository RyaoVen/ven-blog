// 图片接口：上传（multipart，仅 author）与公开直出。
// multipart 表单走不通 ApiCtx.Bind（仅 JSON），与认证接口一样挂 server 原生 fiber。
package interfaces

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"strconv"
	"sync"

	"golang.org/x/image/draw"

	"github.com/gofiber/fiber/v2"

	imgdomain "ven_hybird/build/domain/image"
	"ven_hybird/hybrid"
)

// RegisterImages 注册图片接口：POST /api/upload（multipart 上传，仅 author）、
// GET /images/:id（公开直出，长缓存）。路由挂 server 原生 fiber，不经 /api 自动前缀与角色守卫，
// 鉴权在 handler 内用会话身份（CurrentUser）自行判定。
func RegisterImages(a *hybrid.App, images imgdomain.Repository) {
	server := a.Server()

	server.App().Post("/api/upload", func(ctx *fiber.Ctx) error {
		userID, role, ok := server.CurrentUser(ctx)
		if !ok {
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthenticated"})
		}
		if role != "author" {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		}
		uploaderID, err := strconv.ParseInt(userID, 10, 64)
		if err != nil {
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthenticated"})
		}
		file, err := ctx.FormFile("file")
		if err != nil {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file is required"})
		}
		if file.Size > imgdomain.MaxSize {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file too large (max 5MB)"})
		}
		mime := file.Header.Get("Content-Type")
		if !imgdomain.AllowedMime(mime) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "unsupported image type"})
		}
		src, err := file.Open()
		if err != nil {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "bad file"})
		}
		defer func() { _ = src.Close() }()
		data, err := io.ReadAll(src)
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
		img := &imgdomain.Image{
			UploaderID: uploaderID,
			Filename:   file.Filename,
			Mime:       mime,
			Data:       data,
		}
		if err := images.Create(img); err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
		id := strconv.FormatInt(img.ID, 10)
		return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id, "url": "/images/" + id})
	})

	server.App().Get("/images/:id", func(ctx *fiber.Ctx) error {
		img, err := images.Get(mustID(ctx.Params("id")))
		if errors.Is(err, imgdomain.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "image not found"})
		}
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
		data, mime := img.Data, img.Mime
		// 宽度变体（测评报告 #11：原图直出无响应式）：?w=N 输出缩放重压缩版本，
		// 进程内缓存按 (id,w)；无 w 参数保持原图直出（immutable 缓存不变）。
		if wRaw := ctx.Query("w"); wRaw != "" {
			w, convErr := strconv.Atoi(wRaw)
			if convErr != nil || w < 16 || w > 2000 {
				return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid w (16-2000)"})
			}
			var scaled []byte
			var ok bool
			scaled, ok = cachedVariant(img.ID, w)
			if !ok {
				scaled, ok = scaleImage(data, mime, w)
				if ok {
					cacheVariant(img.ID, w, scaled)
				}
			}
			if ok {
				data, mime = scaled, "image/jpeg"
			}
		}
		ctx.Set(fiber.HeaderContentType, mime)
		ctx.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
		return ctx.Send(data)
	})
}

/* ===== 宽度变体（缩放 + 重压缩；Go 标准库仅支持 JPEG/PNG 编码，WebP 编码留待框架需求） ===== */

const (
	variantMaxWidth    = 2000
	variantJPEGQuality = 85
	variantCacheLimit  = 256
)

var (
	variantMu    sync.Mutex
	variantCache = map[string][]byte{}
	variantOrder []string // 简易 FIFO 淘汰
)

// cachedVariant 取进程内变体缓存。
func cachedVariant(id int64, w int) ([]byte, bool) {
	variantMu.Lock()
	defer variantMu.Unlock()
	data, ok := variantCache[variantKey(id, w)]
	return data, ok
}

// cacheVariant 写缓存（FIFO 超限淘汰）。
func cacheVariant(id int64, w int, data []byte) {
	variantMu.Lock()
	defer variantMu.Unlock()
	key := variantKey(id, w)
	if _, exists := variantCache[key]; !exists {
		variantOrder = append(variantOrder, key)
		if len(variantOrder) > variantCacheLimit {
			delete(variantCache, variantOrder[0])
			variantOrder = variantOrder[1:]
		}
	}
	variantCache[key] = data
}

func variantKey(id int64, w int) string {
	return strconv.FormatInt(id, 10) + ":" + strconv.Itoa(w)
}

// scaleImage 解码 → 等比缩放到目标宽（只缩不放）→ JPEG 重压缩。
// 解码失败（不支持的格式/损坏）返回 ok=false，调用方回退原图。
func scaleImage(data []byte, mime string, targetW int) ([]byte, bool) {
	var src image.Image
	var decodeErr error
	switch mime {
	case "image/jpeg":
		src, decodeErr = jpeg.Decode(bytes.NewReader(data))
	case "image/png":
		src, decodeErr = png.Decode(bytes.NewReader(data))
	default:
		return nil, false
	}
	if decodeErr != nil || src == nil {
		return nil, false
	}
	bounds := src.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()
	if srcW <= targetW {
		return nil, false // 原图已小于目标宽，缩放无收益
	}
	targetH := srcH * targetW / srcW
	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)
	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: variantJPEGQuality}); err != nil {
		return nil, false
	}
	return out.Bytes(), true
}
