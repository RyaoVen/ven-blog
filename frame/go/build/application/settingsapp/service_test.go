package settingsapp

import (
	"testing"

	"ven_hybird/build/domain/setting"
)

// fakeRepo 内存实现 setting.Repository（测试用，仅覆盖本包用到的行为）。
type fakeRepo struct {
	kv map[string]string
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{kv: map[string]string{}}
}

func (f *fakeRepo) Get(key string) (string, error) {
	return f.kv[key], nil
}

func (f *fakeRepo) Set(key, value string) error {
	f.kv[key] = value
	return nil
}

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
		if repo.kv[setting.KeyUserAuthEnabled] != "on" {
			t.Fatalf("SetAuthEnabled(true) wrote %q, want on", repo.kv[setting.KeyUserAuthEnabled])
		}
		if on, _ := svc.AuthEnabled(); !on {
			t.Fatal("after SetAuthEnabled(true): AuthEnabled = false, want true")
		}
		if err := svc.SetAuthEnabled(false); err != nil {
			t.Fatalf("SetAuthEnabled(false): %v", err)
		}
		if repo.kv[setting.KeyUserAuthEnabled] != "off" {
			t.Fatalf("SetAuthEnabled(false) wrote %q, want off", repo.kv[setting.KeyUserAuthEnabled])
		}
		if on, _ := svc.AuthEnabled(); on {
			t.Fatal("after SetAuthEnabled(false): AuthEnabled = true, want false")
		}
	})
}
