package subscribeapp

import (
	"errors"
	"testing"

	"ven_hybird/build/domain/subscriber"
)

// fakeRepo 内存实现 subscriber.Repository（测试用）。
// List 语义与真实仓储一致：按 ID 升序返回全部订阅者。
type fakeRepo struct {
	emails []string // 按订阅先后顺序
	next   int64
	err    error // 模拟仓储故障
}

func (f *fakeRepo) Create(s *subscriber.Subscriber) error {
	if f.err != nil {
		return f.err
	}
	for _, e := range f.emails {
		if e == s.Email {
			return subscriber.ErrAlreadySubscribed
		}
	}
	f.next++
	s.ID = f.next
	f.emails = append(f.emails, s.Email)
	return nil
}

func (f *fakeRepo) Count() (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	return len(f.emails), nil
}

func (f *fakeRepo) List() ([]*subscriber.Subscriber, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([]*subscriber.Subscriber, 0, len(f.emails))
	for i, e := range f.emails {
		out = append(out, &subscriber.Subscriber{ID: int64(i + 1), Email: e})
	}
	return out, nil
}

func TestSubscribers(t *testing.T) {
	t.Run("returns all emails in subscribe order", func(t *testing.T) {
		repo := &fakeRepo{emails: []string{"a@example.com", "b@example.com", "c@example.com"}}
		got, err := NewService(repo).Subscribers()
		if err != nil {
			t.Fatalf("Subscribers: %v", err)
		}
		want := []string{"a@example.com", "b@example.com", "c@example.com"}
		if len(got) != len(want) {
			t.Fatalf("Subscribers = %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("Subscribers = %v, want %v", got, want)
			}
		}
	})

	t.Run("empty when no subscribers", func(t *testing.T) {
		got, err := NewService(&fakeRepo{}).Subscribers()
		if err != nil {
			t.Fatalf("Subscribers: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("Subscribers = %v, want empty", got)
		}
	})

	t.Run("repo error propagates", func(t *testing.T) {
		repo := &fakeRepo{err: errors.New("db down")}
		if _, err := NewService(repo).Subscribers(); err == nil {
			t.Fatal("Subscribers want error, got nil")
		}
	})
}

func TestSubscribeIdempotent(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo)
	already, err := svc.Subscribe("a@example.com")
	if err != nil || already {
		t.Fatalf("first subscribe: already=%v err=%v", already, err)
	}
	already, err = svc.Subscribe("a@example.com")
	if err != nil || !already {
		t.Fatalf("repeat subscribe: already=%v err=%v, want already=true", already, err)
	}
	if n, _ := svc.Count(); n != 1 {
		t.Fatalf("Count = %d, want 1", n)
	}
}

func TestSubscribeValidation(t *testing.T) {
	svc := NewService(&fakeRepo{})
	if _, err := svc.Subscribe("not-an-email"); err == nil {
		t.Fatal("invalid email want ValidationError, got nil")
	}
}
