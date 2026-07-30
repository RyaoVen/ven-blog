// 图片接口：上传（multipart，仅 author）与公开直出。
// multipart 表单走不通 ApiCtx.Bind（仅 JSON），与认证接口一样挂 server 原生 fiber。
package interfaces

import (
	"errors"
	"io"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"ven_hybird/build/domain/image"
	"ven_hybird/hybrid"
)

// RegisterImages 注册图片接口：POST /api/upload（multipart 上传，仅 author）、
// GET /images/:id（公开直出，长缓存）。路由挂 server 原生 fiber，不经 /api 自动前缀与角色守卫，
// 鉴权在 handler 内用会话身份（CurrentUser）自行判定。
func RegisterImages(a *hybrid.App, images image.Repository) {
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
		if file.Size > image.MaxSize {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file too large (max 5MB)"})
		}
		mime := file.Header.Get("Content-Type")
		if !image.AllowedMime(mime) {
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
		img := &image.Image{
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
		if errors.Is(err, image.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "image not found"})
		}
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
		ctx.Set(fiber.HeaderContentType, img.Mime)
		ctx.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
		return ctx.Send(img.Data)
	})
}
