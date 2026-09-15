package docs

import (
	"errors"
	"testing"
	"time"
)

// memRepo 内存假仓储（表驱动测试用）。
type memRepo struct {
	byID   map[int64]*Doc
	nextID int64
}

func newMemRepo() *memRepo { return &memRepo{byID: map[int64]*Doc{}, nextID: 1} }

func (r *memRepo) Create(doc *Doc) error {
	for _, d := range r.byID {
		if d.Path == doc.Path {
			return ErrDuplicatePath
		}
	}
	doc.ID = r.nextID
	r.nextID++
	cp := *doc
	r.byID[doc.ID] = &cp
	return nil
}

func (r *memRepo) GetByPath(path string) (*Doc, error) {
	for _, d := range r.byID {
		if d.Path == path {
			cp := *d
			return &cp, nil
		}
	}
	return nil, ErrNotFound
}

func (r *memRepo) GetByID(id int64) (*Doc, error) {
	if d, ok := r.byID[id]; ok {
		cp := *d
		return &cp, nil
	}
	return nil, ErrNotFound
}

func (r *memRepo) ListChildren(parentID int64) ([]*Doc, error) {
	out := []*Doc{}
	for _, d := range r.byID {
		if d.ParentID == parentID {
			cp := *d
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *memRepo) ListAll() ([]*Doc, error) {
	out := []*Doc{}
	for _, d := range r.byID {
		cp := *d
		out = append(out, &cp)
	}
	return out, nil
}

func (r *memRepo) Update(doc *Doc) error {
	if _, ok := r.byID[doc.ID]; !ok {
		return ErrNotFound
	}
	for _, d := range r.byID {
		if d.Path == doc.Path && d.ID != doc.ID {
			return ErrDuplicatePath
		}
	}
	cp := *doc
	r.byID[doc.ID] = &cp
	return nil
}

func (r *memRepo) Delete(id int64) error {
	delete(r.byID, id)
	return nil
}

func (r *memRepo) CountChildren(id int64) (int, error) {
	n := 0
	for _, d := range r.byID {
		if d.ParentID == id {
			n++
		}
	}
	return n, nil
}

func newTestService() *Service {
	svc := NewService(newMemRepo())
	base := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	tick := base
	svc.SetClock(func() time.Time { tick = tick.Add(time.Second); return tick })
	return svc
}

func TestCreate_BasicAndImplicitParents(t *testing.T) {
	svc := newTestService()
	// path=a/b/note：自动补 a、b 两个 section。
	doc, err := svc.Create(CreateInput{Title: "Note", Path: "a/b/note", Content: strPtr("# hi")})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if doc.Kind != KindDoc || doc.Status != StatusPublished || doc.ParentID == 0 {
		t.Fatalf("默认值不对：%+v", doc)
	}
	b, err := svc.Get("a/b")
	if err != nil {
		t.Fatalf("隐式父缺失: %v", err)
	}
	if !b.Doc.IsSection() || b.Doc.Title != "b" {
		t.Fatalf("隐式 section 应以 slug 为标题：%+v", b.Doc)
	}
	if len(b.Children) != 1 || b.Children[0].Path != "a/b/note" {
		t.Fatalf("子节点不对：%+v", b.Children)
	}
}

func TestCreate_DuplicatePath(t *testing.T) {
	svc := newTestService()
	if _, err := svc.Create(CreateInput{Title: "A", Path: "dup"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := svc.Create(CreateInput{Title: "B", Path: "dup"}); !errors.Is(err, ErrDuplicatePath) {
		t.Fatalf("应报重复路径，得 %v", err)
	}
}

func TestCreate_Validation(t *testing.T) {
	svc := newTestService()
	cases := []struct {
		name string
		in   CreateInput
		want error
	}{
		{"空路径", CreateInput{Title: "t", Path: ""}, ErrInvalidSlug},
		{"非法 slug", CreateInput{Title: "t", Path: "Bad_Slug"}, ErrInvalidSlug},
		{"太深", CreateInput{Title: "t", Path: "a/b/c/d/e/f/g/h/i"}, ErrTooDeep},
		{"空标题", CreateInput{Title: "  ", Path: "x"}, nil},
		{"标签超量", CreateInput{Title: "t", Path: "x", Tags: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}}, ErrTooManyTags},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Create(tc.in)
			if tc.want == nil {
				if err == nil {
					t.Fatal("应报错（空标题）")
				}
				return
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("want %v got %v", tc.want, err)
			}
		})
	}
}

func TestCreate_DocCannotParent(t *testing.T) {
	svc := newTestService()
	if _, err := svc.Create(CreateInput{Title: "Doc", Path: "d"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := svc.Create(CreateInput{Title: "Child", Path: "d/child"}); err == nil {
		t.Fatal("文档节点下建子节点应报错")
	}
}

func TestUpdate_PartialSemantics(t *testing.T) {
	svc := newTestService()
	orig, _ := svc.Create(CreateInput{Title: "T", Path: "p", Content: strPtr("old"), Tags: []string{"a"}})
	// 只改 title：content/tags 不动。
	got, err := svc.Update(UpdateInput{Path: "p", Title: strPtr("T2")})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Content != "old" || len(got.Tags) != 1 || got.Tags[0] != "a" {
		t.Fatalf("部分更新应保留未提供字段：%+v", got)
	}
	if got.Title != "T2" {
		t.Fatalf("title 未更新：%+v", got)
	}
	// ID 与原节点一致。
	if got.ID != orig.ID {
		t.Fatal("ID 应一致")
	}
	// Tags 显式清空。
	got, err = svc.Update(UpdateInput{Path: "p", Tags: []string{}})
	if err != nil || got.Tags == nil || len(got.Tags) != 0 {
		t.Fatalf("显式空 tags 应清空：%+v %v", got, err)
	}
}

func TestUpdate_Validation(t *testing.T) {
	svc := newTestService()
	svc.Create(CreateInput{Title: "T", Path: "p"})
	if _, err := svc.Update(UpdateInput{Path: "missing"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("更新不存在应 ErrNotFound，得 %v", err)
	}
	if _, err := svc.Update(UpdateInput{Path: "p", Title: strPtr("")}); err == nil {
		t.Fatal("空标题应拒绝")
	}
	bad := Status("archived")
	if _, err := svc.Update(UpdateInput{Path: "p", Status: &bad}); err == nil {
		t.Fatal("非法状态应拒绝")
	}
}

func TestDelete_RecursiveAndRefuse(t *testing.T) {
	svc := newTestService()
	svc.Create(CreateInput{Title: "Note", Path: "sec/note"})
	if err := svc.Delete(DeleteInput{Path: "sec"}); !errors.Is(err, ErrHasChildren) {
		t.Fatalf("有子未 recursive 应拒绝，得 %v", err)
	}
	if err := svc.Delete(DeleteInput{Path: "sec", Recursive: true}); err != nil {
		t.Fatalf("级联删除: %v", err)
	}
	if _, err := svc.Get("sec/note"); !errors.Is(err, ErrNotFound) {
		t.Fatal("子节点应一并删除")
	}
	if _, err := svc.Get("sec"); !errors.Is(err, ErrNotFound) {
		t.Fatal("父节点应删除")
	}
}

func TestList_ChildrenAndRecursive(t *testing.T) {
	svc := newTestService()
	svc.Create(CreateInput{Title: "N1", Path: "a/n1"})
	svc.Create(CreateInput{Title: "N2", Path: "a/b/n2"})
	svc.Create(CreateInput{Title: "Other", Path: "z"})
	// 直接子节点（path=a）：隐式 section b + 文档 n1。
	kids, total, err := svc.List(ListInput{Path: "a"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 2 || len(kids) != 2 {
		t.Fatalf("直接子节点应 2 个（section b + doc n1），得 %d", total)
	}
	// 递归（path=a）。
	all, total, err := svc.List(ListInput{Path: "a", Recursive: true})
	if err != nil {
		t.Fatalf("List recursive: %v", err)
	}
	if total != 4 || len(all) != 4 {
		t.Fatalf("递归子树应含 a 自身及后代共 4 节点，得 %d", len(all))
	}
	// 根级直接子节点。
	roots, total, err := svc.List(ListInput{})
	if err != nil {
		t.Fatalf("List root: %v", err)
	}
	if total != 3 || len(roots) != 3 { // a、a/b? 不——根级直接子：a、z（b 在 a 下）
		if len(roots) != 2 {
			t.Fatalf("根级应 2 个（a、z），得 %d：%+v", len(roots), roots)
		}
	}
	// limit/offset。
	page, _, err := svc.List(ListInput{Recursive: true, Limit: 2, Offset: 1})
	if err != nil || len(page) != 2 {
		t.Fatalf("分页不对：%d %v", len(page), err)
	}
}

func TestTree_PublishFilterAndOrder(t *testing.T) {
	svc := newTestService()
	draft := StatusDraft
	svc.Create(CreateInput{Title: "Pub1", Path: "s/a"})
	svc.Create(CreateInput{Title: "Hidden", Path: "s/b", Status: draft})
	svc.Create(CreateInput{Title: "Sec", Path: "s", Kind: KindSection})
	tree, err := svc.Tree(true)
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	if len(tree) != 1 || tree[0].Doc.Path != "s" {
		t.Fatalf("根应只有 s，得 %+v", tree)
	}
	if len(tree[0].Children) != 1 || tree[0].Children[0].Doc.Title != "Pub1" {
		t.Fatalf("draft 应被剔除：%+v", tree[0].Children)
	}
	full, err := svc.Tree(false)
	if err != nil {
		t.Fatalf("Tree full: %v", err)
	}
	if len(full[0].Children) != 2 {
		t.Fatalf("全量应含 draft：%+v", full[0].Children)
	}
}

func TestValidateSlug(t *testing.T) {
	valid := []string{"a", "abc-123", "a1-b2", "0"}
	for _, s := range valid {
		if err := ValidateSlug(s); err != nil {
			t.Errorf("%q 应合法: %v", s, err)
		}
	}
	invalid := []string{"", "-a", "a-", "A", "a_b", "a--b", "a b", strings_Repeat("a", 65)}
	for _, s := range invalid {
		if err := ValidateSlug(s); err == nil {
			t.Errorf("%q 应非法", s)
		}
	}
}

func strings_Repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

func strPtr(s string) *string { return &s }
