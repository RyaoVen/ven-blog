package build

import (
	"testing"

	"ven_hybird/build/application/settingsapp"
	"ven_hybird/build/domain/setting"
)

// fakeSettingsRepo 内存实现 setting.Repository（siteURLOf 解析单测用，
// 语义与真实仓储一致：Get 不存在返回空串与 nil，Set 为 upsert）。
type fakeSettingsRepo struct {
	values map[string]string
}

func newFakeSettingsRepo() *fakeSettingsRepo {
	return &fakeSettingsRepo{values: map[string]string{}}
}

func (f *fakeSettingsRepo) Get(key string) (string, error) {
	return f.values[key], nil
}

func (f *fakeSettingsRepo) Set(key, value string) error {
	f.values[key] = value
	return nil
}

// 接口合规断言。
var _ setting.Repository = (*fakeSettingsRepo)(nil)

// 设置键 site_url 非空时优先（env 配置了也以设置值为准）。
func TestSiteURLOfSettingsFirst(t *testing.T) {
	t.Setenv("BLOG_SITE_URL", "https://env.example.com")
	repo := newFakeSettingsRepo()
	repo.values[setting.KeySiteURL] = "https://settings.example.com"
	svc := settingsapp.NewService(repo)
	if got := siteURLOf(svc); got != "https://settings.example.com" {
		t.Fatalf("siteURLOf = %q, want settings value", got)
	}
}

// 设置键为空时回退 env BLOG_SITE_URL。
func TestSiteURLOfEnvFallback(t *testing.T) {
	t.Setenv("BLOG_SITE_URL", "https://env.example.com")
	svc := settingsapp.NewService(newFakeSettingsRepo())
	if got := siteURLOf(svc); got != "https://env.example.com" {
		t.Fatalf("siteURLOf = %q, want env value", got)
	}
}

// 设置键与 env 都为空时回退本地开发默认地址。
func TestSiteURLOfDefault(t *testing.T) {
	t.Setenv("BLOG_SITE_URL", "")
	svc := settingsapp.NewService(newFakeSettingsRepo())
	if got := siteURLOf(svc); got != defaultSiteURL {
		t.Fatalf("siteURLOf = %q, want default %q", got, defaultSiteURL)
	}
}
