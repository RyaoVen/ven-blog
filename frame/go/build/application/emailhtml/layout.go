// Package emailhtml 邮件 HTML 模板渲染：工业极简风格（对齐 src/lib/globalCss.ts 浅色设计令牌），
// 纯函数渲染、全部内联样式（邮件客户端不支持外部 CSS/媒体查询）、html/template 自动转义。
// 站点信息（站点名/公网地址）由调用方注入——本包不依赖 settings，调用方各取所需传入。
package emailhtml

import (
	"html/template"
	"strings"
	"time"
)

// 设计令牌（对齐 globalCss.ts 浅色系）。
const (
	colorBG        = "#fafaf9" // --bg
	colorBGSubtle  = "#f5f4f2" // --bg-subtle
	colorBorder    = "#e7e5e4" // --border
	colorBorderStr = "#d6d3d1" // --border-strong
	colorText      = "#1c1917" // --text
	colorSecondary = "#78716c" // --text-secondary
	colorAccent    = "#0d9488" // --accent
	colorAccentFg  = "#fafaf9" // --primary-fg
	// 字体栈：等宽为主（标题/按钮/元信息）+ 无衬线正文（同 globalCss body 栈）。
	fontMono = "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace"
	fontSans = "system-ui, -apple-system, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif"
)

// layoutData 布局模板数据。
type layoutData struct {
	SiteName string
	SiteURL  string
	Title    string
	Body     template.HTML // 本包各渲染函数输出的可信 HTML
	Year     int
}

// RenderLayout 渲染邮件完整 HTML 文档：
// 头部（站点名 + 细下边框，链接站点公网地址）→ 标题 + 内容区 → 页脚（站点公网地址 + 版权）。
// bodyHTML 只接受本包各类型渲染函数的输出（视为可信，不再转义）；
// siteName/siteURL 由调用方注入（如站点名 "ven-blog"、siteURL 取调用方既有参数）。
func RenderLayout(siteName, siteURL, title, bodyHTML string) string {
	var b strings.Builder
	_ = layoutTmpl.Execute(&b, layoutData{
		SiteName: siteName,
		SiteURL:  siteURL,
		Title:    title,
		Body:     template.HTML(bodyHTML),
		Year:     time.Now().Year(),
	})
	return b.String()
}

// layoutTmpl 布局模板（table 布局 + 内联样式，兼容主流邮件客户端）。
// 模板在包初始化时静态解析，执行失败视为编程错误（数据字段与结构体一一对应）。
var layoutTmpl = template.Must(template.New("layout").Parse(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.Title}}</title>
</head>
<body style="margin:0;padding:0;background-color:` + colorBG + `;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:` + colorBG + `;">
<tr><td align="center" style="padding:32px 16px;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:600px;border:1px solid ` + colorBorder + `;border-radius:6px;background-color:` + colorBG + `;">
<tr>
<td style="padding:18px 24px;border-bottom:1px solid ` + colorBorder + `;">
<a href="{{.SiteURL}}" style="font-family:` + fontMono + `;font-size:13px;font-weight:600;letter-spacing:0.08em;color:` + colorText + `;text-decoration:none;">{{.SiteName}}</a>
</td>
</tr>
<tr>
<td style="padding:24px;color:` + colorText + `;font-family:` + fontSans + `;font-size:15px;line-height:1.7;">
<h1 style="margin:0 0 16px;font-size:18px;font-weight:650;line-height:1.3;color:` + colorText + `;">{{.Title}}</h1>
{{.Body}}
</td>
</tr>
<tr>
<td style="padding:14px 24px;border-top:1px solid ` + colorBorder + `;font-family:` + fontMono + `;font-size:12px;color:` + colorSecondary + `;line-height:1.6;">
<div>{{.SiteName}} · <a href="{{.SiteURL}}" style="color:` + colorSecondary + `;text-decoration:none;">{{.SiteURL}}</a></div>
<div style="margin-top:4px;">© {{.Year}} {{.SiteName}}</div>
</td>
</tr>
</table>
</td></tr>
</table>
</body>
</html>`))

// actionButtonStyle 行动按钮内联样式（强调色块 + 等宽字重）。
const actionButtonStyle = "display:inline-block;padding:9px 20px;border:1px solid " + colorAccent + ";border-radius:3px;" +
	"background-color:" + colorAccent + ";color:" + colorAccentFg + ";" +
	"font-family:" + fontMono + ";font-size:13px;font-weight:600;letter-spacing:0.05em;text-decoration:none;"

// joinURL 拼接站点公网地址与站内路径（兼容 siteURL 带尾斜杠）。
func joinURL(siteURL, path string) string {
	return strings.TrimRight(siteURL, "/") + path
}
