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
	Since     *time.Time // 非空时按 updated_at >= since 过滤（增量拉取，M5）
}

// List 列出子树（recursive）或直接子节点；MCP 侧全量含 draft。
func (s *Service) List(in ListInput) ([]*Doc, int, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 100
	}
	var docs []*Doc
	var err error
	if in.Since != nil {
		// 增量模式：忽略 path/recursive 语义，直接按时间窗拉取。
		docs, err = s.repo.ListUpdatedSince(*in.Since)
		if err != nil {
			return nil, 0, err
		}
		total := len(docs)
		if in.Offset > 0 && in.Offset < len(docs) {
			docs = docs[in.Offset:]
		}
		if len(docs) > limit {
			docs = docs[:limit]
		}
		return docs, total, nil
	}
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
		tags := d.Tags
		if tags == nil {
			tags = []string{}
		}
		nodes[d.ID] = &TreeNode{Doc: &TreeNodeDoc{
			ID: d.ID, Path: d.Path, Slug: d.Slug, Kind: d.Kind, Title: d.Title,
			Summary: d.Summary, SortOrder: d.SortOrder, Status: d.Status,
			Tags: tags, UpdatedAt: d.UpdatedAt,
		}, Children: []*TreeNode{}}
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

// publishedPaths 返回 published 节点的扁平序（DFS：sort_order → slug；根级先、子随后），
// 供上一页/下一页计算。
func (s *Service) publishedPaths() ([]*Doc, error) {
	all, err := s.repo.ListAll()
	if err != nil {
		return nil, err
	}
	children := make(map[int64][]*Doc)
	roots := make([]*Doc, 0)
	for _, d := range all {
		if d.Status != StatusPublished {
			continue
		}
		if d.ParentID == 0 {
			roots = append(roots, d)
		} else {
			children[d.ParentID] = append(children[d.ParentID], d)
		}
	}
	sortDocList(roots)
	for _, list := range children {
		sortDocList(list)
	}
	out := make([]*Doc, 0, len(all))
	var walk func(list []*Doc)
	walk = func(list []*Doc) {
		for _, d := range list {
			out = append(out, d)
			if kids := children[d.ID]; len(kids) > 0 {
				walk(kids)
			}
		}
	}
	walk(roots)
	return out, nil
}

