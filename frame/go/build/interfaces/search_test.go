package interfaces

import (
	"context"
	"errors"
	"testing"

	"ven_hybird/build/application/postapp"
	"ven_hybird/build/domain/post"
	"ven_hybird/build/plugin"
)

func newSearchTestAggregator() *searchAggregator {
	repo := &fakePostRepo{posts: map[int64]*post.Post{
		1: {ID: 1, Title: "Go 并发实战", Content: "goroutine 与 channel", Category: "技术"},
		2: {ID: 2, Title: "TypeScript 类型体操", Content: "conditional types", Category: "技术"},
	}}
	return NewSearchAggregator(postapp.NewService(repo))
}

func TestSearchAggregator_BlogOnlyBitExact(t *testing.T) {
	agg := newSearchTestAggregator()
	// 无插件注册时 scope=all 与 scope=blog 结果一致（bit-exact 承诺）。
	allViews, allGroups, err := agg.search(context.Background(), "Go", "all", 10)
	if err != nil {
		t.Fatalf("scope=all: %v", err)
	}
	blogViews, blogGroups, err := agg.search(context.Background(), "Go", "blog", 10)
	if err != nil {
		t.Fatalf("scope=blog: %v", err)
	}
	if len(allViews) != len(blogViews) || len(allGroups) != 0 || len(blogGroups) != 0 {
		t.Fatalf("无插件时 all 与 blog 应等价且无插件组：%d/%d %d/%d",
			len(allViews), len(blogViews), len(allGroups), len(blogGroups))
	}
}

func TestSearchAggregator_RegisterProvider(t *testing.T) {
	agg := newSearchTestAggregator()
	if err := agg.RegisterProvider(&fakeProvider{name: "docs", hits: []plugin.SearchHit{{Title: "Attention 机制"}}}); err != nil {
		t.Fatalf("RegisterProvider: %v", err)
	}
	if err := agg.RegisterProvider(&fakeProvider{name: "docs"}); err == nil {
		t.Fatal("重名应拒绝")
	}
	if err := agg.RegisterProvider(&fakeProvider{name: "blog"}); err == nil {
		t.Fatal("保留名 blog 应拒绝")
	}
	if err := agg.RegisterProvider(&fakeProvider{name: "Bad_Name"}); err == nil {
		t.Fatal("非法名应拒绝")
	}
	if err := agg.RegisterProvider(nil); err == nil {
		t.Fatal("nil 应拒绝")
	}
}

func TestSearchAggregator_ScopeAllMergesPlugins(t *testing.T) {
	agg := newSearchTestAggregator()
	if err := agg.RegisterProvider(&fakeProvider{name: "docs", hits: []plugin.SearchHit{
		{Title: "Attention", Summary: "注意力机制"},
	}}); err != nil {
		t.Fatalf("RegisterProvider: %v", err)
	}
	_, groups, err := agg.search(context.Background(), "q", "all", 10)
	if err != nil {
		t.Fatalf("scope=all: %v", err)
	}
	if len(groups) != 1 || groups[0].Provider != "docs" || len(groups[0].Hits) != 1 {
		t.Fatalf("插件组应合并，得 %+v", groups)
	}
	// scope=blog 时不带插件组。
	_, groups, err = agg.search(context.Background(), "q", "blog", 10)
	if err != nil || len(groups) != 0 {
		t.Fatalf("scope=blog 不应含插件组：%v %v", groups, err)
	}
	// scope=docs 单源。
	_, groups, err = agg.search(context.Background(), "q", "docs", 10)
	if err != nil || len(groups) != 1 || groups[0].Provider != "docs" {
		t.Fatalf("scope=docs 应单源，得 %+v %v", groups, err)
	}
	// scope 未知。
	if _, _, err := agg.search(context.Background(), "q", "ghost", 10); err == nil {
		t.Fatal("未知 scope 应报错")
	}
}

func TestSearchAggregator_ProviderErrorDegrades(t *testing.T) {
	agg := newSearchTestAggregator()
	if err := agg.RegisterProvider(&fakeProvider{name: "boom", err: errors.New("down")}); err != nil {
		t.Fatalf("RegisterProvider: %v", err)
	}
	views, groups, err := agg.search(context.Background(), "Go", "all", 10)
	if err != nil {
		t.Fatalf("单源失败不应拖垮整页: %v", err)
	}
	if len(groups) != 1 || len(groups[0].Hits) != 0 {
		t.Fatalf("失败源应降级空组，得 %+v", groups)
	}
	if len(views) == 0 {
		t.Fatal("blog 源应正常")
	}
}

// fakeProvider 测试用搜索 provider。
type fakeProvider struct {
	name string
	hits []plugin.SearchHit
	err  error
}

func (f *fakeProvider) Name() string { return f.name }
func (f *fakeProvider) Search(ctx context.Context, q string, limit int) ([]plugin.SearchHit, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.hits, nil
}
