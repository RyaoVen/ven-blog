// Package moderator 接口层：AI 自动审核 worker 的协调器（ticker 驱动，只做协调）。
// 发摘要邮件（mailer 窄接口注入）、失效声明（Invalidator）与日志统计，
// 不碰领域、不碰库；判定逻辑在 application/moderationapp，LLM 实现在 infrastructure/llm。
package moderator

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"ven_hybird/build/application/moderationapp"
	"ven_hybird/build/application/settingsapp"
)

// Mailer 发送窄接口（与 emailauth/service.go 同形；register.go 注入 SMTPMailer）。
type Mailer interface {
	Send(to, subject, text string) error
	SendHTML(to, subject, html string) error
}

// Invalidator 失效声明窄接口（hybrid.App 满足；见 hybrid/staticPage.go:62）。
type Invalidator interface {
	InvalidatePage(path string)
	DataChange(pattern string, params ...string) error
}

// Options worker 运行参数（构造依赖注入，便于测试）。
type Options struct {
	Interval time.Duration // 轮询间隔，默认 5m（BLOG_MODERATOR_INTERVAL 可配）
	Batch    int           // 每类宿主每轮上限，默认 20（BLOG_MODERATOR_BATCH 可配）
	Enabled  func() bool   // 每 tick 现查的开关（读 settings ugc_ai_moderation）
}

// 默认值（与配置总表一致）。
const (
	defaultInterval = 5 * time.Minute
	defaultBatch    = 20
)

// IntervalFromEnv 轮询间隔（BLOG_MODERATOR_INTERVAL；time.ParseDuration 解析失败或 ≤0 回退 5m）。
func IntervalFromEnv() time.Duration {
	if raw := os.Getenv("BLOG_MODERATOR_INTERVAL"); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			return d
		}
	}
	return defaultInterval
}

// BatchFromEnv 每类宿主每轮处理上限（BLOG_MODERATOR_BATCH；strconv.Atoi 失败或 ≤0 回退 20）。
func BatchFromEnv() int {
	if raw := os.Getenv("BLOG_MODERATOR_BATCH"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return n
		}
	}
	return defaultBatch
}

// Handler 自动审核 worker 协调器。
type Handler struct {
	svc          *moderationapp.Service
	settings     *settingsapp.Service // AuthorEmail 取收件人
	mailer       Mailer
	invalidate   Invalidator
	authorNameFn func() string // 留言板失效路径现取作者用户名（register.go 先例）
	siteURL      string        // siteURLFromEnv()：摘要邮件面板链接拼接
	opts         Options
}

// NewHandler 构造 worker 协调器。
func NewHandler(svc *moderationapp.Service, settings *settingsapp.Service, mailer Mailer,
	invalidate Invalidator, authorNameFn func() string, siteURL string, opts Options) *Handler {
	return &Handler{
		svc:          svc,
		settings:     settings,
		mailer:       mailer,
		invalidate:   invalidate,
		authorNameFn: authorNameFn,
		siteURL:      siteURL,
		opts:         opts,
	}
}

// Start 启动后台 goroutine（不阻塞调用方；Register 在 Listen 前调用）。
// 防重叠：上一轮未结束则跳过本次 tick（单实例内不会并发跑两轮）；
// 开关每 tick 现查（改设置即时生效）；ticker goroutine 随进程存活，无需额外关闭。
func (h *Handler) Start() {
	interval := h.opts.Interval
	if interval <= 0 {
		interval = defaultInterval
	}
	go h.run(interval)
}

// run ticker 循环。
func (h *Handler) run(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	busy := false
	for range ticker.C {
		if busy {
			continue // 上一轮未结束，跳过本次 tick
		}
		if h.opts.Enabled != nil && !h.opts.Enabled() {
			continue // 开关关闭，停手
		}
		busy = true
		h.RunOnce(context.Background())
		busy = false
	}
}

// RunOnce 执行一轮完整流程（ticker 回调；测试只测本函数，不测调度）。
func (h *Handler) RunOnce(ctx context.Context) {
	if h.opts.Enabled != nil && !h.opts.Enabled() {
		return
	}
	result, err := h.svc.AutoReview(ctx, h.batch())
	if err != nil {
		log.Printf("moderator: review round failed: %v", err)
		return // 失败安全在用例内已保证（内容保持 pending）
	}
	// 摘要邮件：存在驳回/不确定/失败项才发，避免噪声
	if result.Rejected+result.Uncertain+result.Failed > 0 {
		h.sendSummary(result)
	}
	// 失效声明：放行（与驳回的留言板）产生可见性变化，按结果逐条声明
	applyInvalidations(h.invalidate, h.authorNameFn, result)
	log.Printf("moderator: round done: processed=%d approved=%d rejected=%d uncertain=%d failed=%d",
		result.Processed, result.Approved, result.Rejected, result.Uncertain, result.Failed)
}

// batch 每类宿主每轮上限（零值回退默认）。
func (h *Handler) batch() int {
	if h.opts.Batch <= 0 {
		return defaultBatch
	}
	return h.opts.Batch
}

// sendSummary 发送摘要邮件；收件人缺失/发送失败都只记日志，不阻断本轮。
// 去重：已报告过的条目（settings 键 moderator_reported，kind:id）不再进邮件——
// LLM 持续故障或条目长期待人工时不会每轮刷相同内容；发送成功后把本轮条目键落库。
func (h *Handler) sendSummary(result *moderationapp.Result) {
	to, err := h.settings.AuthorEmail()
	if err != nil || to == "" {
		log.Printf("moderator: summary email skipped: no author email (err=%v)", err)
		return
	}
	filtered, newKeys := h.filterUnreported(result)
	if len(newKeys) == 0 {
		return // 全部报告过，本轮无新增异常
	}
	subject, html := buildSummaryEmail(filtered, h.siteURL)
	if err := h.mailer.SendHTML(to, subject, html); err != nil {
		log.Printf("moderator: summary email to %s failed: %v", to, err)
		return // 发送失败不落键，下轮重报
	}
	if err := h.settings.AppendModeratorReported(newKeys); err != nil {
		log.Printf("moderator: persist reported keys failed: %v", err)
	}
}

// filterUnreported 过滤已报告条目：返回只含新增异常的 Result 副本与其条目键（kind:id）。
// 读取已报告键失败时按"全部未报告"处理（宁可多发，不漏报）。
func (h *Handler) filterUnreported(result *moderationapp.Result) (*moderationapp.Result, []string) {
	reported, err := h.settings.ModeratorReported()
	if err != nil {
		reported = map[string]bool{}
	}
	filtered := &moderationapp.Result{
		Processed: result.Processed,
		Approved:  result.Approved,
	}
	keys := make([]string, 0, result.Rejected+result.Uncertain+result.Failed)
	keep := func(items []moderationapp.Item) []moderationapp.Item {
		out := make([]moderationapp.Item, 0, len(items))
		for _, it := range items {
			key := it.Kind + ":" + strconv.FormatInt(it.ID, 10)
			if reported[key] {
				continue
			}
			keys = append(keys, key)
			out = append(out, it)
		}
		return out
	}
	filtered.RejectedItems = keep(result.RejectedItems)
	filtered.UncertainItems = keep(result.UncertainItems)
	filtered.FailedItems = keep(result.FailedItems)
	filtered.Rejected = len(filtered.RejectedItems)
	filtered.Uncertain = len(filtered.UncertainItems)
	filtered.Failed = len(filtered.FailedItems)
	return filtered, keys
}
