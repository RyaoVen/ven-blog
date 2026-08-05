// /api/mcp 网关测试：中间件（Bearer/格式/吊销/大小预检）、分发与协议、14 个 action 的
// happy path 与错误映射。全部用假仓储（内存实现，本包测试专用），不起真实 MySQL。
// 服务器构造对齐 hybrid/page_test.go 的 setupTestApp 手法；MCP 路由不触发 SSR。
package interfaces

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"ven_hybird/build/application/commentapp"
	"ven_hybird/build/application/momentapp"
	"ven_hybird/build/application/postapp"
	"ven_hybird/build/application/settingsapp"
	"ven_hybird/build/application/userapp"
	"ven_hybird/build/domain/comment"
	"ven_hybird/build/domain/moment"
	"ven_hybird/build/domain/post"
	"ven_hybird/build/domain/setting"
	"ven_hybird/build/domain/user"
	"ven_hybird/hybrid"
	"ven_hybird/internal/auth"
	"ven_hybird/internal/config"
	"ven_hybird/internal/httpserver"
	"ven_hybird/internal/pagepattern"
	"ven_hybird/internal/ssr"
)

/* ===== 测试基础设施 ===== */

type fakeSSRClient struct {
	submitted chan ssr.RenderTask
}

func (f *fakeSSRClient) Submit(_ context.Context, task ssr.RenderTask) error {
	select {
	case f.submitted <- task:
	default:
	}
	return nil
}

type fakeHookIDs struct{}

func (fakeHookIDs) New() (string, error) { return "hook-test", nil }

// fakeKeys 假 KeyAuthenticator（记录调用次数，供"格式错不调 AuthenticateKey"断言）。
type fakeKeys struct {
	mu    sync.Mutex
	calls int
	fn    func(rawKey string) (int64, error)
}

func (f *fakeKeys) AuthenticateKey(rawKey string) (int64, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	return f.fn(rawKey)
}

func (f *fakeKeys) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// mcpTestEnv 是 MCP 测试环境：完整服务链 + 假仓储，可直接断言仓储状态。
type mcpTestEnv struct {
	app          *hybrid.App
	server       *httpserver.Server
	keys         *fakeKeys
	posts        *postapp.Service
	moments      *momentapp.Service
	comments     *commentapp.Service
	settings     *settingsapp.Service
	users        *userapp.Service
	postRepo     *fakePostRepo
	momentRepo   *fakeMomentRepo
	commentRepo  *fakeCommentRepo
	settingRepo  *fakeSettingRepo
	userRepo     *fakeUserRepo
	authorFn     func() (*user.User, error)
	authorNameFn func() string
	authorErr    error // 注入 authorFn 错误（author.get 错误映射用例）
}

// newMCPTestEnv 构造测试环境；authenticate 为 nil 时使用默认实现
// （"ven_valid" → userID 1，其余报错）。
func newMCPTestEnv(t *testing.T, authenticate func(rawKey string) (int64, error)) *mcpTestEnv {
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

	env := &mcpTestEnv{
		app:         app,
		server:      server,
		postRepo:    &fakePostRepo{posts: map[int64]*post.Post{}},
		momentRepo:  &fakeMomentRepo{moments: map[int64]*moment.Moment{}},
		commentRepo: &fakeCommentRepo{comments: map[int64]*comment.Comment{}, reviewed: map[int64]bool{}},
		settingRepo: &fakeSettingRepo{values: map[string]string{}},
		userRepo:    newFakeUserRepo(),
	}
	env.keys = &fakeKeys{fn: func(rawKey string) (int64, error) {
		if rawKey == "ven_valid" {
			return 1, nil
		}
		return 0, errors.New("unknown key")
	}}
	if authenticate != nil {
		env.keys = &fakeKeys{fn: authenticate}
	}
	env.authorFn = func() (*user.User, error) {
		if env.authorErr != nil {
			return nil, env.authorErr
		}
		return env.userRepo.FindByRole(user.RoleAuthor)
	}
	env.authorNameFn = func() string {
		u, err := env.authorFn()
		if err != nil {
			return ""
		}
		return u.Username
	}

	env.posts = postapp.NewService(env.postRepo)
	env.moments = momentapp.NewService(env.momentRepo)
	env.comments = commentapp.NewService(env.commentRepo, func() bool { return true })
	env.settings = settingsapp.NewService(env.settingRepo)
	env.users = userapp.NewService(env.userRepo)

	if err := RegisterMCP(app, env.keys, env.posts, env.moments, env.comments,
		env.settings, env.users, env.authorFn, env.authorNameFn); err != nil {
		t.Fatalf("RegisterMCP: %v", err)
	}
	return env
}

// call 发起 POST /api/mcp 请求；key 为空串表示不带 Authorization 头。
func (e *mcpTestEnv) call(t *testing.T, key, body string) (*http.Response, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := e.server.App().Test(req)
	if err != nil {
		t.Fatalf("mcp request failed: %v", err)
	}
	data, _ := io.ReadAll(resp.Body)
	return resp, string(data)
}

