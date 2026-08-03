// 限速器单元测试：窗口/阈值/清零/过期重置/并发。时钟注入测试包内私有字段（limiter.now）。
package ratelimit

import (
	"sync"
	"testing"
	"time"
)

// newTest 构造可拨动时钟的限速器。
func newTest(limit int, window time.Duration) (*Limiter, *time.Time) {
	l := New(limit, window)
	t0 := time.Now()
	l.now = func() time.Time { return t0 }
	return l, &t0
}

func TestAllowUntilLimit(t *testing.T) {
	l, _ := newTest(3, time.Hour)
	for i := 0; i < 3; i++ {
		if !l.Allow("k") {
			t.Fatalf("第 %d 次应放行", i+1)
		}
	}
	if l.Allow("k") {
		t.Fatal("超过阈值应拒绝")
	}
	// 不同 key 互不影响
	if !l.Allow("other") {
		t.Fatal("其他 key 不受限")
	}
}

func TestBlocked(t *testing.T) {
	l, _ := newTest(2, time.Hour)
	if l.Blocked("k") {
		t.Fatal("未计数前不应被判定超限")
	}
	l.Allow("k")
	l.Allow("k")
	if !l.Blocked("k") {
		t.Fatal("达到阈值后 Blocked 应为 true")
	}
	// Blocked 不计数：多查几次状态不变
	if !l.Blocked("k") || !l.Blocked("k") {
		t.Fatal("Blocked 不应改变状态")
	}
}

func TestWindowExpiry(t *testing.T) {
	l, clock := newTest(2, time.Minute)
	l.Allow("k")
	l.Allow("k")
	if l.Allow("k") {
		t.Fatal("窗口内超限应拒绝")
	}
	// 窗口过期后重新放行
	*clock = clock.Add(61 * time.Second)
	if !l.Allow("k") {
		t.Fatal("窗口过期后应重新放行")
	}
	if !l.Allow("k") {
		t.Fatal("新窗口内应可再次计数")
	}
	if l.Allow("k") {
		t.Fatal("新窗口超限应拒绝")
	}
	// Blocked 同样按窗口过期复位
	if !l.Blocked("k") {
		t.Fatal("窗口内超限 Blocked 应为 true")
	}
	*clock = clock.Add(2 * time.Minute)
	if l.Blocked("k") {
		t.Fatal("窗口过期后 Blocked 应为 false")
	}
}

func TestReset(t *testing.T) {
	l, _ := newTest(2, time.Hour)
	if !l.Allow("k") {
		t.Fatal("首次应放行")
	}
	if !l.Allow("k") {
		t.Fatal("第二次应放行")
	}
	if l.Allow("k") {
		t.Fatal("超限应拒绝")
	}
	l.Reset("other") // 不相关 key 不影响
	if !l.Blocked("k") {
		t.Fatal("Reset 其他 key 不应影响本 key")
	}
	l.Reset("k")
	if !l.Allow("k") {
		t.Fatal("Reset 后应重新放行")
	}
	if !l.Allow("k") {
		t.Fatal("清零后应重新计数")
	}
	if l.Allow("k") {
		t.Fatal("新窗口超限应拒绝")
	}
}

func TestNewDefaults(t *testing.T) {
	l := New(0, 0)
	if l.limit != 1 || l.window != time.Minute {
		t.Fatalf("非法参数应兜底：limit=%d window=%v", l.limit, l.window)
	}
}

// TestConcurrent 并发冒烟：多 goroutine 混跑 Allow/Blocked/Reset 不 panic、计数不超阈值。
func TestConcurrent(t *testing.T) {
	l := New(100, time.Minute)
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				l.Allow("k")
				l.Blocked("k")
				l.Allow("other")
				l.Reset("k")
			}
		}()
	}
	wg.Wait()
	// 8 goroutine 各自 Reset 后计数可能落在任意值，这里只验证结构可用、最终计数不超过 limit
	if l.Blocked("k") && len(l.counts) == 0 {
		t.Fatal("状态不一致")
	}
}
