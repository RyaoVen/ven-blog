package persistence

import (
	"database/sql"
	"errors"
	"fmt"

	"ven_hybird/build/domain/post"
)

// PostRepository 是 post.Repository 的 MySQL 实现。
type PostRepository struct {
	db *sql.DB
}

// NewPostRepository 构造文章仓储。
func NewPostRepository(db *sql.DB) *PostRepository {
	return &PostRepository{db: db}
}

// 列表/详情查询联表 users 取作者名；标签在后续期次接入（tags/post_tags 表已建）。
const postSelect = `SELECT p.id, p.author_id, u.username, p.title, p.summary, p.content, p.cover_url, p.created_at, p.updated_at
FROM posts p JOIN users u ON u.id = p.author_id`

// scanPost 从行扫描文章（列序与 postSelect 一致）。
func scanPost(row interface{ Scan(...any) error }) (*post.Post, error) {
	p := &post.Post{}
	err := row.Scan(&p.ID, &p.AuthorID, &p.AuthorName, &p.Title, &p.Summary, &p.Content, &p.CoverURL, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

// List 返回全部文章（创建时间倒序）。
func (r *PostRepository) List() ([]*post.Post, error) {
	rows, err := r.db.Query(postSelect + " ORDER BY p.created_at DESC, p.id DESC")
	if err != nil {
		return nil, fmt.Errorf("list posts: %w", err)
	}
	defer func() { _ = rows.Close() }()
	posts := make([]*post.Post, 0)
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, fmt.Errorf("scan post: %w", err)
		}
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

// Get 按 ID 取文章，不存在返回 post.ErrNotFound。
func (r *PostRepository) Get(id int64) (*post.Post, error) {
	p, err := scanPost(r.db.QueryRow(postSelect+" WHERE p.id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, post.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get post %d: %w", id, err)
	}
	return p, nil
}

// Create 新建文章并回填 ID 与时间戳。
func (r *PostRepository) Create(p *post.Post) error {
	res, err := r.db.Exec(
		"INSERT INTO posts (author_id, title, summary, content, cover_url) VALUES (?, ?, ?, ?, ?)",
		p.AuthorID, p.Title, p.Summary, p.Content, p.CoverURL,
	)
	if err != nil {
		return fmt.Errorf("create post: %w", err)
	}
	p.ID, err = res.LastInsertId()
	if err != nil {
		return err
	}
	created, err := r.Get(p.ID)
	if err != nil {
		return err
	}
	*p = *created
	return nil
}

// Update 更新标题与正文，不存在返回 post.ErrNotFound。
func (r *PostRepository) Update(p *post.Post) error {
	res, err := r.db.Exec("UPDATE posts SET title = ?, content = ? WHERE id = ?", p.Title, p.Content, p.ID)
	if err != nil {
		return fmt.Errorf("update post %d: %w", p.ID, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return post.ErrNotFound
	}
	return nil
}

// Delete 删除文章，不存在返回 post.ErrNotFound。
func (r *PostRepository) Delete(id int64) error {
	res, err := r.db.Exec("DELETE FROM posts WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete post %d: %w", id, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return post.ErrNotFound
	}
	return nil
}