// mcpOK 解析成功响应并返回 data 对象（非 ok 直接失败）。
func mcpOK(t *testing.T, body string) map[string]any {
	t.Helper()
	var env struct {
		OK   bool           `json:"ok"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		t.Fatalf("decode ok response %q: %v", body, err)
	}
	if !env.OK {
		t.Fatalf("expected ok:true, body: %s", body)
	}
	return env.Data
}

// mcpFail 解析错误响应，返回（HTTP 状态码, 协议 code, message）。
func mcpFail(t *testing.T, resp *http.Response, body string) (int, string, string) {
	t.Helper()
	var env struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		t.Fatalf("decode error response %q: %v", body, err)
	}
	return resp.StatusCode, env.Error.Code, env.Error.Message
}

func strField(t *testing.T, m map[string]any, key string) string {
	t.Helper()
	v, ok := m[key].(string)
	if !ok {
		t.Fatalf("field %q is not string: %v", key, m[key])
	}
	return v
}

func numField(t *testing.T, m map[string]any, key string) float64 {
	t.Helper()
	v, ok := m[key].(float64)
	if !ok {
		t.Fatalf("field %q is not number: %v", key, m[key])
	}
	return v
}

func listField(t *testing.T, m map[string]any, key string) []any {
	t.Helper()
	v, ok := m[key].([]any)
	if !ok {
		t.Fatalf("field %q is not array: %v", key, m[key])
	}
	return v
}

/* ===== 假仓储（内存实现，本包测试专用，不进生产代码） ===== */

// fakePostRepo 实现 post.Repository。
type fakePostRepo struct {
	mu        sync.Mutex
	next      int64
	posts     map[int64]*post.Post
	createErr error
	updateErr error
	deleteErr error
	listErr   error
}

func (r *fakePostRepo) Create(p *post.Post) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.createErr != nil {
		return r.createErr
	}
	r.next++
	p.ID = r.next
	now := time.Now()
	p.CreatedAt, p.UpdatedAt = now, now
	p.AuthorName = "author"
	cp := *p
	r.posts[p.ID] = &cp
	return nil
}

func (r *fakePostRepo) Get(id int64) (*post.Post, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.posts[id]
	if !ok {
		return nil, post.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (r *fakePostRepo) Update(p *post.Post) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.updateErr != nil {
		return r.updateErr
	}
	old, ok := r.posts[p.ID]
	if !ok {
		return post.ErrNotFound
	}
	p.CreatedAt = old.CreatedAt
	p.UpdatedAt = time.Now()
	p.AuthorName = old.AuthorName
	cp := *p
	r.posts[p.ID] = &cp
	return nil
}

func (r *fakePostRepo) Delete(id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.deleteErr != nil {
		return r.deleteErr
	}
	if _, ok := r.posts[id]; !ok {
		return post.ErrNotFound
	}
	delete(r.posts, id)
	return nil
}

func (r *fakePostRepo) SetPinned(id int64, pinned bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.posts[id]
	if !ok {
		return post.ErrNotFound
	}
	p.Pinned = pinned
	return nil
}

func (r *fakePostRepo) ListPaged(category string, page, pageSize int) ([]*post.Post, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.listErr != nil {
		return nil, 0, r.listErr
	}
	var all []*post.Post
	for _, p := range r.posts {
		if category == "" || p.Category == category {
			all = append(all, p)
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID > all[j].ID })
	total := len(all)
	if pageSize > 0 {
		start := (page - 1) * pageSize
		if start < 0 {
			start = 0
		}
		if start >= len(all) {
			return []*post.Post{}, total, nil
		}
		end := start + pageSize
		if end > len(all) {
			end = len(all)
		}
		all = all[start:end]
	}
	return all, total, nil
}

func (r *fakePostRepo) ListByAuthor(authorID int64) ([]*post.Post, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*post.Post
	for _, p := range r.posts {
		if p.AuthorID == authorID {
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *fakePostRepo) Search(query string, limit int) ([]*post.Post, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*post.Post
	for _, p := range r.posts {
		if strings.Contains(p.Title, query) || strings.Contains(p.Content, query) {
			out = append(out, p)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *fakePostRepo) AllTags() ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	seen := map[string]struct{}{}
	for _, p := range r.posts {
		for _, tag := range p.Tags {
			seen[tag] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for tag := range seen {
		out = append(out, tag)
	}
	sort.Strings(out)
	return out, nil
}

func (r *fakePostRepo) Stats() (int, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	chars := 0
	for _, p := range r.posts {
		chars += len(p.Content)
	}
	return len(r.posts), chars, nil
}

func (r *fakePostRepo) ListFavorites(userID int64) ([]*post.Post, error) { return nil, nil }
func (r *fakePostRepo) DailyPublication(days int) ([]post.DayPublication, error) {
	return nil, nil
}
func (r *fakePostRepo) CategoryCounts() ([]post.CategoryCount, error) { return nil, nil }

func (r *fakePostRepo) CountByCategory(category string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, p := range r.posts {
		if p.Category == category {
			n++
		}
	}
	return n, nil
}

func (r *fakePostRepo) UpdateCategory(from, to string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.posts {
		if p.Category == from {
			p.Category = to
		}
	}
	return nil
}

// fakeMomentRepo 实现 moment.Repository。
type fakeMomentRepo struct {
	mu      sync.Mutex
	next    int64
	moments map[int64]*moment.Moment
	listErr error
}

func (r *fakeMomentRepo) List(limit int) ([]*moment.Moment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.listErr != nil {
		return nil, r.listErr
	}
	var all []*moment.Moment
	for _, m := range r.moments {
		all = append(all, m)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID > all[j].ID })
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

func (r *fakeMomentRepo) Create(m *moment.Moment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	m.ID = r.next
	m.CreatedAt = time.Now()
	m.AuthorName = "author"
	cp := *m
	r.moments[m.ID] = &cp
	return nil
}

func (r *fakeMomentRepo) Get(id int64) (*moment.Moment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.moments[id]
	if !ok {
		return nil, moment.ErrNotFound
	}
	cp := *m
	return &cp, nil
}

func (r *fakeMomentRepo) Delete(id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.moments[id]; !ok {
		return moment.ErrNotFound
	}
	delete(r.moments, id)
	return nil
}

func (r *fakeMomentRepo) SetPinned(id int64, pinned bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.moments[id]
	if !ok {
		return moment.ErrNotFound
	}
	m.Pinned = pinned
	return nil
}

func (r *fakeMomentRepo) Count() (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.moments), nil
}

func (r *fakeMomentRepo) DailyCounts(days int) (map[string]int, error) { return nil, nil }

// fakeCommentRepo 实现 comment.Repository。
// reviewed 模拟 ai_reviewed_at（claim 抢占/回滚），ListUnreviewedPending 排除已打标。
type fakeCommentRepo struct {
	mu       sync.Mutex
	next     int64
	comments map[int64]*comment.Comment
	reviewed map[int64]bool
	listErr  error
}

func (r *fakeCommentRepo) Create(c *comment.Comment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	c.ID = r.next
	c.CreatedAt = time.Now()
	c.Username = "reader"
	cp := *c
	r.comments[c.ID] = &cp
	return nil
}

func (r *fakeCommentRepo) Get(id int64) (*comment.Comment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.comments[id]
	if !ok {
		return nil, comment.ErrNotFound
	}
	cp := *c
	return &cp, nil
}

func (r *fakeCommentRepo) SetStatus(id int64, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.comments[id]
	if !ok {
		return comment.ErrNotFound
	}
	c.Status = status
	c.RejectedReason = ""
	return nil
}

func (r *fakeCommentRepo) SetRejected(id int64, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.comments[id]
	if !ok {
		return comment.ErrNotFound
	}
	c.Status = comment.StatusRejected
	c.RejectedReason = reason
	return nil
}

func (r *fakeCommentRepo) ListPending() ([]*comment.Comment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.listErr != nil {
		return nil, r.listErr
	}
	var out []*comment.Comment
	for _, c := range r.comments {
		if c.Status == comment.StatusPending {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *fakeCommentRepo) ListUnreviewedPending() ([]*comment.Comment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.listErr != nil {
		return nil, r.listErr
	}
	var out []*comment.Comment
	for _, c := range r.comments {
		if c.Status == comment.StatusPending && !r.reviewed[c.ID] {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *fakeCommentRepo) ClaimAIReview(id int64) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// 与真仓储一致：仅 pending 且未审行抢占（返回是否抢到），幂等。
	if c, ok := r.comments[id]; ok && c.Status == comment.StatusPending && !r.reviewed[id] {
		r.reviewed[id] = true
		return true, nil
	}
	return false, nil
}

func (r *fakeCommentRepo) UnclaimAIReview(id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	// 与真仓储一致：仅 pending 行回滚抢占（清回未审），幂等。
	if c, ok := r.comments[id]; ok && c.Status == comment.StatusPending {
		r.reviewed[id] = false
	}
	return nil
}

func (r *fakeCommentRepo) ListRejected() ([]*comment.Comment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.listErr != nil {
		return nil, r.listErr
	}
	var out []*comment.Comment
	for _, c := range r.comments {
		if c.Status == comment.StatusRejected {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *fakeCommentRepo) ListAll(limit int) ([]*comment.Comment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.listErr != nil {
		return nil, r.listErr
	}
	var out []*comment.Comment
	for _, c := range r.comments {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *fakeCommentRepo) ListByPost(postID int64) ([]*comment.Comment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*comment.Comment
	for _, c := range r.comments {
		if c.PostID == postID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *fakeCommentRepo) ListByMoment(momentID int64) ([]*comment.Comment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*comment.Comment
	for _, c := range r.comments {
		if c.MomentID == momentID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *fakeCommentRepo) MomentCommentCounts() (map[int64]int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[int64]int{}
	for _, c := range r.comments {
		if c.MomentID > 0 {
			out[c.MomentID]++
		}
	}
	return out, nil
}

func (r *fakeCommentRepo) Count() (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.comments), nil
}

func (r *fakeCommentRepo) Delete(id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.comments[id]; !ok {
		return comment.ErrNotFound
	}
	delete(r.comments, id)
	return nil
}

// fakeSettingRepo 实现 setting.Repository。
type fakeSettingRepo struct {
	mu     sync.Mutex
	values map[string]string
}

func (r *fakeSettingRepo) Get(key string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.values[key], nil
}

func (r *fakeSettingRepo) Set(key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values[key] = value
	return nil
}

// fakeUserRepo 实现 user.Repository；预置 author 用户（ID 1，username "author"）。
type fakeUserRepo struct {
	mu        sync.Mutex
	next      int64
	users     map[int64]*user.User
	author    *user.User
	updateErr error
}

func newFakeUserRepo() *fakeUserRepo {
	author := &user.User{
		ID: 1, Username: "author", Role: user.RoleAuthor,
		Bio: "hi", AvatarURL: "http://x/avatar.png", Email: "author@example.com",
		CreatedAt: time.Now(),
	}
	return &fakeUserRepo{next: 2, users: map[int64]*user.User{1: author}, author: author}
}

func (r *fakeUserRepo) FindByID(id int64) (*user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return nil, user.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (r *fakeUserRepo) FindByUsername(name string) (*user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.Username == name {
			cp := *u
			return &cp, nil
		}
	}
	return nil, user.ErrNotFound
}

func (r *fakeUserRepo) FindByRole(role user.Role) (*user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.Role == role {
			cp := *u
			return &cp, nil
		}
	}
	return nil, user.ErrNotFound
}

func (r *fakeUserRepo) FindByEmail(email string) (*user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.Email == email {
			cp := *u
			return &cp, nil
		}
	}
	return nil, user.ErrNotFound
}

func (r *fakeUserRepo) UpdateUsername(userID int64, username string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.updateErr != nil {
		return r.updateErr
	}
	u, ok := r.users[userID]
	if !ok {
		return user.ErrNotFound
	}
	for _, other := range r.users {
		if other.ID != userID && other.Username == username {
			return user.ErrUsernameTaken
		}
	}
	u.Username = username
	if r.author != nil && r.author.ID == userID {
		r.author = u
	}
	return nil
}

func (r *fakeUserRepo) UpdateEmail(userID int64, email string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[userID]
	if !ok {
		return user.ErrNotFound
	}
	u.Email = email
	return nil
}

func (r *fakeUserRepo) UpdateProfile(userID int64, bio, avatarURL string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.updateErr != nil {
		return r.updateErr
	}
	u, ok := r.users[userID]
	if !ok {
		return user.ErrNotFound
	}
	u.Bio, u.AvatarURL = bio, avatarURL
	return nil
}

func (r *fakeUserRepo) UpdatePassword(userID int64, passwordHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[userID]
	if !ok {
		return user.ErrNotFound
	}
	u.PasswordHash = passwordHash
	return nil
}

func (r *fakeUserRepo) Create(u *user.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, other := range r.users {
		if other.Username == u.Username {
			return user.ErrUsernameTaken
		}
	}
	r.next++
	u.ID = r.next
	u.CreatedAt = time.Now()
	cp := *u
	r.users[u.ID] = &cp
	return nil
}

func (r *fakeUserRepo) Count() (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.users), nil
}

func (r *fakeUserRepo) DailyRegistrations(days int) ([]user.DayCount, error) { return nil, nil }
func (r *fakeUserRepo) CountSince(t time.Time) (int, error)                  { return 0, nil }
func (r *fakeUserRepo) CountPosts(userID int64) (int, error)                 { return 0, nil }
func (r *fakeUserRepo) CountComments(userID int64) (int, error)              { return 0, nil }

/* ===== 种子辅助 ===== */

func (e *mcpTestEnv) seedPost(id int64, title, category string) {
	e.postRepo.mu.Lock()
	defer e.postRepo.mu.Unlock()
	e.postRepo.posts[id] = &post.Post{
		ID: id, AuthorID: e.userRepo.author.ID, AuthorName: "author",
		Title: title, Category: category, Content: "# " + title,
		Summary: "s", Tags: []string{"a"}, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if id >= e.postRepo.next {
		e.postRepo.next = id + 1
	}
}

func (e *mcpTestEnv) seedMoment(id int64, content string) {
	e.momentRepo.mu.Lock()
	defer e.momentRepo.mu.Unlock()
	e.momentRepo.moments[id] = &moment.Moment{
		ID: id, AuthorID: e.userRepo.author.ID, AuthorName: "author",
		Content: content, CreatedAt: time.Now(),
	}
	if id >= e.momentRepo.next {
		e.momentRepo.next = id + 1
	}
}

func (e *mcpTestEnv) seedComment(id int64, status string, target comment.Target) {
	e.commentRepo.mu.Lock()
	defer e.commentRepo.mu.Unlock()
	e.commentRepo.comments[id] = &comment.Comment{
		ID: id, UserID: 9, Username: "reader", Content: "hello",
		Status: status, PostID: target.PostID, MomentID: target.MomentID,
		CreatedAt: time.Now(),
	}
	if id >= e.commentRepo.next {
		e.commentRepo.next = id + 1
	}
}

/* ===== 中间件测试（§9.2 M1-M8） ===== */

func TestMCPAuth_MissingHeader(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	resp, body := env.call(t, "", `{"action":"post.list"}`)
	status, code, _ := mcpFail(t, resp, body)
	if status != http.StatusUnauthorized || code != mcpCodeInvalidKey {
		t.Fatalf("expected 401 invalid_key, got %d %s (%s)", status, code, body)
	}
	// 401 不设置 X-Ven-Login-Path（与 web cookie 鉴权链隔离）
	if got := resp.Header.Get("X-Ven-Login-Path"); got != "" {
		t.Fatalf("expected no X-Ven-Login-Path, got %q", got)
	}
}

func TestMCPAuth_NonBearer(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/mcp", strings.NewReader(`{"action":"post.list"}`))
	req.Header.Set("Authorization", "Basic xyz")
	resp, err := env.server.App().Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	status, code, _ := mcpFail(t, resp, string(body))
	if status != http.StatusUnauthorized || code != mcpCodeInvalidKey {
		t.Fatalf("expected 401 invalid_key, got %d %s", status, code)
	}
}

func TestMCPAuth_BadKeyFormatSkipsAuthenticate(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	resp, body := env.call(t, "foo", `{"action":"post.list"}`) // Bearer foo，无 ven_ 前缀
	status, code, _ := mcpFail(t, resp, body)
	if status != http.StatusUnauthorized || code != mcpCodeInvalidKey {
		t.Fatalf("expected 401 invalid_key, got %d %s", status, code)
	}
	if n := env.keys.callCount(); n != 0 {
		t.Fatalf("expected AuthenticateKey not called, called %d times", n)
	}
}

func TestMCPAuth_UnknownKey(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	resp, body := env.call(t, "ven_unknown", `{"action":"post.list"}`)
	status, code, message := mcpFail(t, resp, body)
	if status != http.StatusUnauthorized || code != mcpCodeInvalidKey {
		t.Fatalf("expected 401 invalid_key, got %d %s", status, code)
	}
	if message != "invalid api key" {
		t.Fatalf("expected message 'invalid api key', got %q", message)
	}
}

func TestMCPAuth_RevokedKey(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	// 吊销 key 与未知 key 响应完全一致（不区分原因，不泄露 key 存在性）
	env.keys.fn = func(rawKey string) (int64, error) {
		if rawKey == "ven_revoked" {
			return 0, errors.New("api key revoked")
		}
		return 0, errors.New("unknown key")
	}
	resp, body := env.call(t, "ven_revoked", `{"action":"post.list"}`)
	status, code, _ := mcpFail(t, resp, body)
	if status != http.StatusUnauthorized || code != mcpCodeInvalidKey {
		t.Fatalf("expected 401 invalid_key, got %d %s", status, code)
	}
}

func TestMCPAuth_ZeroUserIDIsInternal(t *testing.T) {
	// 理论上不可达的防御分支：(0, nil) → internal 而非放行
	env := newMCPTestEnv(t, func(string) (int64, error) { return 0, nil })
	resp, body := env.call(t, "ven_weird", `{"action":"post.list"}`)
	status, code, _ := mcpFail(t, resp, body)
	if status != http.StatusInternalServerError || code != mcpCodeInternal {
		t.Fatalf("expected 500 internal, got %d %s", status, code)
	}
}

func TestMCPAuth_CookieIgnored(t *testing.T) {
	// 携带合法形态的会话 cookie 但无 key → 同样 401（与 cookie 鉴权隔离）
	env := newMCPTestEnv(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/mcp", strings.NewReader(`{"action":"post.list"}`))
	req.AddCookie(&http.Cookie{Name: auth.AuthCookieName, Value: "some-session-token"})
	resp, err := env.server.App().Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	status, code, _ := mcpFail(t, resp, string(body))
	if status != http.StatusUnauthorized || code != mcpCodeInvalidKey {
		t.Fatalf("expected 401 invalid_key, got %d %s", status, code)
	}
}

func TestMCPAuth_BodyTooLarge(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	huge := `{"action":"post.list","payload":{"x":"` + strings.Repeat("a", maxMCPBodyBytes+1) + `"}}`
	resp, body := env.call(t, "ven_valid", huge)
	status, code, _ := mcpFail(t, resp, body)
	if status != http.StatusRequestEntityTooLarge || code != mcpCodeBadRequest {
		t.Fatalf("expected 413 bad_request, got %d %s (%s)", status, code, body)
	}
}

func TestMCPAuth_ValidKeyPasses(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	resp, body := env.call(t, "ven_valid", `{"action":"moment.list","payload":{}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, body)
	}
	data := mcpOK(t, body)
	if got := listField(t, data, "moments"); len(got) != 0 {
		t.Fatalf("expected empty moments, got %v", got)
	}
}

