package postapp

import (
	"errors"
	"testing"

	"ven_hybird/build/domain/post"
)

// fakeRepo 内存实现 post.Repository（测试用，仅覆盖本包用到的行为）。
type fakeRepo struct {
	byID map[int64]*post.Post
	next int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byID: map[int64]*post.Post{}, next: 1}
}

func (f *fakeRepo) add(p *post.Post) *post.Post {
	p.ID = f.next
	f.next++
	f.byID[p.ID] = p
	return p
}

func (f *fakeRepo) ListPaged(category string, page, pageSize int) ([]*post.Post, int, error) {
	out := make([]*post.Post, 0, len(f.byID))
	for _, p := range f.byID {
		if category == "" || p.Category == category {
			out = append(out, p)
		}
	}
	return out, len(out), nil
}

func (f *fakeRepo) ListByAuthor(authorID int64) ([]*post.Post, error) { return nil, nil }
func (f *fakeRepo) Get(id int64) (*post.Post, error)                 { return nil, nil }
func (f *fakeRepo) Search(query string, limit int) ([]*post.Post, error) {
	return nil, nil
}
func (f *fakeRepo) Create(p *post.Post) error { f.add(p); return nil }
func (f *fakeRepo) Update(p *post.Post) error { return nil }
func (f *fakeRepo) Delete(id int64) error     { return nil }
func (f *fakeRepo) AllTags() ([]string, error) {
	return nil, nil
}
func (f *fakeRepo) Stats() (int, int, error) { return 0, 0, nil }
func (f *fakeRepo) ListFavorites(userID int64) ([]*post.Post, error) {
	return nil, nil
}
func (f *fakeRepo) DailyPublication(days int) ([]post.DayPublication, error) {
	return nil, nil
}
func (f *fakeRepo) CategoryCounts() ([]post.CategoryCount, error) { return nil, nil }
func (f *fakeRepo) CountByCategory(category string) (int, error) { return 0, nil }
func (f *fakeRepo) UpdateCategory(from, to string) error         { return nil }

func (f *fakeRepo) SetPinned(id int64, pinned bool) error {
	p, ok := f.byID[id]
	if !ok {
		return post.ErrNotFound
	}
	p.Pinned = pinned
	return nil
}

func TestSetPinned(t *testing.T) {
	t.Run("pin sets flag and unpin clears", func(t *testing.T) {
		repo := newFakeRepo()
		svc := NewService(repo)
		p := repo.add(&post.Post{Title: "a"})
		if err := svc.SetPinned(p.ID, true); err != nil {
			t.Fatalf("SetPinned(true): %v", err)
		}
		if !repo.byID[p.ID].Pinned {
			t.Fatal("after pin: Pinned = false, want true")
		}
		if err := svc.SetPinned(p.ID, false); err != nil {
			t.Fatalf("SetPinned(false): %v", err)
		}
		if repo.byID[p.ID].Pinned {
			t.Fatal("after unpin: Pinned = true, want false")
		}
	})

	t.Run("not found passes through", func(t *testing.T) {
		svc := NewService(newFakeRepo())
		if err := svc.SetPinned(999, true); !errors.Is(err, post.ErrNotFound) {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	})
}
