package docs

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Service 用例服务（无 fiber 依赖；MCP 与 admin 接口层共用）。
type Service struct {
	repo Repository
	now  func() time.Time // 可注入时钟（测试）
}

// NewService 构造用例服务。
func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// SetClock 注入时钟（测试用）。
func (s *Service) SetClock(now func() time.Time) { s.now = now }

// CreateInput doc.create 入参（unit-7 §4：path 全路径必填，父 section 隐式补齐）。
type CreateInput struct {
	Title   string
	Path    string
	Kind    Kind    // 缺省 doc
	Content *string // nil = 未提供
	Summary *string
	Tags    []string
	Order   *int
	Status  Status // 缺省 published
}

// Create 创建节点；中间缺失的 section 自动补齐（agent 友好）。
// 返回最终节点（含 ID 与时间戳）。
func (s *Service) Create(in CreateInput) (*Doc, error) {
	segments := SplitPath(in.Path)
	if len(segments) == 0 {
		return nil, ErrInvalidSlug
	}
	if len(segments) > MaxDepth {
		return nil, ErrTooDeep
	}
	kind := in.Kind
	if kind == "" {
		kind = KindDoc
	}
	if err := ValidateKind(kind); err != nil {
		return nil, err
	}
	status := in.Status
	if status == "" {
		status = StatusPublished
	}
	if err := ValidateStatus(status); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Title) == "" {
		return nil, errors.New("title is required")
	}
	if len(in.Title) > MaxTitleLen {
		return nil, fmt.Errorf("title too long (max %d)", MaxTitleLen)
	}
	if in.Summary != nil && len(*in.Summary) > 200 {
		return nil, fmt.Errorf("summary too long (max 200)")
	}
	if err := ValidateTags(in.Tags); err != nil {
		return nil, err
	}
	for _, slug := range segments {
		if err := ValidateSlug(slug); err != nil {
			return nil, fmt.Errorf("path 段 %q: %w", slug, err)
		}
	}

	// 隐式补父：逐级确保 section 存在（已存在但为 doc → 报冲突）。
	parentPath := ""
	for _, slug := range segments[:len(segments)-1] {
		path := JoinPath(parentPath, slug)
		existing, err := s.repo.GetByPath(path)
		if errors.Is(err, ErrNotFound) {
			created, err := s.insert(&Doc{
				ParentID: 0, // 由 insert 经 parentPath 反查
				Slug:     slug,
				Path:     path,
				Kind:     KindSection,
				Title:    slug,
				Status:   StatusPublished,
			}, parentPath)
			if err != nil {
				return nil, err
			}
			parentPath = created.Path
			continue
		}
		if err != nil {
			return nil, err
		}
		if !existing.IsSection() {
			return nil, fmt.Errorf("%w: %s 已是文档，不能作为父级", ErrInvalidKind, path)
		}
		parentPath = existing.Path
	}

	slug := segments[len(segments)-1]
	doc := &Doc{
		Slug:   slug,
		Path:   JoinPath(parentPath, slug),
		Kind:   kind,
		Title:  in.Title,
		Status: status,
		Tags:   in.Tags,
	}
	if in.Content != nil {
		doc.Content = *in.Content
	}
	if in.Summary != nil {
		doc.Summary = *in.Summary
	}
	if in.Order != nil {
		doc.SortOrder = *in.Order
	}
	created, err := s.insert(doc, parentPath)
	if err != nil {
		return nil, err
	}
	return created, nil
}