/* ===== 分发与协议测试（§9.3 D1-D6） ===== */

func TestMCPDispatch_InvalidJSON(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	resp, body := env.call(t, "ven_valid", `{not-json`)
	status, code, _ := mcpFail(t, resp, body)
	if status != http.StatusBadRequest || code != mcpCodeBadRequest {
		t.Fatalf("expected 400 bad_request, got %d %s", status, code)
	}
}

func TestMCPDispatch_MissingAction(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	resp, body := env.call(t, "ven_valid", `{"payload":{}}`)
	status, code, _ := mcpFail(t, resp, body)
	if status != http.StatusBadRequest || code != mcpCodeBadRequest {
		t.Fatalf("expected 400 bad_request, got %d %s", status, code)
	}
}

func TestMCPDispatch_ActionNotString(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	resp, body := env.call(t, "ven_valid", `{"action":123}`)
	status, code, _ := mcpFail(t, resp, body)
	if status != http.StatusBadRequest || code != mcpCodeBadRequest {
		t.Fatalf("expected 400 bad_request, got %d %s", status, code)
	}
}

func TestMCPDispatch_UnknownAction(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	resp, body := env.call(t, "ven_valid", `{"action":"foo"}`)
	status, code, message := mcpFail(t, resp, body)
	if status != http.StatusBadRequest || code != mcpCodeBadRequest {
		t.Fatalf("expected 400 bad_request, got %d %s", status, code)
	}
	if !strings.Contains(message, "foo") {
		t.Fatalf("expected message containing action name, got %q", message)
	}
}

