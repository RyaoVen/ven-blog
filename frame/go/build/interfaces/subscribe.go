// 订阅接口：邮箱订阅 API + RSS 输出。
package interfaces

import (
	"encoding/xml"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"ven_hybird/build/application/postapp"
	"ven_hybird/build/application/subscribeapp"
	"ven_hybird/build/domain/post"
	"ven_hybird/hybrid"
)

// subscribeInput 订阅请求体。
type subscribeInput struct {
	Email string `json:"email"`
}

// RegisterSubscribe 注册订阅 API（/api/subscribe）与 RSS 路由（/rss.xml）。
// siteURL 用于 RSS 链接拼接（生产走 BLOG_SITE_URL 环境变量）。
func RegisterSubscribe(a *hybrid.App, sub *subscribeapp.Service, posts *postapp.Service, siteURL string) error {
	if err := a.Post("/subscribe", nil, func(c *hybrid.ApiCtx) error {
		var in subscribeInput
		if err := c.Bind(&in); err != nil {
			return c.Error(400, "bad body")
		}
		already, err := sub.Subscribe(in.Email)
		var vErr *subscribeapp.ValidationError
		if errors.As(err, &vErr) {
			return c.Error(400, vErr.Message)
		}
		if err != nil {
			return c.Error(500, "internal error")
		}
		return c.JSON(200, map[string]any{"ok": true, "already": already})
	}); err != nil {
		return err
	}

	// RSS 2.0：最近 20 篇文章（摘要为 description）
	a.Server().App().Get("/rss.xml", func(ctx *fiber.Ctx) error {
		list, err := posts.ListRecent(20)
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).SendString("internal error")
		}
		ctx.Set(fiber.HeaderContentType, "application/rss+xml; charset=utf-8")
		return ctx.SendString(buildRSS(siteURL, list))
	})
	return nil
}

// rssFeed RSS 2.0 文档结构。
type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

// rssChannel 频道元信息与条目。
type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Language    string    `xml:"language"`
	Items       []rssItem `xml:"item"`
}

// rssItem 单篇文章条目。
type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        string `xml:"guid"`
	PubDate     string `xml:"pubDate"`
	Description string `xml:"description"`
}

// buildRSS 生成 RSS XML（时间按 RFC1123Z，description 用摘要/正文截断）。
func buildRSS(siteURL string, list []*post.Post) string {
	feed := rssFeed{
		Version: "2.0",
		Channel: rssChannel{
			Title:       "RyaoVen 的博客",
			Link:        siteURL,
			Description: "记录技术与生活",
			Language:    "zh-CN",
			Items:       make([]rssItem, 0, len(list)),
		},
	}
	for _, p := range list {
		desc := p.Summary
		if desc == "" {
			desc = truncateRunes(p.Content, 140)
		}
		link := siteURL + "/posts/" + strconv.FormatInt(p.ID, 10)
		feed.Channel.Items = append(feed.Channel.Items, rssItem{
			Title:       p.Title,
			Link:        link,
			GUID:        link,
			PubDate:     p.CreatedAt.Format(time.RFC1123Z),
			Description: desc,
		})
	}
	out, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return ""
	}
	return xml.Header + string(out)
}

// truncateRunes 按字符截断并补省略号。
func truncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}
