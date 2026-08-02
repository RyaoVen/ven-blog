package momentapp

import (
	"errors"
	"testing"

	"ven_hybird/build/domain/moment"
)

// fakeRepo 内存实现 moment.Repository（测试用，仅覆盖本包用到的行为）。
type fakeRepo struct {
	byID map[int64]*moment.Moment
	next int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byID: map[int64]*moment.Moment{}, next: 1}
}

func (f *fakeRepo) add(m *moment.Moment) *moment.Moment {
	m.ID = f.next
	f.next++
	f.byID[m.ID] = m
	return m
}

func (f *fakeRepo) List(limit int) ([]*moment.Moment, error) {
	out := make([]*moment.Moment, 0, len(f.byID))
	for _, m := range f.byID {
		out = append(out, m)
	}
	return out, nil
}

func (f *fakeRepo) Create(m *moment.Moment) error { f.add(m); return nil }
func (f *fakeRepo) Delete(id int64) error         { return nil }
func (f *fakeRepo) Count() (int, error)           { return len(f.byID), nil }
func (f *fakeRepo) DailyCounts(days int) (map[string]int, error) {
	return nil, nil
}

func (f *fakeRepo) SetPinned(id int64, pinned bool) error {
	m, ok := f.byID[id]
	if !ok {
		return moment.ErrNotFound
	}
	m.Pinned = pinned
	return nil
}

func TestSetPinned(t *testing.T) {
	t.Run("pin sets flag and unpin clears", func(t *testing.T) {
		repo := newFakeRepo()
		svc := NewService(repo)
		m := repo.add(&moment.Moment{Content: "hi"})
		if err := svc.SetPinned(m.ID, true); err != nil {
			t.Fatalf("SetPinned(true): %v", err)
		}
		if !repo.byID[m.ID].Pinned {
			t.Fatal("after pin: Pinned = false, want true")
		}
		if err := svc.SetPinned(m.ID, false); err != nil {
			t.Fatalf("SetPinned(false): %v", err)
		}
		if repo.byID[m.ID].Pinned {
			t.Fatal("after unpin: Pinned = true, want false")
		}
	})

	t.Run("not found passes through", func(t *testing.T) {
		svc := NewService(newFakeRepo())
		if err := svc.SetPinned(999, true); !errors.Is(err, moment.ErrNotFound) {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	})
}