func TestMCPDispatch_PayloadNotObject(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	resp, body := env.call(t, "ven_valid", `{"action":"post.list","payload":"str"}`)
	status, code, message := mcpFail(t, resp, body)
	if status != http.StatusBadRequest || code != mcpCodeBadRequest {
		t.Fatalf("expected 400 bad_request, got %d %s", status, code)
	}
	if !strings.Contains(message, "object") {
		t.Fatalf("expected payload-object message, got %q", message)
	}
}

func TestMCPDispatch_PayloadNullNormalized(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	resp, body := env.call(t, "ven_valid", `{"action":"moment.list","payload":null}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, body)
	}
}

/* ===== post.* action（§6.1-6.4） ===== */

func TestMCPPost_CreateHappyPath(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	payload := `{"title":"你好，Agent","category":"随笔","content":"# 正文","summary":"摘要","tags":["agent","agent"]}`
	resp, body := env.call(t, "ven_valid", `{"action":"post.create","payload":`+payload+`}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, body)
	}
	data := mcpOK(t, body)
	if id := strField(t, data, "id"); id != "1" {
		t.Fatalf("expected id 1, got %s", id)
	}
	// 归属 key 校验出的 userID（验收标准第 8 条）
	p, err := env.postRepo.Get(1)
	if err != nil {
		t.Fatalf("post not stored: %v", err)
	}
	if p.AuthorID != 1 || p.Title != "你好，Agent" || len(p.Tags) != 1 || p.Tags[0] != "agent" {
		t.Fatalf("unexpected stored post: %+v", p)
	}
}

