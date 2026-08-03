// 认证限速测试：/auth/login 失败锁定与成功清零、/auth/email/code 每邮箱/每 IP 节流。
// 复用 mcp_test.go 的假仓储与 fakeSSRClient/fakeHookIDs；限速器参数直接注入（不进 register.go）。
package interfaces

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"ven_hybird/build/application/emailauth"
	"ven_hybird/build/application/ratelimit"
	"ven_hybird/build/application/settingsapp"
	"ven_hybird/build/application/userapp"
	"ven_hybird/build/domain/emailcode"
	"ven_hybird/build/domain/user"
	"ven_hybird/hybrid"
	"ven_hybird/internal/config"
	"ven_hybird/internal/httpserver"
	"ven_hybird/internal/pagepattern"
	"ven_hybird/internal/ssr"
)

/* ===== 测试基础设施 ===== */

// authTestEnv 是认证接口测试环境：完整服务链 + 假仓储（不起真实 MySQL）。
type authTestEnv struct {
	app      *hybrid.App
	server   *httpserver.Server
	userRepo *fakeUserRepo
	codeRepo *fakeEmailCodeRepo
	mailer   *fakeCodeMailer
}

// newAuthTestEnv 构造认证测试环境：限速器由用例注入（默认宽松参数），注册登录与邮箱认证路由。
func newAuthTestEnv(t *testing.T, loginLimit, codeEmailLimit, codeIPLimit *ratelimit.Limiter) *authTestEnv {
	t.Helper()
	cfg := config.Config{
		NodeSubmitTimeout: 5 * time.Second,
		RenderTimeout:     10 * time.Second,
	}
	cfg.IsrDir = t.TempDir()
	cfg.IsrEnabled = true
	client := &fakeSSRClient{submitted: make(chan ssr.RenderTask, 1)}
	pending := ssr.NewPendingRegistry(10)
	patterns := pagepattern.NewValidator(nil)
	server := httpserver.New(cfg, client, pending, fakeHookIDs{}, patterns)
	app := hybrid.New(server)
	// 成功登录会下发会话 cookie（GrantAuthWithUser 需先解析角色），注册 reader 角色
	if err := app.RegisterRole("reader", nil); err != nil {
		t.Fatalf("注册 reader 角色失败: %v", err)
	}
	env := &authTestEnv{
		app:      app,
		server:   server,
		userRepo: newFakeUserRepo(),
		codeRepo: &fakeEmailCodeRepo{codes: map[string]*emailcode.Entry{}},
		mailer:   &fakeCodeMailer{},
	}
	users := userapp.NewService(env.userRepo)
	settings := settingsapp.NewService(&fakeSettingRepo{values: map[string]string{}})
	emailAuth := emailauth.NewService(env.codeRepo, env.userRepo, env.mailer)
	RegisterAuth(app, users, settings, loginLimit)
	RegisterEmailAuth(app, emailAuth, users, settings, "http://127.0.0.1:8080", codeEmailLimit, codeIPLimit)
	return env
}

// post 发起 POST 请求并返回响应与响应体。
func (e *authTestEnv) post(t *testing.T, path, body string) (*http.Response, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// 显式 5s 超时：-race 下 bcrypt/限速循环减速，fiber Test 默认 1s 易 flake
	resp, err := e.server.App().Test(req, 5000)
	if err != nil {
		t.Fatalf("%s 请求失败: %v", path, err)
	}
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, string(data)
}

// addUser 往假仓储写入带密码哈希的用户（MinCost 加速测试）。
func (e *authTestEnv) addUser(t *testing.T, username, password string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("生成测试密码哈希失败: %v", err)
	}
	u := &user.User{Username: username, PasswordHash: string(hash), Role: user.RoleReader, Email: username + "@example.com"}
	if err := e.userRepo.Create(u); err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
}

/* ===== 假仓储（本文件专用） ===== */

// fakeEmailCodeRepo 实现 emailcode.Repository（内存）。
type fakeEmailCodeRepo struct {
	mu      sync.Mutex
	codes   map[string]*emailcode.Entry
	next    int64
	creates int
}

func (r *fakeEmailCodeRepo) Create(email, codeHash string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	r.creates++
	r.codes[email] = &emailcode.Entry{ID: r.next, Email: email, CodeHash: codeHash, ExpiresAt: expiresAt}
	return nil
}

func (r *fakeEmailCodeRepo) Latest(email string) (*emailcode.Entry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.codes[email]; ok {
		cp := *e
		return &cp, nil
	}
	return nil, nil
}

func (r *fakeEmailCodeRepo) IncrAttempts(id int64) error { return nil }

func (r *fakeEmailCodeRepo) Delete(id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, e := range r.codes {
		if e.ID == id {
			delete(r.codes, k)
		}
	}
	return nil
}

// fakeCodeMailer 实现 emailauth.Mailer（只记录发送次数）。
type fakeCodeMailer struct {
	mu   sync.Mutex
	sent int
}

