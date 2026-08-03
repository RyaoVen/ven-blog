// 订阅通知：新文章发布 → 异步邮件通知全部订阅者。
// 接口层协调（对齐 moderator 先例）：subscribeapp 取订阅者列表、窄 Mailer 接口注入发信，
// HTML 复用 application/emailhtml.RenderNewArticle，不依赖基础设施实现（组装根注入）。
package interfaces

import (
	"log"
	"strconv"

	"ven_hybird/build/application/emailhtml"
	"ven_hybird/build/domain/post"
)

// Mailer 邮件发送窄接口（组装根注入 infrastructure/mailer.SMTPMailer；对齐 moderator.Mailer 先例）。
type Mailer interface {
	SendHTML(to, subject, html string) error
}

// NewPostNotifier 新文章发布通知回调（组装根注入 RegisterAPIs，创建成功后触发）。
type NewPostNotifier func(p *post.Post)

// notifySubscribers 新文章发布异步通知全部订阅者：goroutine 内拉订阅者 → 无则不发 →
// 逐条 SendHTML（RenderNewArticle 模板：标题+摘要+siteURL+/posts/:id 链接）。
// 调用立即返回，不阻塞发布响应；goroutine 内 defer recover 兜底（单条 panic 不崩进程），
// 单条失败 log.Printf 后继续下一条。
func notifySubscribers(getSubscribers func() ([]string, error), mailer Mailer, siteURL string, p *post.Post) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("postnotify: notify panic: %v", r)
			}
		}()
		runNewArticleNotify(getSubscribers, mailer, siteURL, p)
	}()
}

// runNewArticleNotify goroutine 主体（同步核心，可单测）：
// 拉订阅者失败记日志不发；无订阅者不发；否则逐条发送，单条失败记日志继续。
func runNewArticleNotify(getSubscribers func() ([]string, error), mailer Mailer, siteURL string, p *post.Post) {
	emails, err := getSubscribers()
	if err != nil {
		log.Printf("postnotify: list subscribers failed: %v", err)
		return
	}
	if len(emails) == 0 {
		return
	}
	subject := "ven-blog 新文章：" + p.Title
	html := emailhtml.RenderNewArticle("ven-blog", siteURL, p.Title, p.Summary, "/posts/"+strconv.FormatInt(p.ID, 10))
	for _, email := range emails {
		if err := mailer.SendHTML(email, subject, html); err != nil {
			log.Printf("postnotify: send to %s failed: %v", email, err)
		}
	}
}