func TestMCPPost_CreateValidation(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	resp, body := env.call(t, "ven_valid", `{"action":"post.create","payload":{"title":"","category":"x","content":"y"}}`)
	status, code, message := mcpFail(t, resp, body)
	if status != http.StatusBadRequest || code != mcpCodeValidation {
		t.Fatalf("expected 400 validation, got %d %s", status, code)
	}
	if !strings.Contains(message, "title") {
		t.Fatalf("expected title message, got %q", message)
	}
}

func TestMCPPost_CreateRepoError(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	env.postRepo.createErr = errors.New("db down")
	resp, body := env.call(t, "ven_valid", `{"action":"post.create","payload":{"title":"t","category":"c","content":"y"}}`)
	status, code, _ := mcpFail(t, resp, body)
	if status != http.StatusInternalServerError || code != mcpCodeInternal {
		t.Fatalf("expected 500 internal, got %d %s", status, code)
	}
}

func TestMCPPost_UpdateHappyPath(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	env.seedPost(7, "旧标题", "随笔")
	resp, body := env.call(t, "ven_valid", `{"action":"post.update","payload":{"id":"7","title":"新标题","category":"随笔","content":"# 更新"}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, body)
	}
	data := mcpOK(t, body)
	pv, ok := data["post"].(map[string]any)
	if !ok {
		t.Fatalf("expected post object, got %v", data["post"])
	}
	if strField(t, pv, "id") != "7" || strField(t, pv, "title") != "新标题" {
		t.Fatalf("unexpected post view: %v", pv)
	}
	p, err := env.postRepo.Get(7)
	if err != nil || p.Title != "新标题" {
		t.Fatalf("post not updated: %+v err=%v", p, err)
	}
}

func TestMCPPost_UpdateNotFound(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	resp, body := env.call(t, "ven_valid", `{"action":"post.update","payload":{"id":"999","title":"t","category":"c","content":"y"}}`)
	status, code, _ := mcpFail(t, resp, body)
	if status != http.StatusNotFound || code != mcpCodeNotFound {
		t.Fatalf("expected 404 not_found, got %d %s", status, code)
	}
}

func TestMCPPost_DeleteHappyPath(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	env.seedPost(3, "t", "随笔")
	resp, body := env.call(t, "ven_valid", `{"action":"post.delete","payload":{"id":"3"}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, body)
	}
	if v := mcpOK(t, body)["deleted"]; v != true {
		t.Fatalf("expected deleted:true, got %v", v)
	}
	if _, err := env.postRepo.Get(3); err != post.ErrNotFound {
		t.Fatalf("expected post gone, err=%v", err)
	}
}

