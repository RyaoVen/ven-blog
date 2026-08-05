// Package moderationapp 自动审核用例服务：编排评论/留言板待审列表与 Moderator 判定。
// 不做失效声明、不发邮件（协调归接口层）；返回统计供摘要邮件与日志。
package moderationapp

import (
	"context"
	"strconv"
	"unicode/utf8"

	"ven_hybird/build/application/commentapp"
	"ven_hybird/build/application/guestbookapp"
	"ven_hybird/build/domain/comment"
	"ven_hybird/build/domain/moderation"
)

// 明细行类型（与 moderation.HostKind 取值一致）。
const (
	KindComment   = string(moderation.HostComment)   // 评论（宿主为文章或动态）
	KindGuestbook = string(moderation.HostGuestbook) // 留言板留言
)

// 每类宿主每轮处理上限（调用方未传 limit 时回退；env BLOG_MODERATOR_BATCH 可配）。
const defaultBatch = 20

// Service 自动审核用例服务。
type Service struct {
	comments  *commentapp.Service
	guestbook *guestbookapp.Service
	moderator moderation.Moderator // 实现由组装根注入（infrastructure/llm）
}

// NewService 构造自动审核用例服务。
func NewService(comments *commentapp.Service, guestbook *guestbookapp.Service, moderator moderation.Moderator) *Service {
	return &Service{comments: comments, guestbook: guestbook, moderator: moderator}
}

// Result 一轮审核统计（供摘要邮件与日志）。
type Result struct {
	Processed      int    // 实际处理条数（两类合计，≤ 2×limit）
	Approved       int    // 放行条数
	Rejected       int    // 驳回条数
	Uncertain      int    // 保持 pending（AI 不确定）
	Failed         int    // 判定/写库失败（重试后仍失败，保持 pending）
	ApprovedItems  []Item // 放行明细（接口层失效声明用）
	RejectedItems  []Item // 驳回明细（邮件用）
	UncertainItems []Item // 需人工复核明细（邮件用）
	FailedItems    []Item // 判定失败明细（邮件用；设计文档 Result 未列，邮件模板需要）
}

// Item 明细行（邮件与失效声明共用）。
type Item struct {
	Kind      string // KindComment | KindGuestbook
	ID        int64
	Username  string
	Content   string
	HostTitle string // 文章标题（有则用）/ "文章 #id" 兜底 / "动态 #id" / 留言板固定 "作者主页"
	Reason    string // rejected 时的驳回原因；其他为空
	PostID    int64  // comment 宿主为文章时非零（失效声明用）
	MomentID  int64  // comment 宿主为动态时非零
}

// AutoReview 拉取两类 AI 未判待审内容（各上限 limit）并逐条判定：
//   claim 抢占成功 → approve → 调用宿主 Service 的 Approve；reject → 调用 Reject(id, reason)；
//   pending → 抢占已打标，交人工（不再重复提交 LLM）；Review 返回 error → 重试 1 次，
//   仍失败回滚抢占（保持 pending，下轮重判）；写库失败同样回滚。
//   claim 失败（已被他实例抢占/已审）→ 跳过不计数（该条由抢占者处理）。
// 逐条串行（成本可控、顺序确定）；ctx 透传给 Moderator（ticker 场景传 Background 派生）。
// 任一宿主查询出错 → 返回部分结果与 error（本轮整体失败由接口层记录日志；已处理的保持已处理状态）。
func (s *Service) AutoReview(ctx context.Context, limit int) (*Result, error) {
	if limit <= 0 {
		limit = defaultBatch
	}
	result := &Result{}
	comments, err := s.comments.ListUnreviewedPending()
	if err != nil {
		return result, err
	}
	entries, err := s.guestbook.ListUnreviewedPending()
	if err != nil {
		return result, err
	}
	if len(comments) > limit {
		comments = comments[:limit]
	}
	if len(entries) > limit {
		entries = entries[:limit]
	}
	for _, c := range comments {
		claimed, err := s.comments.ClaimAIReview(c.ID)
		if err != nil || !claimed {
			continue // 抢占失败：他实例在处理或已审，本轮跳过（不计数）
		}
		hostTitle := hostTitleOfComment(c)
		s.process(ctx, result, Item{
			Kind:      KindComment,
			ID:        c.ID,
			Username:  c.Username,
			Content:   c.Content,
			HostTitle: hostTitle,
			PostID:    c.PostID,
			MomentID:  c.MomentID,
		}, moderation.Request{
			Host:      moderation.HostComment,
			HostTitle: hostTitle,
			Content:   c.Content,
			ReplyTo:   c.ReplyTo,
		}, func(item Item, verdict moderation.Verdict) error {
			if verdict.Action == moderation.ActionApprove {
				_, err := s.comments.Approve(item.ID)
				return err
			}
			_, err := s.comments.Reject(item.ID, clipReason(verdict.Reason))
			return err
		}, s.comments.UnclaimAIReview)
	}
	for _, e := range entries {
		claimed, err := s.guestbook.ClaimAIReview(e.ID)
		if err != nil || !claimed {
			continue // 抢占失败：他实例在处理或已审，本轮跳过（不计数）
		}
		s.process(ctx, result, Item{
			Kind:      KindGuestbook,
			ID:        e.ID,
			Username:  e.Username,
			Content:   e.Content,
			HostTitle: guestbookHostTitle,
		}, moderation.Request{
			Host:      moderation.HostGuestbook,
			HostTitle: guestbookHostTitle,
			Content:   e.Content,
		}, func(item Item, verdict moderation.Verdict) error {
			if verdict.Action == moderation.ActionApprove {
				return s.guestbook.Approve(item.ID)
			}
			return s.guestbook.Reject(item.ID, clipReason(verdict.Reason))
		}, s.guestbook.UnclaimAIReview)
	}
	return result, nil
}

