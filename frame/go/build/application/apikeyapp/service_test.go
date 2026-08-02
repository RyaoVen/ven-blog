// 纯逻辑测试：假 repo（内存 map）驱动 apikeyapp 各用例，不依赖 MySQL。
package apikeyapp

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"ven_hybird/build/domain/apikey"
)

// fakeRepo 是 apikey.Repository 的内存实现：按 hash 索引 + 按 user 索引，记录 UpdateLastUsedAt 调用次数。
type fakeRepo struct {
	mu       sync.Mutex
	byHash   map[string]*apikey.ApiKey
	byUser   map[int64][]*apikey.ApiKey
	lastUsed int // UpdateLastUsedAt 调用次数
	nextID   int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		byHash: map[string]*apikey.ApiKey{},
		byUser: map[int64][]*apikey.ApiKey{},
	}
}

func (f *fakeRepo) Create(k *apikey.ApiKey) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, dup := f.byHash[k.KeyHash]; dup {
		return errors.New("duplicate key hash")
	}
	f.nextID++
	k.ID = f.nextID
	if k.CreatedAt.IsZero() {
		k.CreatedAt = time.Now()
	}
	f.byHash[k.KeyHash] = k
	f.byUser[k.UserID] = append(f.byUser[k.UserID], k)
	return nil
}

func (f *fakeRepo) FindByHash(hash string) (*apikey.ApiKey, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	k, ok := f.byHash[hash]
	if !ok {
		return nil, apikey.ErrNotFound
	}
	cp := *k
	return &cp, nil
}

func (f *fakeRepo) ListByUser(userID int64) ([]*apikey.ApiKey, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	list := f.byUser[userID]
	out := make([]*apikey.ApiKey, 0, len(list))
	// id 递增，逆序 = 创建时间倒序
	for i := len(list) - 1; i >= 0; i-- {
		cp := *list[i]
		out = append(out, &cp)
	}
	return out, nil
}

func (f *fakeRepo) Revoke(userID, id int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, k := range f.byUser[userID] {
		if k.ID == id {
			if !k.RevokedAt.IsZero() {
				return apikey.ErrNotFound // 已吊销 → 统一不泄露
			}
			k.RevokedAt = time.Now()
			return nil
		}
	}
	return apikey.ErrNotFound // 不存在 / 非本人
}

func (f *fakeRepo) UpdateLastUsedAt(id int64, t time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastUsed++
	return nil
}

func mustCreateKey(t *testing.T, s *Service, userID int64, name string) (raw string, view KeyView) {
	t.Helper()
	raw, view, err := s.CreateKey(userID, name)
	if err != nil {
		t.Fatalf("CreateKey(%q) error: %v", name, err)
	}
	return raw, view
}

// 用例 1：创建返回明文（ven_ 前缀、总长 47、两次互不相同）；落库的是哈希而非明文。
func TestCreateKeyPlaintext(t *testing.T) {
	repo := newFakeRepo()
	s := NewService(repo)
	raw1, view1 := mustCreateKey(t, s, 1, "zcode-agent")
	raw2, view2 := mustCreateKey(t, s, 1, "zcode-agent-2")

	if !strings.HasPrefix(raw1, apikey.KeyPrefix) {
		t.Errorf("raw %q should start with %q", raw1, apikey.KeyPrefix)
	}
	if len(raw1) != 47 {
		t.Errorf("raw length = %d, want 47", len(raw1))
	}
	if raw1 == raw2 {
		t.Error("two generated keys should differ (randomness)")
	}
	if view1.Prefix != raw1[:8] || view2.Prefix != raw2[:8] {
		t.Errorf("view prefix mismatch: %q/%q vs %q/%q", view1.Prefix, view2.Prefix, raw1[:8], raw2[:8])
	}

	// 假 repo 里只应有哈希键，不存在明文
	stored, err := repo.FindByHash(apikey.HashKey(raw1))
	if err != nil {
		t.Fatalf("stored key by hash not found: %v", err)
	}
	if stored.KeyHash == raw1 || strings.Contains(stored.KeyHash, raw1) {
		t.Error("repo must store hash, never plaintext")
	}
}

// 用例 2：name 校验边界。
func TestCreateKeyNameValidation(t *testing.T) {
	s := NewService(newFakeRepo())
	for _, name := range []string{"", "   ", strings.Repeat("a", 65)} {
		_, _, err := s.CreateKey(1, name)
		var vErr *ValidationError
		if !errors.As(err, &vErr) {
			t.Errorf("CreateKey(%q) error = %v, want *ValidationError", name, err)
		}
	}
	raw, view, err := s.CreateKey(1, "  zcode-agent  ")
	if err != nil {
		t.Fatalf("valid name rejected: %v", err)
	}
	if view.Name != "zcode-agent" {
		t.Errorf("view.Name = %q, want trimmed %q", view.Name, "zcode-agent")
	}
	if view.Prefix != raw[:8] {
		t.Errorf("view.Prefix = %q, want %q", view.Prefix, raw[:8])
	}
}