// insert 补全 ParentID（按父路径反查）与时间戳后落库。
func (s *Service) insert(doc *Doc, parentPath string) (*Doc, error) {
	if parentPath != "" {
		parent, err := s.repo.GetByPath(parentPath)
		if err != nil {
			return nil, err
		}
		doc.ParentID = parent.ID
	}
	doc.CreatedAt = s.now()
	doc.UpdatedAt = doc.CreatedAt
	if err := s.repo.Create(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// GetResult doc.get 结果：节点 + 直接子节点。
type GetResult struct {
	Doc      *Doc
	Children []*Doc
}

// Get 按路径取节点与直接子节点。
func (s *Service) Get(path string) (*GetResult, error) {
	doc, err := s.repo.GetByPath(strings.Trim(path, "/"))
	if err != nil {
		return nil, err
	}
	children, err := s.repo.ListChildren(doc.ID)
	if err != nil {
		return nil, err
	}
	return &GetResult{Doc: doc, Children: children}, nil
}

// ListInput doc.list 入参。
type ListInput struct {
	Path      string // 缺省 = 根
	Recursive bool
	Limit     int // <=0 = 100
	Offset    int
}

// List 列出子树（recursive）或直接子节点；MCP 侧全量含 draft。
func (s *Service) List(in ListInput) ([]*Doc, int, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 100
	}
	var docs []*Doc
	var err error
	if in.Recursive {
		docs, err = s.repo.ListAll()
		if err != nil {
			return nil, 0, err
		}
		prefix := strings.Trim(in.Path, "/")
		filtered := make([]*Doc, 0, len(docs))
		for _, d := range docs {
			if prefix == "" || d.Path == prefix || strings.HasPrefix(d.Path, prefix+"/") {
				filtered = append(filtered, d)
			}
		}
		docs = filtered
	} else {
		parentID := int64(0)
		if in.Path != "" {
			parent, err := s.repo.GetByPath(strings.Trim(in.Path, "/"))
			if err != nil {
				return nil, 0, err
			}
			parentID = parent.ID
		}
		docs, err = s.repo.ListChildren(parentID)
		if err != nil {
			return nil, 0, err
		}
	}
	total := len(docs)
	if in.Offset > 0 {
		if in.Offset >= len(docs) {
			return []*Doc{}, total, nil
		}
		docs = docs[in.Offset:]
	}
	if len(docs) > limit {
		docs = docs[:limit]
	}
	return docs, total, nil
}

// TreeNode 导航树节点。
type TreeNode struct {
	Doc      *TreeNodeDoc `json:"doc"`
	Children []*TreeNode  `json:"children"`
}

// TreeNodeDoc 树节点携带的文档摘要。
type TreeNodeDoc struct {
	ID        int64     `json:"id"`
	Path      string    `json:"path"`
	Slug      string    `json:"slug"`
	Kind      Kind      `json:"kind"`
	Title     string    `json:"title"`
	Summary   string    `json:"summary"`
	SortOrder int       `json:"sortOrder"`
	Status    Status    `json:"status"`
	Tags      []string  `json:"tags"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Tree 构建全量导航树。onlyPublished=true 时剔除 draft（页面用）；MCP 全量。
func (s *Service) Tree(onlyPublished bool) ([]*TreeNode, error) {
	all, err := s.repo.ListAll()
	if err != nil {
		return nil, err
	}
	nodes := make(map[int64]*TreeNode, len(all))
	var roots []*TreeNode
	for _, d := range all {
		if onlyPublished && d.Status != StatusPublished {
			continue
		}
		nodes[d.ID] = &TreeNode{Doc: &TreeNodeDoc{
			ID: d.ID, Path: d.Path, Slug: d.Slug, Kind: d.Kind, Title: d.Title,
			Summary: d.Summary, SortOrder: d.SortOrder, Status: d.Status,
			Tags: d.Tags, UpdatedAt: d.UpdatedAt,
		}}
	}
	for _, d := range all {
		node, ok := nodes[d.ID]
		if !ok {
			continue // draft 被剔除
		}
		if parent, ok := nodes[d.ParentID]; ok && d.ParentID != 0 {
			parent.Children = append(parent.Children, node)
		} else {
			roots = append(roots, node)
		}
	}
	sortTrees(roots)
	return roots, nil
}

// sortTrees 递归排序：sort_order 升序 → 字典序。
func sortTrees(nodes []*TreeNode) {
	sort.Slice(nodes, func(i, j int) bool {
		a, b := nodes[i].Doc, nodes[j].Doc
		if a.SortOrder != b.SortOrder {
			return a.SortOrder < b.SortOrder
		}
		return a.Slug < b.Slug
	})
	for _, n := range nodes {
		sortTrees(n.Children)
	}
}

// UpdateInput doc.update 入参：指针 = 提供该字段（部分更新，author.update 先例）。
type UpdateInput struct {
	Path    string
	Title   *string
	Content *string
	Summary *string
	Tags    []string // nil = 未提供（区别于清空 []）
	Order   *int
	Status  *Status
}

// Update 部分更新节点（不改 path/slug/kind——结构演进走 Move，M5）。
func (s *Service) Update(in UpdateInput) (*Doc, error) {
	doc, err := s.repo.GetByPath(strings.Trim(in.Path, "/"))
	if err != nil {
		return nil, err
	}
	if in.Title != nil {
		if strings.TrimSpace(*in.Title) == "" {
			return nil, errors.New("title is required")
		}
		if len(*in.Title) > MaxTitleLen {
			return nil, fmt.Errorf("title too long (max %d)", MaxTitleLen)
		}
		doc.Title = *in.Title
	}
	if in.Content != nil {
		doc.Content = *in.Content
	}
	if in.Summary != nil {
		if len(*in.Summary) > 200 {
			return nil, fmt.Errorf("summary too long (max 200)")
		}
		doc.Summary = *in.Summary
	}
	if in.Tags != nil {
		if err := ValidateTags(in.Tags); err != nil {
			return nil, err
		}
		doc.Tags = in.Tags
	}
	if in.Order != nil {
		doc.SortOrder = *in.Order
	}
	if in.Status != nil {
		if err := ValidateStatus(*in.Status); err != nil {
			return nil, err
		}
		doc.Status = *in.Status
	}
	doc.UpdatedAt = s.now()
	if err := s.repo.Update(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// DeleteInput doc.delete 入参。
type DeleteInput struct {
	Path      string
	Recursive bool
}

// Delete 删除节点：section 有子节点且未 recursive → ErrHasChildren；
// recursive 级联删除子树（收集前缀匹配节点逐一删除，Phase 1 无事务包裹——单作者场景可接受，
// M5 Move 落地时一并升级事务化仓储方法）。
func (s *Service) Delete(in DeleteInput) error {
	path := strings.Trim(in.Path, "/")
	doc, err := s.repo.GetByPath(path)
	if err != nil {
		return err
	}
	children, err := s.repo.CountChildren(doc.ID)
	if err != nil {
		return err
	}
	if children > 0 && !in.Recursive {
		return ErrHasChildren
	}
	if children > 0 && in.Recursive {
		all, err := s.repo.ListAll()
		if err != nil {
			return err
		}
		// 深路径先删（子在前），避免父删除后子成孤儿引用混乱。
		victims := make([]*Doc, 0)
		for _, d := range all {
			if d.Path == doc.Path || strings.HasPrefix(d.Path, doc.Path+"/") {
				victims = append(victims, d)
			}
		}
		sort.Slice(victims, func(i, j int) bool { return Depth(victims[i].Path) > Depth(victims[j].Path) })
		for _, v := range victims {
			if err := s.repo.Delete(v.ID); err != nil {
				return err
			}
		}
		return nil
	}
	return s.repo.Delete(doc.ID)
}
