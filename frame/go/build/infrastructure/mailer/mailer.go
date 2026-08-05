// Package mailer 基础设施层：SMTP 邮件发送（465 隐式 TLS 与 587/25 STARTTLS 均支持）。
// 未配置 SMTP 时降级为日志输出（开发/联调可经网关日志拿到验证码走完流程）。
package mailer

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"
)

// Config SMTP 配置。
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	FromName string
}

// Configured 是否已配置（host 与 user 非空即可发信）。
func (c Config) Configured() bool {
	return c.Host != "" && c.User != ""
}

// Mailer 发送接口（业务只依赖这两个方法）。
type Mailer interface {
	Send(to, subject, text string) error
	SendHTML(to, subject, html string) error
}

// SMTPMailer 每次发送时经 resolver 现取配置（设置页改动即时生效）。
type SMTPMailer struct {
	resolver func() (Config, error)
}

// NewSMTPMailer 构造邮件发送器；resolver 由组装根注入（读 settings）。
func NewSMTPMailer(resolver func() (Config, error)) *SMTPMailer {
	return &SMTPMailer{resolver: resolver}
}

// Send 发送纯文本邮件；未配置 SMTP 时降级日志输出并返回 nil。
func (m *SMTPMailer) Send(to, subject, text string) error {
	return m.sendAny(to, subject, text, "text/plain")
}

// SendHTML 发送 HTML 邮件（text/html content-type）；未配置 SMTP 时降级日志输出并返回 nil。
func (m *SMTPMailer) SendHTML(to, subject, html string) error {
	return m.sendAny(to, subject, html, "text/html")
}

// sendAny 解析配置并发送；未配置 SMTP 时降级日志输出并返回 nil。
func (m *SMTPMailer) sendAny(to, subject, body, contentType string) error {
	cfg, err := m.resolver()
	if err != nil {
		return fmt.Errorf("mail: resolve config: %w", err)
	}
	if !cfg.Configured() {
		// 降级默认只打摘要：正文含验证码等敏感内容，不写日志；
		// 开发/联调需要看正文时显式开启 BLOG_MAIL_DEBUG_BODY=1。
		if os.Getenv("BLOG_MAIL_DEBUG_BODY") == "" {
			log.Printf("mail: smtp not configured, would send %s to %s: [%s] (body %d bytes, set BLOG_MAIL_DEBUG_BODY=1 to print)", contentType, to, subject, len(body))
		} else {
			log.Printf("mail: smtp not configured, would send %s to %s: [%s]\n%s", contentType, to, subject, body)
		}
		return nil
	}
	return m.send(cfg, to, subject, body, contentType)
}

// smtpSessionTimeout 单次 SMTP 会话整体超时（dial 后设置连接 deadline）：
// DATA/QUIT 等阶段 TCP 无内建超时，邮件服务器挂起会让发信 goroutine 永久占坑。
const smtpSessionTimeout = 30 * time.Second

// send 经 SMTP 发送（465 隐式 TLS；其余端口尝试 STARTTLS，不支持则明文）。
func (m *SMTPMailer) send(cfg Config, to, subject, body, contentType string) error {
	host := cfg.Host
	port := cfg.Port
	if port == "" {
		port = "465"
	}
	addr := net.JoinHostPort(host, port)

	var client *smtp.Client
	if port == "465" {
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, &tls.Config{ServerName: host})
		if err != nil {
			return fmt.Errorf("mail: tls dial %s: %w", addr, err)
		}
		_ = conn.SetDeadline(time.Now().Add(smtpSessionTimeout))
		client, err = smtp.NewClient(conn, host)
		if err != nil {
			return fmt.Errorf("mail: smtp client: %w", err)
		}
	} else {
		conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
		if err != nil {
			return fmt.Errorf("mail: dial %s: %w", addr, err)
		}
		_ = conn.SetDeadline(time.Now().Add(smtpSessionTimeout))
		client, err = smtp.NewClient(conn, host)
		if err != nil {
			return fmt.Errorf("mail: smtp client: %w", err)
		}
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: host}); err != nil {
				return fmt.Errorf("mail: starttls: %w", err)
			}
		}
	}
	defer func() { _ = client.Close() }()

	if cfg.Password != "" {
		if err := client.Auth(smtp.PlainAuth("", cfg.User, cfg.Password, host)); err != nil {
			return fmt.Errorf("mail: auth: %w", err)
		}
	}
	from := cfg.User
	if cfg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", encodeWord(cfg.FromName), cfg.User)
	}
	if err := client.Mail(cfg.User); err != nil {
		return fmt.Errorf("mail: MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("mail: RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("mail: DATA: %w", err)
	}
	msg := strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: " + encodeWord(subject),
		"MIME-Version: 1.0",
		"Content-Type: " + contentType + "; charset=utf-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		body,
	}, "\r\n")
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("mail: write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mail: close data: %w", err)
	}
	return client.Quit()
}

// encodeWord 编码 RFC2047 非 ASCII 头部（中文主题/发件名）。
func encodeWord(s string) string {
	ascii := true
	for _, r := range s {
		if r > 127 {
			ascii = false
			break
		}
	}
	if ascii {
		return s
	}
	return "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(s)) + "?="
}
