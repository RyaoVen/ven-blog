package settingsapp

import (
	"errors"
	"testing"

	"ven_hybird/build/domain/setting"
)

// fakeRepo 内存实现 setting.Repository（测试用）：
// 语义与真实仓储一致——Get 不存在返回空串与 nil，Set 为 upsert。
type fakeRepo struct {
	values map[string]string
	fail   bool // 置 true 后所有操作报错（错误传播用例）
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{values: map[string]string{}}
}

func (f *fakeRepo) Get(key string) (string, error) {
	if f.fail {
		return "", errors.New("boom")
	}
	return f.values[key], nil
}

func (f *fakeRepo) Set(key, value string) error {
	if f.fail {
		return errors.New("boom")
	}
	f.values[key] = value
	return nil
}

// 接口合规断言。
var _ setting.Repository = (*fakeRepo)(nil)

func TestAuthEnabled(t *testing.T) {
	t.Run("default on when unset", func(t *testing.T) {
		svc := NewService(newFakeRepo())
		on, err := svc.AuthEnabled()
		if err != nil {
			t.Fatalf("AuthEnabled: %v", err)
		}
		if !on {
			t.Fatal("unset key: AuthEnabled = false, want true (默认开)")
		}
	})

	t.Run("explicit on and off roundtrip", func(t *testing.T) {
		repo := newFakeRepo()
		svc := NewService(repo)
		if err := svc.SetAuthEnabled(true); err != nil {
			t.Fatalf("SetAuthEnabled(true): %v", err)
		}
		if repo.values[setting.KeyUserAuthEnabled] != "on" {
			t.Fatalf("SetAuthEnabled(true) wrote %q, want on", repo.values[setting.KeyUserAuthEnabled])
		}
		if on, _ := svc.AuthEnabled(); !on {
			t.Fatal("after SetAuthEnabled(true): AuthEnabled = false, want true")
		}
		if err := svc.SetAuthEnabled(false); err != nil {
			t.Fatalf("SetAuthEnabled(false): %v", err)
		}
		if repo.values[setting.KeyUserAuthEnabled] != "off" {
			t.Fatalf("SetAuthEnabled(false) wrote %q, want off", repo.values[setting.KeyUserAuthEnabled])
		}
		if on, _ := svc.AuthEnabled(); on {
			t.Fatal("after SetAuthEnabled(false): AuthEnabled = true, want false")
		}
	})
}

// 未设置（库里无键）时评论总开关默认开。
func TestCommentsEnabledDefaultOn(t *testing.T) {
	svc := NewService(newFakeRepo())
	on, err := svc.CommentsEnabled()
	if err != nil {
		t.Fatalf("CommentsEnabled() error = %v", err)
	}
	if !on {
		t.Fatal("CommentsEnabled() = false, want true (default on)")
	}
}

// 显式存 "on" 视为开（存量键值兼容）。
func TestCommentsEnabledStoredOn(t *testing.T) {
	repo := newFakeRepo()
	repo.values[setting.KeyCommentsEnabled] = "on"
	svc := NewService(repo)
	on, err := svc.CommentsEnabled()
	if err != nil {
		t.Fatalf("CommentsEnabled() error = %v", err)
	}
	if !on {
		t.Fatal("CommentsEnabled() = false, want true for stored \"on\"")
	}
}

// SetCommentsEnabled(false) 落库 "off" 并读回关闭。
func TestSetCommentsEnabledOff(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	if err := svc.SetCommentsEnabled(false); err != nil {
		t.Fatalf("SetCommentsEnabled(false) error = %v", err)
	}
	if got := repo.values[setting.KeyCommentsEnabled]; got != "off" {
		t.Fatalf("stored value = %q, want %q", got, "off")
	}
	on, err := svc.CommentsEnabled()
	if err != nil {
		t.Fatalf("CommentsEnabled() error = %v", err)
	}
	if on {
		t.Fatal("CommentsEnabled() = true, want false after SetCommentsEnabled(false)")
	}
}

// SetCommentsEnabled(true) 落库 "on" 并读回开启。
func TestSetCommentsEnabledOn(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	if err := svc.SetCommentsEnabled(true); err != nil {
		t.Fatalf("SetCommentsEnabled(true) error = %v", err)
	}
	if got := repo.values[setting.KeyCommentsEnabled]; got != "on" {
		t.Fatalf("stored value = %q, want %q", got, "on")
	}
	on, err := svc.CommentsEnabled()
	if err != nil {
		t.Fatalf("CommentsEnabled() error = %v", err)
	}
	if !on {
		t.Fatal("CommentsEnabled() = false, want true after SetCommentsEnabled(true)")
	}
}

// 仓储报错时开关方法透传错误（不吞错、不误报默认值）。
func TestCommentsEnabledRepoError(t *testing.T) {
	repo := newFakeRepo()
	repo.fail = true
	svc := NewService(repo)
	if _, err := svc.CommentsEnabled(); err == nil {
		t.Fatal("CommentsEnabled() error = nil, want repo error propagated")
	}
	if err := svc.SetCommentsEnabled(false); err == nil {
		t.Fatal("SetCommentsEnabled(false) error = nil, want repo error propagated")
	}
}

// SiteURL 未设置时返回空串与 nil（调用方回退 env/默认，不误报）。
func TestSiteURLDefaultEmpty(t *testing.T) {
	svc := NewService(newFakeRepo())
	u, err := svc.SiteURL()
	if err != nil {
		t.Fatalf("SiteURL() error = %v", err)
	}
	if u != "" {
		t.Fatalf("SiteURL() = %q, want empty", u)
	}
}

// SetSiteURL 落库 site_url 键并读回（往返一致）。
func TestSetSiteURLRoundtrip(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	if err := svc.SetSiteURL("https://blog.example.com"); err != nil {
		t.Fatalf("SetSiteURL error = %v", err)
	}
	if got := repo.values[setting.KeySiteURL]; got != "https://blog.example.com" {
		t.Fatalf("stored value = %q, want %q", got, "https://blog.example.com")
	}
	u, err := svc.SiteURL()
	if err != nil {
		t.Fatalf("SiteURL() error = %v", err)
	}
	if u != "https://blog.example.com" {
		t.Fatalf("SiteURL() = %q, want roundtrip value", u)
	}
}

// 仓储报错时 SiteURL/SetSiteURL 透传错误。
func TestSiteURLRepoError(t *testing.T) {
	repo := newFakeRepo()
	repo.fail = true
	svc := NewService(repo)
	if _, err := svc.SiteURL(); err == nil {
		t.Fatal("SiteURL() error = nil, want repo error propagated")
	}
	if err := svc.SetSiteURL("https://blog.example.com"); err == nil {
		t.Fatal("SetSiteURL error = nil, want repo error propagated")
	}
}
