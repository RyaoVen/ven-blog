package docs

import (
	"context"
	"sort"
	"strings"

	"ven_hybird/build/plugin"
)

// DocSearchProvider docs 搜索贡献源（unit-6 §5.4）：标题/摘要/正文子串匹配，
// 命中按更新时间倒序，limit 截断。仅匹配 published 节点。
type DocSearchProvider struct {
	svc *Service
}

// NewDocSearchProvider 构造 provider。
func NewDocSearchProvider(svc *Service) *DocSearchProvider { return &DocSearchProvider{svc: svc} }

// Name provider 名 = scope 取值 = 插件名。
func (p *DocSearchProvider) Name() string { return "docs" }

// Search 检索实现（ctx 超时由聚合器统一 2s 控制；遍历全量内存匹配）。
func (p *DocSearchProvider) Search(ctx context.Context, q string, limit int) ([]plugin.SearchHit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	keyword := strings.ToLower(strings.TrimSpace(q))
	if keyword == "" {
		return nil, nil
	}
	all, err := p.svc.repo.ListAll()
	if err != nil {
		return nil, err
	}
	hits := make([]plugin.SearchHit, 0, 8)
	for _, d := range all {
		if d.Status != StatusPublished {
			continue
		}
		if !strings.Contains(strings.ToLower(d.Title), keyword) &&
			!strings.Contains(strings.ToLower(d.Summary), keyword) &&
			!strings.Contains(strings.ToLower(d.Content), keyword) {
			continue
		}
		summary := d.Summary
		if summary == "" {
			summary = excerpt(d.Content, keyword)
		}
		hits = append(hits, plugin.SearchHit{
			Title:     d.Title,
			Summary:   summary,
			URL:       "/docs/" + d.Path,
			UpdatedAt: d.UpdatedAt,
		})
		if limit > 0 && len(hits) >= limit {
			break
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].UpdatedAt.After(hits[j].UpdatedAt) })
	return hits, nil
}

// excerpt 提取关键词命中的上下文片段（约 80 字符）。
func excerpt(content, keyword string) string {
	idx := strings.Index(strings.ToLower(content), keyword)
	if idx < 0 {
		if len(content) > 80 {
			return content[:80]
		}
		return content
	}
	start := idx - 20
	if start < 0 {
		start = 0
	}
	end := idx + len(keyword) + 60
	if end > len(content) {
		end = len(content)
	}
	return content[start:end]
}
