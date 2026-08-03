// Package ratelimit 提供内存固定窗口限速器：认证接口防暴力破解/刷验证码用。
// 进程内存实现（无外部依赖）；窗口过期惰性重置，无后台协程。
package ratelimit

import (
	"sync"
	"time"
)

// Limiter 固定窗口内存限速器：同一 key 在一个窗口内最多放行 limit 次。
// 并发安全（单把互斥锁，计数结构足够轻量，认证路径低频无瓶颈）。
type Limiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	counts map[string]*windowCount
	now    func() time.Time // 可注入时钟（同包测试用），默认 time.Now
}

// windowCount 单个 key 的窗口计数：窗口起点 + 当前计数。
type windowCount struct {
	count int
	start time.Time
}

// New 构造限速器：window 内同一 key 最多放行 limit 次（limit<=0 按 1、window<=0 按 1 分钟兜底）。
func New(limit int, window time.Duration) *Limiter {
	if limit <= 0 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	return &Limiter{limit: limit, window: window, counts: make(map[string]*windowCount), now: time.Now}
}

// Allow 放行判定并计数：窗口内未超限返回 true（计数 +1）；已超限返回 false（不计数）。
// 失败计数随窗口整体过期归零（固定窗口，不做滑动）。
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	wc, ok := l.counts[key]
	if !ok || now.Sub(wc.start) >= l.window {
		l.sweepIfLarge(now)
		l.counts[key] = &windowCount{count: 1, start: now}
		return true
	}
	if wc.count >= l.limit {
		return false
	}
	wc.count++
	return true
}

// Blocked 查询 key 是否已超限（不计数）：窗口已过期视为未超限。
func (l *Limiter) Blocked(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	wc, ok := l.counts[key]
	if !ok {
		return false
	}
	if l.now().Sub(wc.start) >= l.window {
		return false
	}
	return wc.count >= l.limit
}

// Reset 清零 key 的计数（如登录成功后清除失败计数）。
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.counts, key)
}

// sweepIfLarge 计数表膨胀时顺带清扫已过期窗口，防止只增不减的内存泄漏。
// 惰性触发（表超过 1024 项才扫），摊薄到正常请求路径上，无后台协程。
func (l *Limiter) sweepIfLarge(now time.Time) {
	if len(l.counts) < 1024 {
		return
	}
	for k, wc := range l.counts {
		if now.Sub(wc.start) >= l.window {
			delete(l.counts, k)
		}
	}
}
