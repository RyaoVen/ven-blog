package persistence

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

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

// 列表/详情查询联表 users 取作者名；标签经 post_tags/tags 表单独批量填充。
const postSelect = `SELECT p.id, p.author_id, u.username, p.title, p.category, p.summary, p.content, p.cover_url, p.created_at, p.updated_at
FROM posts p JOIN users u ON u.id = p.author_id`

// scanPost 从行扫描文章（列序与 postSelect 一致）。
func scanPost(row interface{ Scan(...any) error }) (*post.Post, error) {
	p := &post.Post{}
	err := row.Scan(&p.ID, &p.AuthorID, &p.AuthorName, &p.Title, &p.Category, &p.Summary, &p.Content, &p.CoverURL, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

// ListPaged 分页返回文章（创建时间倒序）与符合过滤条件的总数；
// tag 非空时用 EXISTS 子查询按标签过滤，pageSize <= 0 表示不分页（返回全部）。
func (r *PostRepository) ListPaged(tag string, page, pageSize int) ([]*post.Post, int, error) {
	where := ""
	args := make([]any, 0, 3)
	if tag != "" {
		where = ` WHERE EXISTS (SELECT 1 FROM post_tags pt JOIN tags t ON t.id = pt.tag_id
			WHERE pt.post_id = p.id AND t.name = ?)`
		args = append(args, tag)
	}

	var total int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM posts p"+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count posts: %w", err)
	}

	query := postSelect + where + " ORDER BY p.created_at DESC, p.id DESC"
	if pageSize > 0 {
		if page < 1 {
			page = 1
		}
		query += " LIMIT ? OFFSET ?"
		args = append(args, pageSize, (page-1)*pageSize)
	}
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list posts: %w", err)
	}
	defer func() { _ = rows.Close() }()
	posts := make([]*post.Post, 0)
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan post: %w", err)
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if err := r.fillTags(posts); err != nil {
		return nil, 0, err
	}
	return posts, total, nil
}

// maxSearchLimit 单次搜索返回上限。
const maxSearchLimit = 50