// sortDocList 排序：sort_order 升序 → slug 字典序。
func sortDocList(list []*Doc) {
	sort.Slice(list, func(i, j int) bool {
		if list[i].SortOrder != list[j].SortOrder {
			return list[i].SortOrder < list[j].SortOrder
		}
		return list[i].Slug < list[j].Slug
	})
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

// GetByID 按主键取节点。
func (s *Service) GetByID(id int64) (*Doc, error) {
	return s.repo.GetByID(id)
}

// UpdateByID 按主键部分更新（admin API 用；字段语义同 Update）。
func (s *Service) UpdateByID(id int64, in UpdateInput) (*Doc, error) {
	doc, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	in.Path = doc.Path
	return s.Update(in)
}

// DeleteByID 按主键删除（admin API 用）。
func (s *Service) DeleteByID(id int64, recursive bool) error {
	doc, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	return s.Delete(DeleteInput{Path: doc.Path, Recursive: recursive})
}

// MoveInput doc.move 入参（unit-7 §5）：newParent/newSlug/order 至少一项。
type MoveInput struct {
	Path      string
	NewParent string // 新父全路径；"" = 保持
	NewSlug   string // 新 slug；"" = 保持
	Order     *int
}

// Move 移动/改名/重排节点并级联更新子树 path。
// 规则：新位置不得是自身或自身后代（防环）；新 path 冲突拒绝；树深上限校验。
func (s *Service) Move(in MoveInput) (*Doc, error) {
	doc, err := s.repo.GetByPath(strings.Trim(in.Path, "/"))
	if err != nil {
		return nil, err
	}
	if in.NewParent == "" && in.NewSlug == "" && in.Order == nil {
		return nil, errors.New("newParent/newSlug/order 至少提供一项")
	}
	newParentPath := ""
	if in.NewParent != "" {
		newParentPath = strings.Trim(in.NewParent, "/")
		if newParentPath == doc.Path || strings.HasPrefix(newParentPath, doc.Path+"/") {
			return nil, fmt.Errorf("%w: 不能移动到自身或其后代之下", ErrInvalidKind)
		}
		parent, err := s.repo.GetByPath(newParentPath)
		if err != nil {
			return nil, err
		}
		if !parent.IsSection() {
			return nil, fmt.Errorf("%w: %s 不是目录", ErrInvalidKind, newParentPath)
		}
	}
	slug := doc.Slug
	if in.NewSlug != "" {
		if err := ValidateSlug(in.NewSlug); err != nil {
			return nil, err
		}
		slug = in.NewSlug
	}
	newParentID := doc.ParentID
	if in.NewParent != "" {
		parent, err := s.repo.GetByPath(newParentPath)
		if err != nil {
			return nil, err
		}
		newParentID = parent.ID
	}
	newParentPathStr := newParentPath
	if in.NewParent == "" {
		// 保持原父：从旧 path 去掉尾段。
		segs := SplitPath(doc.Path)
		if len(segs) > 1 {
			newParentPathStr = strings.Join(segs[:len(segs)-1], "/")
		} else {
			newParentPathStr = ""
		}
	}
	newPath := JoinPath(newParentPathStr, slug)
	// 深度校验：自身 + 后代层数。
	if in.NewParent != "" || in.NewSlug != "" {
		all, err := s.repo.ListAll()
		if err != nil {
			return nil, err
		}
		maxDescDepth := 0
		for _, d := range all {
			if strings.HasPrefix(d.Path, doc.Path+"/") {
				if n := Depth(d.Path) - Depth(doc.Path); n > maxDescDepth {
					maxDescDepth = n
				}
			}
		}
		if Depth(newPath)+maxDescDepth > MaxDepth {
			return nil, ErrTooDeep
		}
		// 冲突校验（目标 path 被占）。
		if _, err := s.repo.GetByPath(newPath); err == nil {
			return nil, ErrDuplicatePath
		}
	}
	now := s.now()
	oldPrefix := doc.Path + "/"
	newPrefix := newPath + "/"
	if err := s.repo.UpdatePath(doc.ID, newParentID, slug, newPath, now); err != nil {
		return nil, err
	}
	if newPath != doc.Path {
		if err := s.repo.RenameDescendants(oldPrefix, newPrefix, now); err != nil {
			return nil, err
		}
	}
	if in.Order != nil {
		doc.SortOrder = *in.Order
		doc.UpdatedAt = now
		if err := s.repo.Update(doc); err != nil {
			return nil, err
		}
	}
	return s.repo.GetByPath(newPath)
}

// ImportFile doc.import 单文件（path + markdown 全文，可含 frontmatter）。
type ImportFile struct {
	Path    string
	Content string
}

// ImportResult 导入结果（MCP doc.import 返回；webhook 汇总事件同源）。
type ImportResult struct {
	Created int
	Updated int
	Failed  []ImportFailure
}

// ImportFailure 单文件失败明细。
type ImportFailure struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

// Import 批量导入（upsert：path 存在则部分更新 title/summary/content/tags/order/status，否则创建）。
// 隐式补父与 doc.create 同语义；单文件失败不中断批次。
func (s *Service) Import(files []ImportFile) ImportResult {
	result := ImportResult{}
	for _, f := range files {
		created, err := s.importOne(f)
		if err != nil {
			result.Failed = append(result.Failed, ImportFailure{Path: f.Path, Error: err.Error()})
			continue
		}
		if created {
			result.Created++
		} else {
			result.Updated++
		}
	}
	return result
}

// importOne 单文件 upsert；返回是否为新建。
func (s *Service) importOne(f ImportFile) (bool, error) {
	fm, body, err := parseFrontmatter(f.Content)
	if err != nil {
		return false, err
	}
	title := fm.Title
	if title == "" {
		title = lastSegment(f.Path)
	}
	status := fm.Status
	if status == "" {
		status = StatusPublished
	}
	if existing, err := s.repo.GetByPath(strings.Trim(f.Path, "/")); err == nil {
		in := UpdateInput{
			Path:    existing.Path,
			Title:   &title,
			Content: &body,
			Summary: &fm.Summary,
			Tags:    fm.Tags,
			Status:  &status,
		}
		if fm.Order != nil {
			in.Order = fm.Order
		}
		if _, err := s.Update(in); err != nil {
			return false, err
		}
		return false, nil
	}
	if len(SplitPath(f.Path)) == 0 {
		return false, ErrInvalidSlug
	}
	if _, err := s.Create(CreateInput{
		Title:   title,
		Path:    f.Path,
		Content: &body,
		Summary: &fm.Summary,
		Tags:    fm.Tags,
		Status:  status,
	}); err != nil {
		return false, err
	}
	return true, nil
}

// lastSegment 取路径尾段。
func lastSegment(path string) string {
	segs := SplitPath(path)
	if len(segs) == 0 {
		return ""
	}
	return segs[len(segs)-1]
}

// ExportFile doc.export 单文件。
type ExportFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// Export 按子树导出 markdown 包（frontmatter 还原；path 缺省 = 全站）。
func (s *Service) Export(path string) ([]ExportFile, error) {
	all, err := s.repo.ListAll()
	if err != nil {
		return nil, err
	}
	prefix := strings.Trim(path, "/")
	files := make([]ExportFile, 0, len(all))
	for _, d := range all {
		if prefix != "" && d.Path != prefix && !strings.HasPrefix(d.Path, prefix+"/") {
			continue
		}
		files = append(files, ExportFile{Path: d.Path, Content: buildFrontmatter(d, true)})
	}
	return files, nil
}

// DeleteInput doc.delete 入参。
type DeleteInput struct {
	Path      string
	Recursive bool
}

// Delete 删除节点：section 有子节点且未 recursive → ErrHasChildren；
// recursive 走仓储事务（自身 + 前缀后代原子删除）。
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
		return s.repo.DeleteSubtree(doc.Path)
	}
	return s.repo.Delete(doc.ID)
}
