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

// Mailer 发送接口（业务只依赖这一个方法）。
type Mailer interface {
	Send(to, subject, text string) error
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
	cfg, err := m.resolver()
	if err != nil {
		return fmt.Errorf("mail: resolve config: %w", err)
	}
	if !cfg.Configured() {
		log.Printf("mail: smtp not configured, would send to %s: [%s]\n%s", to, subject, text)
		return nil
	}
	return m.send(cfg, to, subject, text)
}

// send 经 SMTP 发送（465 隐式 TLS；其余端口尝试 STARTTLS，不支持则明文）。
func (m *SMTPMailer) send(cfg Config, to, subject, text string) error {
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
		client, err = smtp.NewClient(conn, host)
		if err != nil {
			return fmt.Errorf("mail: smtp client: %w", err)
		}
	} else {
		c, err := smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("mail: dial %s: %w", addr, err)
		}
		client = c
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
		"Content-Type: text/plain; charset=utf-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		text,
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