func TestMCPPost_DeleteNotFound(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	resp, body := env.call(t, "ven_valid", `{"action":"post.delete","payload":{"id":"999"}}`)
	status, code, _ := mcpFail(t, resp, body)
	if status != http.StatusNotFound || code != mcpCodeNotFound {
		t.Fatalf("expected 404 not_found, got %d %s", status, code)
	}
}

func TestMCPPost_ListLimitBranch(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	env.seedPost(1, "a", "随笔")
	env.seedPost(2, "b", "随笔")
	env.seedPost(3, "c", "随笔")
	resp, body := env.call(t, "ven_valid", `{"action":"post.list","payload":{"limit":2}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, body)
	}
	data := mcpOK(t, body)
	posts := listField(t, data, "posts")
	if len(posts) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(posts))
	}
	if _, hasTotal := data["total"]; hasTotal {
		t.Fatalf("limit branch should not include total")
	}
	// authorName 为 author 本人（验收标准第 8 条）
	first := posts[0].(map[string]any)
	if strField(t, first, "authorName") != "author" {
		t.Fatalf("unexpected authorName: %v", first)
	}
}

func TestMCPPost_ListPagedBranch(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	env.seedPost(1, "a", "随笔")
	env.seedPost(2, "b", "随笔")
	env.seedPost(3, "c", "技术")
	resp, body := env.call(t, "ven_valid", `{"action":"post.list","payload":{"category":"随笔","page":1,"pageSize":2}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, body)
	}
	data := mcpOK(t, body)
	if n := numField(t, data, "total"); n != 2 {
		t.Fatalf("expected total 2, got %v", n)
	}
	if n := numField(t, data, "page"); n != 1 {
		t.Fatalf("expected page 1, got %v", n)
	}
	if n := numField(t, data, "pageSize"); n != 2 {
		t.Fatalf("expected pageSize 2, got %v", n)
	}
	if got := listField(t, data, "posts"); len(got) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(got))
	}
}

func TestMCPPost_ListRepoError(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	env.postRepo.listErr = errors.New("db down")
	resp, body := env.call(t, "ven_valid", `{"action":"post.list","payload":{}}`)
	status, code, _ := mcpFail(t, resp, body)
	if status != http.StatusInternalServerError || code != mcpCodeInternal {
		t.Fatalf("expected 500 internal, got %d %s", status, code)
	}
}

/* ===== moment.* action（§6.5-6.7） ===== */

func TestMCPMoment_CreateHappyPath(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	resp, body := env.call(t, "ven_valid", `{"action":"moment.create","payload":{"content":"今天的随想"}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, body)
	}
	data := mcpOK(t, body)
	if id := strField(t, data, "id"); id != "1" {
		t.Fatalf("expected id 1, got %s", id)
	}
	m, err := env.momentRepo.Get(1)
	if err != nil {
		t.Fatalf("moment not stored: %v", err)
	}
	if m.AuthorID != 1 || m.Content != "今天的随想" {
		t.Fatalf("unexpected stored moment: %+v", m)
	}
}

func TestMCPMoment_CreateValidation(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	resp, body := env.call(t, "ven_valid", `{"action":"moment.create","payload":{"content":""}}`)
	status, code, _ := mcpFail(t, resp, body)
	if status != http.StatusBadRequest || code != mcpCodeValidation {
		t.Fatalf("expected 400 validation, got %d %s", status, code)
	}
}

func TestMCPMoment_DeleteHappyPath(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	env.seedMoment(5, "旧动态")
	resp, body := env.call(t, "ven_valid", `{"action":"moment.delete","payload":{"id":"5"}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, body)
	}
	if v := mcpOK(t, body)["deleted"]; v != true {
		t.Fatalf("expected deleted:true, got %v", v)
	}
	if _, err := env.momentRepo.Get(5); err != moment.ErrNotFound {
		t.Fatalf("expected moment gone, err=%v", err)
	}
}

