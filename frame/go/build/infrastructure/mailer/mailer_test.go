package mailer

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

// unconfiguredMailer 构造 SMTP 未配置（降级日志路径）的发送器。
func unconfiguredMailer() *SMTPMailer {
	return NewSMTPMailer(func() (Config, error) {
		return Config{}, nil
	})
}

// captureLog 捕获包级 log 输出（测试串行，包内无并行测试）。
func captureLog(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	old := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(old)
	fn()
	return buf.String()
}

// TestSend_DegradedLog_RedactsBody 降级日志默认不打印正文（验证码等敏感内容不进日志）。
func TestSend_DegradedLog_RedactsBody(t *testing.T) {
	t.Setenv("BLOG_MAIL_DEBUG_BODY", "")
	secret := "验证码 482913，5 分钟内有效"
	out := captureLog(t, func() {
		if err := unconfiguredMailer().Send("reader@example.com", "验证码", secret); err != nil {
			t.Fatalf("degraded send should succeed: %v", err)
		}
	})
	if strings.Contains(out, secret) {
		t.Fatalf("degraded log leaked body content: %s", out)
	}
	for _, want := range []string{"reader@example.com", "验证码", "text/plain", "body"} {
		if !strings.Contains(out, want) {
			t.Errorf("degraded log missing %q: %s", want, out)
		}
	}
}

// TestSendHTML_DegradedLog_RedactsHTMLBody 同上，HTML 邮件（审核摘要）路径。
func TestSendHTML_DegradedLog_RedactsHTMLBody(t *testing.T) {
	t.Setenv("BLOG_MAIL_DEBUG_BODY", "")
	secret := `<p>被驳回的评论内容：敏感言论</p>`
	out := captureLog(t, func() {
		if err := unconfiguredMailer().SendHTML("author@example.com", "审核摘要", secret); err != nil {
			t.Fatalf("degraded send should succeed: %v", err)
		}
	})
	if strings.Contains(out, secret) {
		t.Fatalf("degraded log leaked html body: %s", out)
	}
	if !strings.Contains(out, "text/html") {
		t.Errorf("degraded log missing content type: %s", out)
	}
}

// TestSend_DegradedLog_DebugBody 开启 BLOG_MAIL_DEBUG_BODY=1 时打印全文（开发/联调用）。
func TestSend_DegradedLog_DebugBody(t *testing.T) {
	t.Setenv("BLOG_MAIL_DEBUG_BODY", "1")
	body := "验证码 482913"
	out := captureLog(t, func() {
		if err := unconfiguredMailer().Send("reader@example.com", "验证码", body); err != nil {
			t.Fatalf("degraded send should succeed: %v", err)
		}
	})
	if !strings.Contains(out, body) {
		t.Errorf("debug body flag should print body, got: %s", out)
	}
}
