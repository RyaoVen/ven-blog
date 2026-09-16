// SEO 基础设施：favicon、robots.txt、sitemap.xml、manifest.json（测评报告 #9/#1 静态资源层；
// head 注入依赖框架需求 8，另行推进）。站点基址来自 BLOG_SITE_URL（与 RSS 同源）。
// 路由用原生 fiber 注册（auth.go/mcp.go 先例）：二进制与纯文本响应不经 ApiCtx JSON 截流。
package interfaces

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"ven_hybird/build/application/postapp"
	"ven_hybird/hybrid"
)

// RegisterSEO 注册 SEO 基础资源（公开；站点基址启动期解析，与 RSS 同策略）。
// sitemap v1 覆盖：静态页 + published posts；docs 具体文档页由插件后续贡献。
func RegisterSEO(a *hybrid.App, posts *postapp.Service, siteURL string) error {
	app := a.Server().App()

	// /favicon.ico：程序生成 32x32 品牌色 PNG 内嵌 ICO 容器（一次性生成，进程内缓存）。
	var (
		icoOnce sync.Once
		icoData []byte
	)
	app.Get("/favicon.ico", func(ctx *fiber.Ctx) error {
		icoOnce.Do(func() { icoData = buildFaviconICO() })
		ctx.Set("Content-Type", "image/x-icon")
		ctx.Set("Cache-Control", "public, max-age=604800")
		return ctx.Send(icoData)
	})

	// /robots.txt：全站允许（后台与接口不暴露）+ sitemap 指向。
	app.Get("/robots.txt", func(ctx *fiber.Ctx) error {
		ctx.Set("Content-Type", "text/plain; charset=utf-8")
		ctx.Set("Cache-Control", "public, max-age=86400")
		return ctx.SendString("User-agent: *\nAllow: /\nDisallow: /admin\nDisallow: /api\n\nSitemap: " + siteURL + "/sitemap.xml\n")
	})

	// /sitemap.xml：静态页 + published posts（lastmod = updated_at）。
	app.Get("/sitemap.xml", func(ctx *fiber.Ctx) error {
		recent, err := posts.ListRecent(0)
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
		var b bytes.Buffer
		b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
		b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
		writeURL := func(path string, lastmod time.Time) {
			mod := ""
			if !lastmod.IsZero() {
				mod = "<lastmod>" + lastmod.UTC().Format("2006-01-02") + "</lastmod>"
			}
			b.WriteString("<url><loc>" + siteURL + path + "</loc>" + mod + "</url>\n")
		}
		writeURL("/", time.Time{})
		writeURL("/posts", time.Time{})
		writeURL("/moments", time.Time{})
		writeURL("/docs", time.Time{})
		for _, p := range recent {
			writeURL("/posts/"+strconv.FormatInt(p.ID, 10), p.UpdatedAt)
		}
		b.WriteString("</urlset>")
		ctx.Set("Content-Type", "application/xml; charset=utf-8")
		ctx.Set("Cache-Control", "public, max-age=3600")
		return ctx.Send(b.Bytes())
	})

	// /manifest.json：PWA 基础清单（name/icons/theme）。
	app.Get("/manifest.json", func(ctx *fiber.Ctx) error {
		ctx.Set("Content-Type", "application/manifest+json")
		ctx.Set("Cache-Control", "public, max-age=86400")
		return ctx.JSON(fiber.Map{
			"name":             "ven-blog",
			"short_name":       "ven-blog",
			"start_url":        "/",
			"display":          "standalone",
			"background_color": "#fafaf9",
			"theme_color":      "#0d9488",
			"icons": []fiber.Map{
				{"src": "/favicon.ico", "sizes": "32x32", "type": "image/x-icon"},
			},
		})
	})
	return nil
}

// buildFaviconICO 生成 32x32 品牌色 PNG 内嵌 ICO 容器（teal #0d9488，与站点强调色一致）。
func buildFaviconICO() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	teal := color.RGBA{R: 0x0d, G: 0x94, B: 0x88, A: 0xff}
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			// 圆角遮罩：四角 6px 内透明。
			corner := 6
			inCorner := (x < corner && y < corner) || (x >= 32-corner && y < corner) ||
				(x < corner && y >= 32-corner) || (x >= 32-corner && y >= 32-corner)
			if !inCorner {
				img.Set(x, y, teal)
			}
		}
	}
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		return nil
	}
	pngData := pngBuf.Bytes()
	// ICO 容器：ICONDIR(6B) + ICONDIRENTRY(16B) + PNG 数据（Vista+ 支持 PNG-in-ICO）。
	ico := make([]byte, 22+len(pngData))
	ico[2], ico[3] = 1, 0   // type = icon
	ico[4], ico[5] = 1, 0   // count = 1
	ico[6] = 32             // width
	ico[7] = 32             // height
	ico[10], ico[11] = 1, 0 // planes
	size := uint32(len(pngData))
	ico[14] = byte(size)
	ico[15] = byte(size >> 8)
	ico[16] = byte(size >> 16)
	ico[17] = byte(size >> 24)
	ico[18] = 22 // data offset
	copy(ico[22:], pngData)
	return ico
}