func TestMCPMoment_DeleteNotFound(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	resp, body := env.call(t, "ven_valid", `{"action":"moment.delete","payload":{"id":"999"}}`)
	status, code, _ := mcpFail(t, resp, body)
	if status != http.StatusNotFound || code != mcpCodeNotFound {
		t.Fatalf("expected 404 not_found, got %d %s", status, code)
	}
}

func TestMCPMoment_List(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	env.seedMoment(1, "第一条")
	env.seedMoment(2, "第二条")
	resp, body := env.call(t, "ven_valid", `{"action":"moment.list","payload":{}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, body)
	}
	if got := listField(t, mcpOK(t, body), "moments"); len(got) != 2 {
		t.Fatalf("expected 2 moments, got %d", len(got))
	}
}

/* ===== comment.* action（§6.8-6.12） ===== */

func TestMCPComment_ListPending(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	env.seedComment(1, comment.StatusPending, comment.Target{PostID: 1})
	env.seedComment(2, comment.StatusPending, comment.Target{PostID: 1})
	env.seedComment(3, comment.StatusApproved, comment.Target{PostID: 1})
	env.seedComment(4, comment.StatusRejected, comment.Target{PostID: 1})
	resp, body := env.call(t, "ven_valid", `{"action":"comment.list_pending","payload":{}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, body)
	}
	items := listField(t, mcpOK(t, body), "comments")
	if len(items) != 2 {
		t.Fatalf("expected 2 pending, got %d", len(items))
	}
	for _, it := range items {
		if strField(t, it.(map[string]any), "status") != comment.StatusPending {
			t.Fatalf("expected pending only, got %v", it)
		}
	}
}

func TestMCPComment_ApprovePostHost(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	env.seedComment(1, comment.StatusPending, comment.Target{PostID: 5})
	resp, body := env.call(t, "ven_valid", `{"action":"comment.approve","payload":{"id":"1"}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, body)
	}
	data := mcpOK(t, body)
	if strField(t, data, "id") != "1" || strField(t, data, "status") != comment.StatusApproved {
		t.Fatalf("unexpected data: %v", data)
	}
	c, _ := env.commentRepo.Get(1)
	if c.Status != comment.StatusApproved {
		t.Fatalf("expected approved, got %s", c.Status)
	}
}

func TestMCPComment_ApproveMomentHost(t *testing.T) {
	// 动态宿主走 DataChange("/moments") 分支（只验证不报错，失效副作用见 §9.5 策略）
	env := newMCPTestEnv(t, nil)
	env.seedComment(1, comment.StatusPending, comment.Target{MomentID: 3})
	resp, body := env.call(t, "ven_valid", `{"action":"comment.approve","payload":{"id":"1"}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, body)
	}
}

func TestMCPComment_ApproveNotFound(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	resp, body := env.call(t, "ven_valid", `{"action":"comment.approve","payload":{"id":"999"}}`)
	status, code, _ := mcpFail(t, resp, body)
	if status != http.StatusNotFound || code != mcpCodeNotFound {
		t.Fatalf("expected 404 not_found, got %d %s", status, code)
	}
}

func TestMCPComment_RejectHappyPath(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	env.seedComment(1, comment.StatusApproved, comment.Target{PostID: 5})
	resp, body := env.call(t, "ven_valid", `{"action":"comment.reject","payload":{"id":"1","reason":"广告"}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, body)
	}
	data := mcpOK(t, body)
	if strField(t, data, "id") != "1" || strField(t, data, "status") != comment.StatusRejected {
		t.Fatalf("unexpected data: %v", data)
	}
	c, _ := env.commentRepo.Get(1)
	if c.Status != comment.StatusRejected || c.RejectedReason != "广告" {
		t.Fatalf("expected rejected with reason, got %+v", c)
	}
}

func TestMCPComment_RejectValidation(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	env.seedComment(1, comment.StatusPending, comment.Target{PostID: 5})
	resp, body := env.call(t, "ven_valid", `{"action":"comment.reject","payload":{"id":"1","reason":""}}`)
	status, code, _ := mcpFail(t, resp, body)
	if status != http.StatusBadRequest || code != mcpCodeValidation {
		t.Fatalf("expected 400 validation, got %d %s", status, code)
	}
}

func TestMCPComment_RecoverHappyPath(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	env.seedComment(1, comment.StatusRejected, comment.Target{PostID: 5})
	env.commentRepo.mu.Lock()
	env.commentRepo.comments[1].RejectedReason = "误杀"
	env.commentRepo.mu.Unlock()
	resp, body := env.call(t, "ven_valid", `{"action":"comment.recover","payload":{"id":"1"}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, body)
	}
	data := mcpOK(t, body)
	// 与设计文档的偏差：Unit 3 定稿 recover 为 rejected → approved（非文档预期的 pending）
	if strField(t, data, "id") != "1" || strField(t, data, "status") != comment.StatusApproved {
		t.Fatalf("unexpected data: %v", data)
	}
	c, _ := env.commentRepo.Get(1)
	if c.Status != comment.StatusApproved || c.RejectedReason != "" {
		t.Fatalf("expected approved with cleared reason, got %+v", c)
	}
}

func TestMCPComment_RecoverInvalidState(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	env.seedComment(1, comment.StatusApproved, comment.Target{PostID: 5})
	resp, body := env.call(t, "ven_valid", `{"action":"comment.recover","payload":{"id":"1"}}`)
	status, code, message := mcpFail(t, resp, body)
	if status != http.StatusBadRequest || code != mcpCodeValidation {
		t.Fatalf("expected 400 validation, got %d %s", status, code)
	}
	if !strings.Contains(message, "rejected") {
		t.Fatalf("expected rejected-state message, got %q", message)
	}
}