// 用例 3：列表倒序、脱敏视图结构（不含明文/哈希字段）。
func TestListKeys(t *testing.T) {
	repo := newFakeRepo()
	s := NewService(repo)
	raw1, _ := mustCreateKey(t, s, 1, "first")
	raw2, _ := mustCreateKey(t, s, 1, "second")
	mustCreateKey(t, s, 2, "other-user") // 他人 key 不应混入

	views, err := s.ListKeys(1)
	if err != nil {
		t.Fatalf("ListKeys error: %v", err)
	}
	if len(views) != 2 {
		t.Fatalf("ListKeys(1) returned %d keys, want 2", len(views))
	}
	// 倒序：后创建的 second 在前
	if views[0].Name != "second" || views[1].Name != "first" {
		t.Errorf("order = [%q, %q], want [second, first]", views[0].Name, views[1].Name)
	}
	// 视图字段结构断言：KeyView 无明文字段（编译期已保证），此处运行时兜底——任何字段不得含明文
	for _, v := range views {
		if strings.Contains(v.ID+v.Name+v.Prefix, raw1) || strings.Contains(v.ID+v.Name+v.Prefix, raw2) {
			t.Errorf("view leaks plaintext: %+v", v)
		}
		if v.ID == "" || v.Prefix == "" {
			t.Errorf("view missing id/prefix: %+v", v)
		}
		if v.LastUsedAt != nil {
			t.Errorf("new key should have LastUsedAt nil: %+v", v)
		}
		if v.RevokedAt != nil {
			t.Errorf("new key should have RevokedAt nil: %+v", v)
		}
	}
}

// 用例 4：鉴权——正确明文换 userID 且更新 last_used_at；错误明文 ErrNotFound；吊销后立即 ErrRevoked。
func TestAuthenticateKey(t *testing.T) {
	repo := newFakeRepo()
	s := NewService(repo)
	raw, _ := mustCreateKey(t, s, 42, "agent")

	userID, err := s.AuthenticateKey(raw)
	if err != nil {
		t.Fatalf("AuthenticateKey(valid) error: %v", err)
	}
	if userID != 42 {
		t.Errorf("userID = %d, want 42", userID)
	}
	if repo.lastUsed == 0 {
		t.Error("UpdateLastUsedAt should be called on success")
	}

	if _, err := s.AuthenticateKey("ven_" + strings.Repeat("A", 43)); !errors.Is(err, apikey.ErrNotFound) {
		t.Errorf("wrong key error = %v, want ErrNotFound", err)
	}

	// 吊销前正常 → 吊销后立即失败（即时生效）
	if err := s.Revoke(42, 1); err != nil {
		t.Fatalf("Revoke error: %v", err)
	}
	if _, err := s.AuthenticateKey(raw); !errors.Is(err, apikey.ErrRevoked) {
		t.Errorf("revoked key error = %v, want ErrRevoked", err)
	}
}

// 用例 5：吊销——本人可吊销；重复吊销 / 非本人 → ErrNotFound。
func TestRevoke(t *testing.T) {
	s := NewService(newFakeRepo())
	raw, view := mustCreateKey(t, s, 7, "agent")

	id := mustViewID(t, view)
	if err := s.Revoke(7, id); err != nil {
		t.Fatalf("first revoke error: %v", err)
	}
	if err := s.Revoke(7, id); !errors.Is(err, apikey.ErrNotFound) {
		t.Errorf("second revoke error = %v, want ErrNotFound", err)
	}
	if err := s.Revoke(999, id); !errors.Is(err, apikey.ErrNotFound) {
		t.Errorf("non-owner revoke error = %v, want ErrNotFound", err)
	}
	if _, err := s.AuthenticateKey(raw); !errors.Is(err, apikey.ErrRevoked) {
		t.Errorf("revoked key error = %v, want ErrRevoked", err)
	}
}

// mustViewID 解析视图 ID 为 int64（测试助手）。
func mustViewID(t *testing.T, view KeyView) int64 {
	t.Helper()
	id, err := strconv.ParseInt(view.ID, 10, 64)
	if err != nil {
		t.Fatalf("parse view id %q: %v", view.ID, err)
	}
	return id
}
