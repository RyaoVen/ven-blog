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

// ListByPost 返回文章下的评论（创建时间倒序，联表取用户名）。
func (r *CommentRepository) ListByPost(postID int64) ([]*comment.Comment, error) {
	rows, err := r.db.Query(
		`SELECT c.id, c.post_id, c.user_id, u.username, c.content, c.reply_to, c.created_at
		FROM comments c JOIN users u ON u.id = c.user_id
		WHERE c.post_id = ? ORDER BY c.created_at DESC, c.id DESC`,
		postID,
	)
	if err != nil {
		return nil, fmt.Errorf("list comments for post %d: %w", postID, err)
	}
	defer func() { _ = rows.Close() }()
	comments := make([]*comment.Comment, 0)
	for rows.Next() {
		c := &comment.Comment{}
		if err := rows.Scan(&c.ID, &c.PostID, &c.UserID, &c.Username, &c.Content, &c.ReplyTo, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan comment: %w", err)
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

// Get 按 ID 取评论，不存在返回 comment.ErrNotFound。
func (r *CommentRepository) Get(id int64) (*comment.Comment, error) {
	c := &comment.Comment{}
	err := r.db.QueryRow(
		`SELECT c.id, c.post_id, c.user_id, u.username, c.content, c.reply_to, c.created_at
		FROM comments c JOIN users u ON u.id = c.user_id WHERE c.id = ?`,
		id,
	).Scan(&c.ID, &c.PostID, &c.UserID, &c.Username, &c.Content, &c.ReplyTo, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, comment.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get comment %d: %w", id, err)
	}
	return c, nil
}

// Create 创建评论并回填 ID 与时间戳。
func (r *CommentRepository) Create(c *comment.Comment) error {
	res, err := r.db.Exec(
		"INSERT INTO comments (post_id, user_id, content, reply_to) VALUES (?, ?, ?, ?)",
		c.PostID, c.UserID, c.Content, c.ReplyTo,
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

// ListAll 返回全站评论（创建时间倒序，联表用户名与所属文章标题）。
func (r *CommentRepository) ListAll(limit int) ([]*comment.Comment, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.Query(
		`SELECT c.id, c.post_id, c.user_id, u.username, p.title, c.content, c.reply_to, c.created_at
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
		var title sql.NullString
		if err := rows.Scan(&c.ID, &c.PostID, &c.UserID, &c.Username, &title, &c.Content, &c.ReplyTo, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan comment: %w", err)
		}
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