func TestMCPComment_List(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	env.seedComment(1, comment.StatusApproved, comment.Target{PostID: 1})
	env.seedComment(2, comment.StatusPending, comment.Target{PostID: 1})
	env.seedComment(3, comment.StatusRejected, comment.Target{PostID: 1})
	resp, body := env.call(t, "ven_valid", `{"action":"comment.list","payload":{}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, body)
	}
	if got := listField(t, mcpOK(t, body), "comments"); len(got) != 3 {
		t.Fatalf("expected 3 comments (default limit 100), got %d", len(got))
	}
	resp, body = env.call(t, "ven_valid", `{"action":"comment.list","payload":{"limit":2}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, body)
	}
	if got := listField(t, mcpOK(t, body), "comments"); len(got) != 2 {
		t.Fatalf("expected 2 comments with limit 2, got %d", len(got))
	}
}

/* ===== author.* action（§6.13-6.14） ===== */

func TestMCPAuthor_GetHappyPath(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	resp, body := env.call(t, "ven_valid", `{"action":"author.get","payload":{}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, body)
	}
	data := mcpOK(t, body)
	content, ok := data["content"].(map[string]any)
	if !ok {
		t.Fatalf("expected content object, got %v", data["content"])
	}
	// 空设置仓储 → 缺省回退（defaultParagraphs 非空）
	if got := listField(t, content, "paragraphs"); len(got) == 0 {
		t.Fatalf("expected default paragraphs fallback")
	}
	profile := data["profile"].(map[string]any)
	if strField(t, profile, "username") != "author" ||
		strField(t, profile, "role") != "author" ||
		strField(t, profile, "email") != "author@example.com" {
		t.Fatalf("unexpected profile: %v", profile)
	}
}

func TestMCPAuthor_GetAuthorNotFound(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	env.authorErr = user.ErrNotFound
	resp, body := env.call(t, "ven_valid", `{"action":"author.get","payload":{}}`)
	status, code, _ := mcpFail(t, resp, body)
	if status != http.StatusNotFound || code != mcpCodeNotFound {
		t.Fatalf("expected 404 not_found, got %d %s", status, code)
	}
}

func TestMCPAuthor_GetAuthorFnError(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	env.authorErr = errors.New("db down")
	resp, body := env.call(t, "ven_valid", `{"action":"author.get","payload":{}}`)
	status, code, _ := mcpFail(t, resp, body)
	if status != http.StatusInternalServerError || code != mcpCodeInternal {
		t.Fatalf("expected 500 internal, got %d %s", status, code)
	}
}

func TestMCPAuthor_UpdateContentPartial(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	resp, body := env.call(t, "ven_valid", `{"action":"author.update","payload":{"paragraphs":["p1","p2"]}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, body)
	}
	if v := mcpOK(t, body)["updated"]; v != true {
		t.Fatalf("expected updated:true, got %v", v)
	}
	if got := env.settingRepo.values[setting.KeyIntroParagraphs]; got != "p1\np2" {
		t.Fatalf("expected paragraphs stored, got %q", got)
	}
	// 部分更新：未传字段不被写入
	if _, touched := env.settingRepo.values[setting.KeySkills]; touched {
		t.Fatalf("skills should not be touched")
	}
}

func TestMCPAuthor_UpdateProfilePartial(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	resp, body := env.call(t, "ven_valid", `{"action":"author.update","payload":{"bio":"新简介"}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, body)
	}
	mcpOK(t, body)
	u, _ := env.userRepo.FindByID(1)
	if u.Bio != "新简介" {
		t.Fatalf("expected bio updated, got %q", u.Bio)
	}
	if u.AvatarURL != "http://x/avatar.png" {
		t.Fatalf("avatar should stay unchanged, got %q", u.AvatarURL)
	}
}

func TestMCPAuthor_UpdateUsername(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	resp, body := env.call(t, "ven_valid", `{"action":"author.update","payload":{"username":"author2"}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, body)
	}
	if u := strField(t, mcpOK(t, body), "username"); u != "author2" {
		t.Fatalf("expected new username author2, got %s", u)
	}
	u, _ := env.userRepo.FindByRole(user.RoleAuthor)
	if u.Username != "author2" {
		t.Fatalf("expected repo username author2, got %q", u.Username)
	}
	// authorFn 随改名生效
	cur, _ := env.authorFn()
	if cur.Username != "author2" {
		t.Fatalf("expected authorFn to reflect rename, got %q", cur.Username)
	}
}

func TestMCPAuthor_UpdateUsernameTaken(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	env.userRepo.mu.Lock()
	env.userRepo.users[2] = &user.User{ID: 2, Username: "author2", Role: user.RoleReader}
	env.userRepo.mu.Unlock()
	resp, body := env.call(t, "ven_valid", `{"action":"author.update","payload":{"username":"author2"}}`)
	status, code, message := mcpFail(t, resp, body)
	if status != http.StatusBadRequest || code != mcpCodeValidation {
		t.Fatalf("expected 400 validation, got %d %s", status, code)
	}
	if message != "username taken" {
		t.Fatalf("expected 'username taken', got %q", message)
	}
}

func TestMCPAuthor_UpdateBioTooLong(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	payload := `{"action":"author.update","payload":{"bio":"` + strings.Repeat("长", 201) + `"}}`
	resp, body := env.call(t, "ven_valid", payload)
	status, code, _ := mcpFail(t, resp, body)
	if status != http.StatusBadRequest || code != mcpCodeValidation {
		t.Fatalf("expected 400 validation, got %d %s", status, code)
	}
}

func TestMCPAuthor_UpdateNoFields(t *testing.T) {
	env := newMCPTestEnv(t, nil)
	resp, body := env.call(t, "ven_valid", `{"action":"author.update","payload":{}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, body)
	}
	if v := mcpOK(t, body)["updated"]; v != true {
		t.Fatalf("expected updated:true, got %v", v)
	}
}