// likeEscaper 转义 LIKE 模式中的通配符（配 ESCAPE '\\' 子句）。
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// Search 按关键词 LIKE 匹配标题或正文（创建时间倒序）；limit 归一到 [1, maxSearchLimit]。
func (r *PostRepository) Search(query string, limit int) ([]*post.Post, error) {
	if limit <= 0 || limit > maxSearchLimit {
		limit = maxSearchLimit
	}
	pattern := "%" + likeEscaper.Replace(query) + "%"
	rows, err := r.db.Query(
		postSelect+` WHERE p.title LIKE ? ESCAPE '\\' OR p.content LIKE ? ESCAPE '\\'
ORDER BY p.created_at DESC, p.id DESC LIMIT ?`,
		pattern, pattern, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search posts: %w", err)
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

// ListByAuthor 返回指定作者的全部文章（创建时间倒序）。
func (r *PostRepository) ListByAuthor(authorID int64) ([]*post.Post, error) {
	rows, err := r.db.Query(postSelect+" WHERE p.author_id = ? ORDER BY p.created_at DESC, p.id DESC", authorID)
	if err != nil {
		return nil, fmt.Errorf("list posts of author %d: %w", authorID, err)
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

// Get 按 ID 取文章（含标签），不存在返回 post.ErrNotFound。
func (r *PostRepository) Get(id int64) (*post.Post, error) {
	p, err := scanPost(r.db.QueryRow(postSelect+" WHERE p.id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, post.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get post %d: %w", id, err)
	}
	if err := r.fillTags([]*post.Post{p}); err != nil {
		return nil, err
	}
	return p, nil
}

// fillTags 批量填充文章标签（单查询 IN，按标签名字典序）。
func (r *PostRepository) fillTags(posts []*post.Post) error {
	if len(posts) == 0 {
		return nil
	}
	byID := make(map[int64]*post.Post, len(posts))
	placeholders := make([]string, 0, len(posts))
	args := make([]any, 0, len(posts))
	for _, p := range posts {
		p.Tags = []string{}
		byID[p.ID] = p
		placeholders = append(placeholders, "?")
		args = append(args, p.ID)
	}
	rows, err := r.db.Query(
		`SELECT pt.post_id, t.name FROM post_tags pt JOIN tags t ON t.id = pt.tag_id
		WHERE pt.post_id IN (`+strings.Join(placeholders, ",")+`) ORDER BY t.name`,
		args...,
	)
	if err != nil {
		return fmt.Errorf("list post tags: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var postID int64
		var name string
		if err := rows.Scan(&postID, &name); err != nil {
			return fmt.Errorf("scan post tag: %w", err)
		}
		if p, ok := byID[postID]; ok {
			p.Tags = append(p.Tags, name)
		}
	}
	return rows.Err()
}

// replacePostTags 在事务内重建文章标签关联：标签不存在则先建档（INSERT IGNORE），
// 旧关联全删后按当前标签重插。
func replacePostTags(tx *sql.Tx, postID int64, tags []string) error {
	for _, name := range tags {
		if _, err := tx.Exec("INSERT IGNORE INTO tags (name) VALUES (?)", name); err != nil {
			return err
		}
	}
	if _, err := tx.Exec("DELETE FROM post_tags WHERE post_id = ?", postID); err != nil {
		return err
	}
	for _, name := range tags {
		var tagID int64
		if err := tx.QueryRow("SELECT id FROM tags WHERE name = ?", name).Scan(&tagID); err != nil {
			return err
		}
		if _, err := tx.Exec("INSERT INTO post_tags (post_id, tag_id) VALUES (?, ?)", postID, tagID); err != nil {
			return err
		}
	}
	return nil
}

// Create 新建文章（含标签，同事务）并回填 ID 与时间戳。
func (r *PostRepository) Create(p *post.Post) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("create post: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.Exec(
		"INSERT INTO posts (author_id, title, category, summary, content, cover_url) VALUES (?, ?, ?, ?, ?, ?)",
		p.AuthorID, p.Title, p.Category, p.Summary, p.Content, p.CoverURL,
	)
	if err != nil {
		return fmt.Errorf("create post: %w", err)
	}
	p.ID, err = res.LastInsertId()
	if err != nil {
		return err
	}
	if err := replacePostTags(tx, p.ID, p.Tags); err != nil {
		return fmt.Errorf("create post %d tags: %w", p.ID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("create post %d commit: %w", p.ID, err)
	}
	created, err := r.Get(p.ID)
	if err != nil {
		return err
	}
	*p = *created
	return nil
}

// Update 更新文章字段与标签（同事务），不存在返回 post.ErrNotFound。
func (r *PostRepository) Update(p *post.Post) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("update post %d: %w", p.ID, err)
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.Exec(
		"UPDATE posts SET title = ?, category = ?, summary = ?, content = ?, cover_url = ? WHERE id = ?",
		p.Title, p.Category, p.Summary, p.Content, p.CoverURL, p.ID,
	)
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
	if err := replacePostTags(tx, p.ID, p.Tags); err != nil {
		return fmt.Errorf("update post %d tags: %w", p.ID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("update post %d commit: %w", p.ID, err)
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

// AllTags 返回全部标签名（字典序）。
func (r *PostRepository) AllTags() ([]string, error) {
	rows, err := r.db.Query("SELECT name FROM tags ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer func() { _ = rows.Close() }()
	tags := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, name)
	}
	return tags, rows.Err()
}

// Stats 返回站点统计：文章总数与正文总字符数（CHAR_LENGTH 按字符而非字节）。
func (r *PostRepository) Stats() (int, int, error) {
	var posts, totalChars int
	err := r.db.QueryRow("SELECT COUNT(*), COALESCE(SUM(CHAR_LENGTH(content)), 0) FROM posts").Scan(&posts, &totalChars)
	if err != nil {
		return 0, 0, fmt.Errorf("post stats: %w", err)
	}
	return posts, totalChars, nil
}

// ListFavorites 返回某用户收藏的文章（收藏时间倒序，联表作者与标签）。
func (r *PostRepository) ListFavorites(userID int64) ([]*post.Post, error) {
	rows, err := r.db.Query(
		`SELECT p.id, p.author_id, u.username, p.title, p.category, p.summary, p.content, p.cover_url, p.created_at, p.updated_at
		FROM favorites f
		JOIN posts p ON p.id = f.post_id
		JOIN users u ON u.id = p.author_id
		WHERE f.user_id = ?
		ORDER BY f.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list favorite posts of user %d: %w", userID, err)
	}
	defer func() { _ = rows.Close() }()
	posts := make([]*post.Post, 0)
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, fmt.Errorf("scan favorite post: %w", err)
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := r.fillTags(posts); err != nil {
		return nil, err
	}
	return posts, nil
}

// DailyPublication 近 days 天每日发布篇数与字数（GROUP BY DATE，Go 侧补零，日期升序）。
func (r *PostRepository) DailyPublication(days int) ([]post.DayPublication, error) {
	if days <= 0 {
		days = 365
	}
	rows, err := r.db.Query(
		`SELECT DATE(created_at) AS d, COUNT(*), COALESCE(SUM(CHAR_LENGTH(content)), 0)
		FROM posts WHERE created_at >= DATE_SUB(CURDATE(), INTERVAL ? DAY) GROUP BY d`,
		days-1,
	)
	if err != nil {
		return nil, fmt.Errorf("daily publication: %w", err)
	}
	defer func() { _ = rows.Close() }()
	type cell struct {
		count int
		chars int
	}
	cells := make(map[string]cell)
	for rows.Next() {
		var d time.Time
		var n, chars int
		if err := rows.Scan(&d, &n, &chars); err != nil {
			return nil, fmt.Errorf("scan daily publication: %w", err)
		}
		cells[d.Format("2006-01-02")] = cell{count: n, chars: chars}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]post.DayPublication, 0, days)
	for i := days - 1; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		c := cells[date]
		out = append(out, post.DayPublication{Date: date, Count: c.count, Chars: c.chars})
	}
	return out, nil
}

// CategoryCounts 各分类文章数（字典序）。
func (r *PostRepository) CategoryCounts() ([]post.CategoryCount, error) {
	rows, err := r.db.Query("SELECT category, COUNT(*) FROM posts GROUP BY category ORDER BY category")
	if err != nil {
		return nil, fmt.Errorf("category counts: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make([]post.CategoryCount, 0)
	for rows.Next() {
		var c post.CategoryCount
		if err := rows.Scan(&c.Category, &c.Count); err != nil {
			return nil, fmt.Errorf("scan category count: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CountByCategory 某分类文章数。
func (r *PostRepository) CountByCategory(category string) (int, error) {
	var n int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM posts WHERE category = ?", category).Scan(&n); err != nil {
		return 0, fmt.Errorf("count posts of category %q: %w", category, err)
	}
	return n, nil
}

// UpdateCategory 把某分类全部文章改到目标分类。
func (r *PostRepository) UpdateCategory(from, to string) error {
	if _, err := r.db.Exec("UPDATE posts SET category = ? WHERE category = ?", to, from); err != nil {
		return fmt.Errorf("migrate posts from category %q to %q: %w", from, to, err)
	}
	return nil
}