// 留言板宿主标题固定为作者主页。
const guestbookHostTitle = "作者主页"

// hostTitleOfComment 评论宿主标题：文章标题（读取模型字段，仓储联表填充）；
// 无标题字段时按宿主兜底 "文章 #id" / "动态 #id"。
func hostTitleOfComment(c *comment.Comment) string {
	if c.PostID > 0 {
		if c.PostTitle != "" {
			return c.PostTitle
		}
		return "文章 #" + strconv.FormatInt(c.PostID, 10)
	}
	if c.MomentID > 0 {
		return "动态 #" + strconv.FormatInt(c.MomentID, 10)
	}
	return "评论"
}

// process 判定单条并落结果（调用前已完成 claim 抢占）：
//   approve → 宿主 Approve（写库失败回滚抢占，保持 pending 下轮重试）；reject → 宿主 Reject（reason 截断到领域上限）；
//   pending → 抢占已打标（ai_reviewed_at 非空），交人工复核；Review 两次都失败 → 回滚抢占（保持 pending）记 failed。
func (s *Service) process(ctx context.Context, result *Result, item Item, req moderation.Request,
	write func(Item, moderation.Verdict) error, unclaim func(int64) error) {
	result.Processed++
	verdict, err := s.reviewWithRetry(ctx, req)
	if err != nil {
		// LLM 两次失败：回滚抢占标记，保持"下轮重审"（回滚失败则条目不再进队列，记 failed 上报）
		if uerr := unclaim(item.ID); uerr != nil {
			result.Failed++
			result.FailedItems = append(result.FailedItems, item)
			return
		}
		result.Failed++
		result.FailedItems = append(result.FailedItems, item)
		return
	}
	switch verdict.Action {
	case moderation.ActionApprove, moderation.ActionReject:
		if err := write(item, verdict); err != nil {
			// 写库失败：回滚抢占（保持 pending 下轮重试）——绝不让失败路径放行或误杀
			if uerr := unclaim(item.ID); uerr != nil {
				result.Failed++
				result.FailedItems = append(result.FailedItems, item)
				return
			}
			result.Failed++
			result.FailedItems = append(result.FailedItems, item)
			return
		}
	}
	switch verdict.Action {
	case moderation.ActionApprove:
		result.Approved++
		result.ApprovedItems = append(result.ApprovedItems, item)
	case moderation.ActionReject:
		item.Reason = clipReason(verdict.Reason)
		result.Rejected++
		result.RejectedItems = append(result.RejectedItems, item)
	default: // pending：抢占已打标（ai_reviewed_at 非空），交人工复核，不再重复提交 LLM
		result.Uncertain++
		result.UncertainItems = append(result.UncertainItems, item)
	}
}

// reviewWithRetry 判定 + 重试 1 次（仅 error 触发；成功或返回 pending 不重试）。
func (s *Service) reviewWithRetry(ctx context.Context, req moderation.Request) (moderation.Verdict, error) {
	verdict, err := s.moderator.Review(ctx, req)
	if err == nil {
		return verdict, nil
	}
	return s.moderator.Review(ctx, req) // 重试 1 次，仍失败由调用方保持 pending
}

// clipReason 截断驳回原因到领域上限（200 字节，UTF-8 安全，避免模型超长 reason 导致 Reject 校验失败）。
func clipReason(reason string) string {
	if len(reason) <= 200 {
		return reason
	}
	cut := 0
	used := 0
	for _, r := range reason {
		n := utf8.RuneLen(r)
		if used+n > 200 {
			break
		}
		cut += n
		used += n
	}
	return reason[:cut]
}
