// 搜索页注册：按查询参数 q 检索文章（标题/正文），scope 支持插件贡献结果源（unit-6 §5.4）。
package interfaces

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"ven_hybird/build/application/postapp"
	"ven_hybird/build/plugin"
	"ven_hybird/hybrid"
)

// blogProviderName 内置 provider 名（保留名，插件不可注册）。
const blogProviderName = "blog"

// searchAggregator 搜索聚合器：内置 blog provider（文章检索）+ 插件注册的 provider。
// 实现 plugin.SearchRegistry。scope 取值 = "all"（缺省，合并全部）| "blog" | 各 provider 名。
// 兼容性硬约束：无插件注册时，scope=all 与 scope=blog 的 results 与改造前 bit-exact；
// 插件结果一律放 pluginResults 新字段，旧前端忽略即无感。
type searchAggregator struct {
	mu        sync.RWMutex
	providers map[string]plugin.SearchProvider
	posts     *postapp.Service
}

// NewSearchAggregator 创建聚合器（blog provider 内置，插件不可覆盖）。
func NewSearchAggregator(posts *postapp.Service) *searchAggregator {
	return &searchAggregator{providers: make(map[string]plugin.SearchProvider), posts: posts}
}

// RegisterProvider 注册插件搜索 provider（实现 plugin.SearchRegistry）。
// name 须 kebab 且非保留名；重名拒绝（fail-fast，启动期调用）。
func (s *searchAggregator) RegisterProvider(p plugin.SearchProvider) error {
	if p == nil {
		return fmt.Errorf("search: provider 为空")
	}
	name := p.Name()
	if name == blogProviderName || !isKebabName(name) {
		return fmt.Errorf("search: provider 名 %q 非法（须 kebab 且非保留名 %s）", name, blogProviderName)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, dup := s.providers[name]; dup {
		return fmt.Errorf("search: provider %q 重复注册", name)
	}
	s.providers[name] = p
	return nil
}

// providerNames 返回全部可用 scope 名（含 blog）。
func (s *searchAggregator) providerNames() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := []string{blogProviderName}
	for name := range s.providers {
		names = append(names, name)
	}
	return names
}

// searchHitGroup 插件源结果组（initialState.pluginResults 项）。
type searchHitGroup struct {
	Provider string             `json:"provider"`
	Hits     []plugin.SearchHit `json:"hits"`
}

// search 执行检索：scope=all/缺省 → blog 全量 + 逐插件源；scope=<name> → 单源。
// 单源失败降级为该源空结果（不拖垮整页）。
func (s *searchAggregator) search(ctx context.Context, q, scope string, limit int) (postViews []PostView, pluginGroups []searchHitGroup, err error) {
	if scope == "" || scope == "all" {
		views, err := s.searchBlog(q, limit)
		if err != nil {
			return nil, nil, err
		}
		return views, s.collectPlugins(ctx, q, limit), nil
	}
	if scope == blogProviderName {
		views, err := s.searchBlog(q, limit)
		if err != nil {
			return nil, nil, err
		}
		return views, nil, nil
	}
	s.mu.RLock()
	p, ok := s.providers[scope]
	s.mu.RUnlock()
	if !ok {
		return nil, nil, fmt.Errorf("unknown scope: %s（可用：%s）", scope, strings.Join(s.providerNames(), "/"))
	}
	return []PostView{}, []searchHitGroup{{Provider: scope, Hits: s.queryProvider(ctx, p, q, limit)}}, nil
}

// searchBlog 内置文章检索（保持原 posts.Search 语义）。
func (s *searchAggregator) searchBlog(q string, limit int) ([]PostView, error) {
	results, err := s.posts.Search(q)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return toPostViews(results), nil
}

// collectPlugins 并发查询全部插件源（每源独立超时，失败降级空组），按 provider 名稳定排序。
func (s *searchAggregator) collectPlugins(ctx context.Context, q string, limit int) []searchHitGroup {
	s.mu.RLock()
	providers := make([]plugin.SearchProvider, 0, len(s.providers))
	for _, p := range s.providers {
		providers = append(providers, p)
	}
	s.mu.RUnlock()
	if len(providers) == 0 {
		return nil
	}
	groups := make([]searchHitGroup, len(providers))
	var wg sync.WaitGroup
	for i, p := range providers {
		wg.Add(1)
		go func(i int, p plugin.SearchProvider) {
			defer wg.Done()
			groups[i] = searchHitGroup{Provider: p.Name(), Hits: s.queryProvider(ctx, p, q, limit)}
		}(i, p)
	}
	wg.Wait()
	sort.Slice(groups, func(i, j int) bool { return groups[i].Provider < groups[j].Provider })
	return groups
}

// queryProvider 单源查询（2s 超时；失败记日志返回空）。
func (s *searchAggregator) queryProvider(ctx context.Context, p plugin.SearchProvider, q string, limit int) []plugin.SearchHit {
	cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	hits, err := p.Search(cctx, q, limit)
	if err != nil {
		log.Printf("search: provider %s 查询失败: %v", p.Name(), err)
		return nil
	}
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

// isKebabName kebab-case 校验（与 plugin 包同规则，避免反向依赖）。
func isKebabName(s string) bool {
	if s == "" {
		return false
	}
	prevHyphen := true
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			prevHyphen = false
		case r == '-':
			if prevHyphen {
				return false
			}
			prevHyphen = true
		default:
			return false
		}
	}
	return !prevHyphen
}

// RegisterSearch 注册搜索页（公开动态页；空关键词由应用层归一为空结果）。
// scope 查询参数：all（缺省）| blog | 插件 provider 名。
func RegisterSearch(a *hybrid.App, posts *postapp.Service, aggregator *searchAggregator) error {
	return a.Page("/search", nil, func(c *hybrid.PageCtx) error {
		q := strings.TrimSpace(c.Query("q"))
		scope := strings.TrimSpace(c.Query("scope"))
		results, pluginGroups, err := aggregator.search(context.Background(), q, scope, 0)
		if err != nil {
			return err
		}
		return c.JSON(map[string]any{
			"q":             q,
			"scope":         scope,
			"results":       results,
			"pluginResults": pluginGroups,
		})
	})
}
