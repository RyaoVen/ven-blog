// Package visitapp 访问统计用例服务：记录访问与后台统计查询。
package visitapp

import (
	"errors"
	"strings"
	"sync"
	"time"

	"ven_hybird/build/domain/visit"
)

// ErrInvalidPath 非法埋点路径（不记录；接口层映射 400）。
var ErrInvalidPath = errors.New("invalid visit path")

// maxPathLen 与 visits.path 列（VARCHAR(255)）一致。
const maxPathLen = 255

// reportThrottle 是同 path 上报的最小间隔（防抖；测试可临时缩短）。
var reportThrottle = 30 * time.Second

// ValidatePath 校验埋点路径：/ 开头、非协议相对（//）、无 query/片段、长度 ≤ 255。
func ValidatePath(path string) error {
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return ErrInvalidPath
	}
	if len(path) > maxPathLen {
		return ErrInvalidPath
	}
	if strings.ContainsAny(path, "?#") {
		return ErrInvalidPath
	}
	return nil
}

// Service 访问统计用例服务。
type Service struct {
	repo visit.Repository

	// 30s 同 path 内存节流（SPA 上报接口用；中间件注入的 Record 不受限）
	mu   sync.Mutex
	last map[string]time.Time
}

// NewService 构造访问统计用例服务。
func NewService(repo visit.Repository) *Service {
	return &Service{repo: repo}
}

// Record 记录一次访问（无节流；网关埋点中间件注入用）。
// 路径非法时返回 ErrInvalidPath，不落库。
func (s *Service) Record(date time.Time, path string) error {
	if err := ValidatePath(path); err != nil {
		return err
	}
	return s.repo.Record(date, path)
}

// Report 记录一次访问并做 30s 同 path 内存节流（SPA 上报接口用）：
// 节流窗口内同路径重复上报直接吞掉（返回 nil 不落库）。
func (s *Service) Report(date time.Time, path string) error {
	if err := ValidatePath(path); err != nil {
		return err
	}
	now := time.Now()
	s.mu.Lock()
	if s.last == nil {
		s.last = make(map[string]time.Time)
	}
	if t, ok := s.last[path]; ok && now.Sub(t) < reportThrottle {
		s.mu.Unlock()
		return nil
	}
	// 简单内存节流：表膨胀时清一次过期键（重启即清，无持久化负担）
	if len(s.last) >= 4096 {
		for p, t := range s.last {
			if now.Sub(t) >= reportThrottle {
				delete(s.last, p)
			}
		}
	}
	s.last[path] = now
	s.mu.Unlock()
	return s.repo.Record(date, path)
}

// Totals 全站访问总量与文章点击总量（后台统计）。
func (s *Service) Totals() (total int, postTotal int, err error) {
	return s.repo.Totals()
}

// Daily 近 days 天每日访问总量（日期升序，不足补零）。
func (s *Service) Daily(days int) ([]visit.DailyCount, error) {
	return s.repo.Daily(days)
}

// PostHits 各文章累计点击（键为文章 ID；后台文章列表用）。
func (s *Service) PostHits() (map[int64]int, error) {
	return s.repo.PostHits()
}
