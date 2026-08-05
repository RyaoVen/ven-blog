package persistence

import (
	"database/sql"
	"errors"
	"fmt"

	"ven_hybird/build/domain/comment"
)

// CommentRepository 是 comment.Repository 的 MySQL 实现。
type CommentRepository struct {
	db *sql.DB
}

// NewCommentRepository 构造评论仓储。
func NewCommentRepository(db *sql.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

// commentSelect 评论查询列（post_id/moment_id 为 NULLable，扫描走 NullInt64 转换）。
const commentSelect = `SELECT c.id, c.post_id, c.moment_id, c.user_id, u.username, c.content, c.reply_to, c.status, c.rejected_reason, c.created_at
FROM comments c JOIN users u ON u.id = c.user_id`

// scanComment 从行扫描评论（宿主 NULLable 列转为零值 int64）。
func scanComment(row interface{ Scan(...any) error }) (*comment.Comment, error) {
	c := &comment.Comment{}
	var postID, momentID sql.NullInt64
	err := row.Scan(&c.ID, &postID, &momentID, &c.UserID, &c.Username, &c.Content, &c.ReplyTo, &c.Status, &c.RejectedReason, &c.CreatedAt)
	c.PostID = postID.Int64
	c.MomentID = momentID.Int64
	return c, err
}

// ListByPost 返回文章下的可见评论（approved，创建时间倒序，联表取用户名）。
func (r *CommentRepository) ListByPost(postID int64) ([]*comment.Comment, error) {
	return r.listWhere("c.post_id = ? AND c.status = ?", postID, comment.StatusApproved)
}

// ListByMoment 返回动态下的可见评论（approved，创建时间倒序，联表取用户名）。
func (r *CommentRepository) ListByMoment(momentID int64) ([]*comment.Comment, error) {
	return r.listWhere("c.moment_id = ? AND c.status = ?", momentID, comment.StatusApproved)
}

// listWhere 按宿主条件查询评论列表（args 依次填占位符）。
func (r *CommentRepository) listWhere(where string, args ...any) ([]*comment.Comment, error) {
	rows, err := r.db.Query(commentSelect+" WHERE "+where+" ORDER BY c.created_at DESC, c.id DESC", args...)
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	defer func() { _ = rows.Close() }()
	comments := make([]*comment.Comment, 0)
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan comment: %w", err)
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

// MomentCommentCounts 动态评论数分组统计（/moments 页展示用；仅统计 approved，避免 pending/rejected 虚增公开计数）。
func (r *CommentRepository) MomentCommentCounts() (map[int64]int, error) {
	rows, err := r.db.Query("SELECT moment_id, COUNT(*) FROM comments WHERE moment_id IS NOT NULL AND status = ? GROUP BY moment_id", comment.StatusApproved)
	if err != nil {
		return nil, fmt.Errorf("moment comment counts: %w", err)
	}
	defer func() { _ = rows.Close() }()
	counts := make(map[int64]int)
	for rows.Next() {
		var id int64
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, fmt.Errorf("scan moment comment count: %w", err)
		}
		counts[id] = n
	}
	return counts, rows.Err()
}

// ListAll 返回全站评论（创建时间倒序，联表用户名与所属文章标题）。
func (r *CommentRepository) ListAll(limit int) ([]*comment.Comment, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.Query(
		`SELECT c.id, c.post_id, c.moment_id, c.user_id, u.username, p.title, c.content, c.reply_to, c.status, c.rejected_reason, c.created_at
		FROM comments c
		JOIN users u ON u.id = c.user_id
		LEFT JOIN posts p ON p.id = c.post_id
		ORDER BY c.created_at DESC, c.id DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list all comments: %w", err)
	}
	defer func() { _ = rows.Close() }()
	comments := make([]*comment.Comment, 0)
	for rows.Next() {
		c := &comment.Comment{}
		var postID, momentID sql.NullInt64
		var title sql.NullString
		if err := rows.Scan(&c.ID, &postID, &momentID, &c.UserID, &c.Username, &title, &c.Content, &c.ReplyTo, &c.Status, &c.RejectedReason, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan comment: %w", err)
		}
		c.PostID = postID.Int64
		c.MomentID = momentID.Int64
		c.PostTitle = title.String
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

// Count 返回评论总数。
func (r *CommentRepository) Count() (int, error) {
	var n int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM comments").Scan(&n); err != nil {
		return 0, fmt.Errorf("count comments: %w", err)
	}
	return n, nil
}

// Get 按 ID 取评论，不存在返回 comment.ErrNotFound。
func (r *CommentRepository) Get(id int64) (*comment.Comment, error) {
	c, err := scanComment(r.db.QueryRow(commentSelect+" WHERE c.id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, comment.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get comment %d: %w", id, err)
	}
	return c, nil
}

// Create 创建评论并回填 ID 与时间戳（宿主 NULLable 列按 Target 二选一写入）。
func (r *CommentRepository) Create(c *comment.Comment) error {
	var postID, momentID any
	if c.PostID > 0 {
		postID = c.PostID
	}
	if c.MomentID > 0 {
		momentID = c.MomentID
	}
	res, err := r.db.Exec(
		"INSERT INTO comments (post_id, moment_id, user_id, content, reply_to, status) VALUES (?, ?, ?, ?, ?, ?)",
		postID, momentID, c.UserID, c.Content, c.ReplyTo, c.Status,
	)
	if err != nil {
		return fmt.Errorf("create comment: %w", err)
	}
	c.ID, err = res.LastInsertId()
	if err != nil {
		return err
	}
	created, err := r.Get(c.ID)
	if err != nil {
		return err
	}
	*c = *created
	return nil
}

// Delete 删除评论，不存在返回 comment.ErrNotFound。
func (r *CommentRepository) Delete(id int64) error {
	res, err := r.db.Exec("DELETE FROM comments WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete comment %d: %w", id, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return comment.ErrNotFound
	}
	return nil
}

// 带文章标题的联表查询（列序：commentSelect 列 + p.title 紧随 u.username 后）。
const commentSelectWithTitle = `SELECT c.id, c.post_id, c.moment_id, c.user_id, u.username, p.title, c.content, c.reply_to, c.status, c.rejected_reason, c.created_at
FROM comments c
JOIN users u ON u.id = c.user_id
LEFT JOIN posts p ON p.id = c.post_id`

// scanCommentWithTitle 从行扫描评论（含文章标题；列序与 commentSelectWithTitle 一致）。
func scanCommentWithTitle(row interface{ Scan(...any) error }) (*comment.Comment, error) {
	c := &comment.Comment{}
	var postID, momentID sql.NullInt64
	var title sql.NullString
	err := row.Scan(&c.ID, &postID, &momentID, &c.UserID, &c.Username, &title, &c.Content, &c.ReplyTo, &c.Status, &c.RejectedReason, &c.CreatedAt)
	c.PostID = postID.Int64
	c.MomentID = momentID.Int64
	c.PostTitle = title.String
	return c, err
}

// listReviewQueue 审核队列查询（创建时间正序，带文章标题；extraWhere 附加条件如 AI 未判）。
func (r *CommentRepository) listReviewQueue(status string, extraWhere string) ([]*comment.Comment, error) {
	rows, err := r.db.Query(
		commentSelectWithTitle+" WHERE c.status = ?"+extraWhere+" ORDER BY c.created_at ASC, c.id ASC",
		status,
	)
	if err != nil {
		return nil, fmt.Errorf("list review queue comments: %w", err)
	}
	defer func() { _ = rows.Close() }()
	comments := make([]*comment.Comment, 0)
	for rows.Next() {
		c, err := scanCommentWithTitle(rows)
		if err != nil {
			return nil, fmt.Errorf("scan comment: %w", err)
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

// ListPending 返回待审核评论（创建时间正序，联表文章标题）。
func (r *CommentRepository) ListPending() ([]*comment.Comment, error) {
	return r.listReviewQueue(comment.StatusPending, "")
}

// ListUnreviewedPending 返回 AI 未判的待审评论（worker 队列）。
func (r *CommentRepository) ListUnreviewedPending() ([]*comment.Comment, error) {
	return r.listReviewQueue(comment.StatusPending, " AND c.ai_reviewed_at IS NULL")
}

// ClaimAIReview 原子抢占 AI 审核权：仅 pending 且 AI 未审行置 ai_reviewed_at（NOW()）。
// RowsAffected=1 抢占成功；=0 表示已被其他实例抢占或已审（返回 false）——
// 多实例 worker 并发拉同一队列时，同一条只有一方抢到，杜绝重复审核。
func (r *CommentRepository) ClaimAIReview(id int64) (bool, error) {
	res, err := r.db.Exec("UPDATE comments SET ai_reviewed_at = NOW() WHERE id = ? AND status = ? AND ai_reviewed_at IS NULL", id, comment.StatusPending)
	if err != nil {
		return false, fmt.Errorf("claim ai review of comment %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("claim ai review of comment %d: %w", id, err)
	}
	return n == 1, nil
}

// UnclaimAIReview 回滚抢占（LLM 判定/写库失败后释放；仅 pending 行可回滚，保持"失败下轮重审"）。
func (r *CommentRepository) UnclaimAIReview(id int64) error {
	if _, err := r.db.Exec("UPDATE comments SET ai_reviewed_at = NULL WHERE id = ? AND status = ?", id, comment.StatusPending); err != nil {
		return fmt.Errorf("unclaim ai review of comment %d: %w", id, err)
	}
	return nil
}

// ListRejected 返回被驳回评论（创建时间正序，联表文章标题）。
func (r *CommentRepository) ListRejected() ([]*comment.Comment, error) {
	return r.listReviewQueue(comment.StatusRejected, "")
}

// SetStatus 更新评论状态（审核通过/打回），同时清空驳回原因。
func (r *CommentRepository) SetStatus(id int64, status string) error {
	if _, err := r.db.Exec("UPDATE comments SET status = ?, rejected_reason = '' WHERE id = ?", status, id); err != nil {
		return fmt.Errorf("set status of comment %d: %w", id, err)
	}
	return nil
}

// SetRejected 驳回评论并记录驳回原因。
func (r *CommentRepository) SetRejected(id int64, reason string) error {
	if _, err := r.db.Exec("UPDATE comments SET status = ?, rejected_reason = ? WHERE id = ?", comment.StatusRejected, reason, id); err != nil {
		return fmt.Errorf("reject comment %d: %w", id, err)
	}
	return nil
}
