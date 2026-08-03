package interfaces

import (
	"strings"
	"sync"
	"testing"
	"time"

	"ven_hybird/build/domain/post"
)

// fakeMailer 内存收集发送调用（测试用）。
type fakeMailer struct {
	mu    sync.Mutex
	calls []mailCall
}

type mailCall struct {
	to      string
	subject string
	html    string
}

func (m *fakeMailer) SendHTML(to, subject, html string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, mailCall{to: to, subject: subject, html: html})
	return nil
}

func (m *fakeMailer) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func (m *fakeMailer) call(i int) mailCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls[i]
}

// failMailer 首次发送失败（模拟单条 SMTP 故障），之后正常。
type failMailer struct {
	mu    sync.Mutex
	calls []mailCall
	fail  int // 第几次发送返回错误（0-based）
}

func (m *failMailer) SendHTML(to, subject, html string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	i := len(m.calls)
	m.calls = append(m.calls, mailCall{to: to, subject: subject, html: html})
	if i == m.fail {
		return &sendErr{to: to}
	}
	return nil
}

func (m *failMailer) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

type sendErr struct{ to string }

func (e *sendErr) Error() string { return "smtp failed for " + e.to }

// blockingMailer 发送时阻塞直到 release 放行（测异步不阻塞调用方）。
type blockingMailer struct {
	mu      sync.Mutex
	calls   int
	release chan struct{}
	done    chan struct{}
}

func (m *blockingMailer) SendHTML(to, subject, html string) error {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	<-m.release
	close(m.done)
	return nil
}

// panicMailer 发送即 panic（测 goroutine recover 兜底）。
type panicMailer struct{}

func (panicMailer) SendHTML(to, subject, html string) error {
	panic("smtp boom")
}

func TestRunNewArticleNotify(t *testing.T) {
	p := &post.Post{ID: 7, Title: "用 Go 写一个博客", Summary: "从零搭建混合渲染博客"}

	t.Run("sends one html mail per subscriber", func(t *testing.T) {
		mail := &fakeMailer{}
		runNewArticleNotify(func() ([]string, error) {
			return []string{"a@example.com", "b@example.com"}, nil
		}, mail, "https://blog.example.com", p)
		if mail.count() != 2 {
			t.Fatalf("SendHTML calls = %d, want 2", mail.count())
		}
		for i, wantTo := range []string{"a@example.com", "b@example.com"} {
			c := mail.call(i)
			if c.to != wantTo {
				t.Fatalf("call %d to = %q, want %q", i, c.to, wantTo)
			}
			if c.subject != "ven-blog 新文章：用 Go 写一个博客" {
				t.Fatalf("call %d subject = %q", i, c.subject)
			}
			// HTML 含标题/摘要/链接（siteURL + /posts/:id）
			for _, want := range []string{
				"用 Go 写一个博客",
				"从零搭建混合渲染博客",
				`href="https://blog.example.com/posts/7"`,
			} {
				if !strings.Contains(c.html, want) {
					t.Fatalf("call %d html missing %q", i, want)
				}
			}
		}
	})

	t.Run("no subscribers sends nothing", func(t *testing.T) {
		mail := &fakeMailer{}
		runNewArticleNotify(func() ([]string, error) { return []string{}, nil }, mail, "https://blog.example.com", p)
		if mail.count() != 0 {
			t.Fatalf("SendHTML calls = %d, want 0", mail.count())
		}
	})

	t.Run("list failure logs and sends nothing", func(t *testing.T) {
		mail := &fakeMailer{}
		runNewArticleNotify(func() ([]string, error) { return nil, &sendErr{to: "list"} }, mail, "https://blog.example.com", p)
		if mail.count() != 0 {
			t.Fatalf("SendHTML calls = %d, want 0", mail.count())
		}
	})

	t.Run("single failure does not block remaining", func(t *testing.T) {
		mail := &failMailer{fail: 0}
		runNewArticleNotify(func() ([]string, error) {
			return []string{"a@example.com", "b@example.com", "c@example.com"}, nil
		}, mail, "https://blog.example.com", p)
		if mail.count() != 3 {
			t.Fatalf("SendHTML calls = %d, want 3 (all attempted)", mail.count())
		}
		if got := mail.calls[1].to; got != "b@example.com" {
			t.Fatalf("call 1 to = %q, want b@example.com (continue after failure)", got)
		}
		if got := mail.calls[2].to; got != "c@example.com" {
			t.Fatalf("call 2 to = %q, want c@example.com (continue after failure)", got)
		}
	})

	t.Run("html escapes title and summary", func(t *testing.T) {
		mail := &fakeMailer{}
		runNewArticleNotify(func() ([]string, error) { return []string{"a@example.com"}, nil }, mail,
			"https://blog.example.com", &post.Post{ID: 1, Title: `<b>标题</b>`, Summary: `"引号"`})
		html := mail.call(0).html
		if strings.Contains(html, "<b>标题</b>") || !strings.Contains(html, "&lt;b&gt;标题&lt;/b&gt;") {
			t.Fatalf("title should be html-escaped:\n%s", html)
		}
	})
}