func (m *fakeCodeMailer) Send(_, _, _ string) error { return nil }
func (m *fakeCodeMailer) SendHTML(_, _, _ string) error {
	m.mu.Lock()
	m.sent++
	m.mu.Unlock()
	return nil
}

/* ===== 用例 ===== */

// TestLoginRateLimit_LockAndUnlock 失败 5 次锁定，窗口过期自动解锁。
func TestLoginRateLimit_LockAndUnlock(t *testing.T) {
	lim := ratelimit.New(5, 100*time.Millisecond)
	env := newAuthTestEnv(t, lim, ratelimit.New(1000, time.Minute), ratelimit.New(1000, 24*time.Hour))
	body := `{"username":"author","password":"wrong"}`
	for i := 0; i < 5; i++ {
		resp, _ := env.post(t, "/auth/login", body)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("第 %d 次失败登录应 401，got %d", i+1, resp.StatusCode)
		}
	}
	for i := 0; i < 2; i++ {
		resp, _ := env.post(t, "/auth/login", body)
		if resp.StatusCode != http.StatusTooManyRequests {
			t.Fatalf("锁定后应 429，got %d", resp.StatusCode)
		}
	}
	time.Sleep(300 * time.Millisecond) // 越过 100ms 锁定窗口
	resp, _ := env.post(t, "/auth/login", body)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("窗口过期应解锁（恢复 401），got %d", resp.StatusCode)
	}
}

// TestLoginRateLimit_SuccessResetsFailures 成功登录清零失败计数，不误伤正常用户。
func TestLoginRateLimit_SuccessResetsFailures(t *testing.T) {
	lim := ratelimit.New(5, time.Hour)
	env := newAuthTestEnv(t, lim, ratelimit.New(1000, time.Minute), ratelimit.New(1000, 24*time.Hour))
	env.addUser(t, "alice", "secret123")
	for i := 0; i < 2; i++ {
		resp, _ := env.post(t, "/auth/login", `{"username":"alice","password":"wrong"}`)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("第 %d 次错误密码应 401，got %d", i+1, resp.StatusCode)
		}
	}
	resp, _ := env.post(t, "/auth/login", `{"username":"alice","password":"secret123"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("正确密码应 200，got %d", resp.StatusCode)
	}
	// 清零后再失败 3 次：若未清零（2+3=5 达阈值）会被锁定，清零后仅计 3 次
	for i := 0; i < 3; i++ {
		resp, _ := env.post(t, "/auth/login", `{"username":"alice","password":"wrong"}`)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("第 %d 次错误密码应 401，got %d", i+1, resp.StatusCode)
		}
	}
	resp, _ = env.post(t, "/auth/login", `{"username":"alice","password":"wrong"}`)
	if resp.StatusCode == http.StatusTooManyRequests {
		t.Fatal("成功登录后失败计数应已清零，不应触发锁定")
	}
}

// TestEmailCodeThrottle_PerEmail 同一邮箱 1 次/分钟：重复发码 429 且不再真正发码。
func TestEmailCodeThrottle_PerEmail(t *testing.T) {
	env := newAuthTestEnv(t, ratelimit.New(5, time.Hour), ratelimit.New(1, time.Minute), ratelimit.New(1000, 24*time.Hour))
	resp, _ := env.post(t, "/auth/email/code", `{"email":"u1@example.com"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("首次发码应 200，got %d", resp.StatusCode)
	}
	resp, _ = env.post(t, "/auth/email/code", `{"email":"u1@example.com"}`)
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("1 分钟内重复发码应 429，got %d", resp.StatusCode)
	}
	if n := env.codeRepo.creates; n != 1 {
		t.Fatalf("被限速的请求不应再生成验证码，creates=%d", n)
	}
	resp, _ = env.post(t, "/auth/email/code", `{"email":"u2@example.com"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("其他邮箱不受限，应 200，got %d", resp.StatusCode)
	}
}

// TestEmailCodeThrottle_PerIP 每 IP 每日上限 50：第 51 次 429。
func TestEmailCodeThrottle_PerIP(t *testing.T) {
	env := newAuthTestEnv(t, ratelimit.New(5, time.Hour), ratelimit.New(1000, time.Minute), ratelimit.New(50, 24*time.Hour))
	for i := 0; i < 50; i++ {
		resp, _ := env.post(t, "/auth/email/code", fmt.Sprintf(`{"email":"ip%d@example.com"}`, i))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("第 %d 次发码应 200，got %d", i+1, resp.StatusCode)
		}
	}
	resp, _ := env.post(t, "/auth/email/code", `{"email":"ip50@example.com"}`)
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("第 51 次发码（超每 IP 每日上限）应 429，got %d", resp.StatusCode)
	}
	if n := env.codeRepo.creates; n != 50 {
		t.Fatalf("被限速的请求不应再生成验证码，creates=%d", n)
	}
}
