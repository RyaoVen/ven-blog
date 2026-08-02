// visitapp 用例服务单测：路径校验、Record 落库、Report 30s 节流、统计透传（假仓储）。
package visitapp

import (
	"errors"
	"testing"
	"time"

	"ven_hybird/build/domain/visit"
)

// fakeRepo 内存假仓储，记录调用供断言。
type fakeRepo struct {
	dates    []time.Time
	paths    []string
	total    int
	postTotal int
	daily    []visit.DailyCount
	hits     map[int64]int
}

func (f *fakeRepo) Record(date time.Time, path string) error {
	f.dates = append(f.dates, date)
	f.paths = append(f.paths, path)
	return nil
}

func (f *fakeRepo) Totals() (int, int, error) { return f.total, f.postTotal, nil }

func (f *fakeRepo) Daily(int) ([]visit.DailyCount, error) { return f.daily, nil }

func (f *fakeRepo) PostHits() (map[int64]int, error) { return f.hits, nil }

func TestValidatePath(t *testing.T) {
	cases := []struct {
		name string
		path string
		ok   bool
	}{
		{name: "root", path: "/", ok: true},
		{name: "post detail", path: "/posts/42", ok: true},
		{name: "nested", path: "/admin/posts/42/edit", ok: true},
		{name: "missing leading slash", path: "posts/42", ok: false},
		{name: "protocol relative", path: "//evil.example/x", ok: false},
		{name: "query", path: "/posts?page=2", ok: false},
		{name: "fragment", path: "/posts/42#comments", ok: false},
		{name: "empty", path: "", ok: false},
		{name: "max length ok", path: "/" + string(make([]byte, 254)), ok: true},
		{name: "too long", path: "/" + string(make([]byte, 255)), ok: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePath(tc.path)
			if (err == nil) != tc.ok {
				t.Fatalf("ValidatePath(%q) = %v, want ok=%v", tc.path, err, tc.ok)
			}
		})
	}
}

func TestService_Record(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo)
	date := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	if err := svc.Record(date, "/posts/1"); err != nil {
		t.Fatalf("record valid path failed: %v", err)
	}
	if err := svc.Record(date, "/posts/1"); err != nil {
		t.Fatalf("record duplicate path failed: %v", err)
	}
	if len(repo.paths) != 2 || repo.paths[0] != "/posts/1" || repo.paths[1] != "/posts/1" {
		t.Fatalf("expected 2 records of /posts/1, got %v", repo.paths)
	}
	if !repo.dates[0].Equal(date) {
		t.Fatalf("expected date %v passed through, got %v", date, repo.dates[0])
	}

	// 非法路径：返回 ErrInvalidPath 且不落库
	if err := svc.Record(date, "/posts/1?x=1"); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("expected ErrInvalidPath, got %v", err)
	}
	if len(repo.paths) != 2 {
		t.Fatalf("invalid path must not reach repo, got %v", repo.paths)
	}
}

func TestService_Report_Throttle(t *testing.T) {
	old := reportThrottle
	reportThrottle = 50 * time.Millisecond
	defer func() { reportThrottle = old }()

	repo := &fakeRepo{}
	svc := NewService(repo)
	date := time.Now()

	// 窗口内同路径：只落一次
	if err := svc.Report(date, "/posts/2"); err != nil {
		t.Fatalf("report failed: %v", err)
	}
	if err := svc.Report(date, "/posts/2"); err != nil {
		t.Fatalf("report duplicate failed: %v", err)
	}
	if len(repo.paths) != 1 {
		t.Fatalf("expected throttled to 1 record, got %v", repo.paths)
	}
	// 不同路径不受节流影响
	if err := svc.Report(date, "/"); err != nil {
		t.Fatalf("report other path failed: %v", err)
	}
	if len(repo.paths) != 2 {
		t.Fatalf("expected 2 records after other path, got %v", repo.paths)
	}
	// 窗口过期后同路径再次落库
	time.Sleep(2 * reportThrottle)
	if err := svc.Report(date, "/posts/2"); err != nil {
		t.Fatalf("report after window failed: %v", err)
	}
	if len(repo.paths) != 3 {
		t.Fatalf("expected 3 records after window expiry, got %v", repo.paths)
	}
	// 非法路径：不落库也不记录节流时间
	if err := svc.Report(date, "/posts/2#c"); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("expected ErrInvalidPath, got %v", err)
	}
	if len(repo.paths) != 3 {
		t.Fatalf("invalid path must not reach repo, got %v", repo.paths)
	}
}

func TestService_TotalsAndDaily(t *testing.T) {
	repo := &fakeRepo{total: 100, postTotal: 60, daily: []visit.DailyCount{{Date: "2026-08-01", Count: 3}}, hits: map[int64]int{7: 9}}
	svc := NewService(repo)

	total, postTotal, err := svc.Totals()
	if err != nil {
		t.Fatalf("totals failed: %v", err)
	}
	if total != 100 || postTotal != 60 {
		t.Fatalf("expected totals 100/60, got %d/%d", total, postTotal)
	}

	daily, err := svc.Daily(30)
	if err != nil {
		t.Fatalf("daily failed: %v", err)
	}
	if len(daily) != 1 || daily[0].Date != "2026-08-01" || daily[0].Count != 3 {
		t.Fatalf("unexpected daily: %+v", daily)
	}

	hits, err := svc.PostHits()
	if err != nil {
		t.Fatalf("post hits failed: %v", err)
	}
	if hits[7] != 9 {
		t.Fatalf("expected hits[7]=9, got %d", hits[7])
	}
}