func TestNewPostNotifier(t *testing.T) {
	mail := &fakeMailer{}
	siteURL := "https://a.example"
	notify := NewPostNotifier(func() ([]string, error) { return []string{"a@example.com"}, nil }, mail, func() string { return siteURL })
	notify(&post.Post{ID: 3, Title: "T", Summary: "S"})
	time.Sleep(300 * time.Millisecond)
	if mail.count() != 1 {
		t.Fatalf("SendHTML calls = %d, want 1", mail.count())
	}
	if !strings.Contains(mail.call(0).html, `href="https://a.example/posts/3"`) {
		t.Fatalf("html should use resolved siteURL:\n%s", mail.call(0).html)
	}
	// siteURL 惰性解析：解析函数返回值变更后，下一次通知用新值
	siteURL = "https://b.example"
	notify(&post.Post{ID: 4, Title: "T", Summary: "S"})
	time.Sleep(300 * time.Millisecond)
	if mail.count() != 2 || !strings.Contains(mail.call(1).html, `href="https://b.example/posts/4"`) {
		t.Fatalf("html should re-resolve siteURL per notify:\n%s", mail.call(1).html)
	}
}

func TestNotifySubscribersAsync(t *testing.T) {
	mail := &blockingMailer{release: make(chan struct{}), done: make(chan struct{})}
	notifySubscribers(func() ([]string, error) { return []string{"a@example.com"}, nil },
		mail, "https://blog.example.com", &post.Post{ID: 1, Title: "T", Summary: "S"})
	// 异步不阻塞：调用已返回；若为同步实现会卡在 SendHTML（等待 release）导致本测试挂起超时。
	// 给 goroutine 时间进入发信，确认其未完成（仍阻塞在 release 上）。
	select {
	case <-mail.done:
		t.Fatal("notifySubscribers must not wait for send completion")
	case <-time.After(300 * time.Millisecond):
	}
	mail.mu.Lock()
	calls := mail.calls
	mail.mu.Unlock()
	if calls != 1 {
		t.Fatalf("mailer calls = %d while blocked, want 1 (goroutine should be running)", calls)
	}
	close(mail.release)
	select {
	case <-mail.done:
	case <-time.After(2 * time.Second):
		t.Fatal("goroutine did not finish after release")
	}
}

func TestNotifySubscribersPanicRecovered(t *testing.T) {
	// 订阅者拉取 panic：goroutine 内 defer recover 兜底，进程不崩
	// （若 recover 失效，panic 会崩掉整个测试二进制，go test 直接失败）。
	notifySubscribers(func() ([]string, error) { panic("subscriber list boom") },
		&fakeMailer{}, "https://blog.example.com", &post.Post{ID: 1, Title: "T"})
	// 单条发送 panic：同样被兜住，不发下一条、不崩进程。
	notifySubscribers(func() ([]string, error) { return []string{"a@example.com", "b@example.com"}, nil },
		panicMailer{}, "https://blog.example.com", &post.Post{ID: 1, Title: "T"})
	time.Sleep(300 * time.Millisecond)
	// 走到这里即证明两个 goroutine 的 panic 均已被 recover（否则测试进程已崩）。
}
