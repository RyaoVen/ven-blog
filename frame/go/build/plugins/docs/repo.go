package docs

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// OpenDB 插件自建连接（unit-7 §9 决议 1：不走主业务 persistence 包）：
// DSN 优先级 BLOG_DOCS_MYSQL_DSN > BLOG_MYSQL_DSN > 开发默认（与主库同库，部署零新配置）。
// 连接池上限 MaxOpenConns=4（治理规则：防插件数量增长后连接数爆炸）。
func OpenDB() (*sql.DB, error) {
	dsn := os.Getenv("BLOG_DOCS_MYSQL_DSN")
	if dsn == "" {
		dsn = os.Getenv("BLOG_MYSQL_DSN")
	}
	if dsn == "" {
		dsn = "root:root@tcp(127.0.0.1:3306)/ven_blog?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci"
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("docs: open db: %w", err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("docs: ping db: %w", err)
	}
	return db, nil
}

// ensureTable 嵌入式迁移（幂等；自动建库与主业务同策略——DSN 已带库名，库不存在时
// 由驱动侧报错，主业务 Open 已保证库存在，此处不再越权建库）。
func ensureTable(db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS docs (
  id         BIGINT AUTO_INCREMENT PRIMARY KEY,
  parent_id  BIGINT NOT NULL DEFAULT 0,
  slug       VARCHAR(64)  NOT NULL,
  path       VARCHAR(512) NOT NULL,
  kind       ENUM('doc','section') NOT NULL DEFAULT 'doc',
  title      VARCHAR(128) NOT NULL,
  summary    VARCHAR(200) NOT NULL DEFAULT '',
  content    MEDIUMTEXT,
  tags       JSON NULL,
  sort_order INT NOT NULL DEFAULT 0,
  status     ENUM('draft','published') NOT NULL DEFAULT 'published',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE KEY uq_docs_path (path),
  KEY idx_docs_parent (parent_id),
  KEY idx_docs_updated (updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("docs: ensure table: %w", err)
	}
	return nil
}

// DocRepository 是 Repository 的 MySQL 实现（插件自管，不依赖主业务 persistence）。
type DocRepository struct {
	db *sql.DB
}

// NewDocRepository 构造仓储。
func NewDocRepository(db *sql.DB) *DocRepository { return &DocRepository{db: db} }

// docSelect 列序与 scanDoc 一致。
const docSelect = `SELECT id, parent_id, slug, path, kind, title, summary, content, tags, sort_order, status, created_at, updated_at FROM docs`

// scanDoc 行扫描（列序同 docSelect）。
func scanDoc(row interface{ Scan(...any) error }) (*Doc, error) {
	d := &Doc{}
	var tags sql.NullString
	err := row.Scan(&d.ID, &d.ParentID, &d.Slug, &d.Path, &d.Kind, &d.Title, &d.Summary,
		&d.Content, &tags, &d.SortOrder, &d.Status, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if tags.Valid && strings.TrimSpace(tags.String) != "" && tags.String != "null" {
		d.Tags = decodeTags(tags.String)
	} else {
		d.Tags = []string{}
	}
	return d, nil
}

// decodeTags 解析 JSON 标签数组（容错：坏格式返回空数组）。
func decodeTags(raw string) []string {
	tags := []string{}
	_ = json.Unmarshal([]byte(raw), &tags)
	if tags == nil {
		return []string{}
	}
	return tags
}

// encodeTags 序列化标签为 JSON（nil → null）。
func encodeTags(tags []string) any {
	if tags == nil {
		return nil
	}
	b, err := json.Marshal(tags)
	if err != nil {
		return nil
	}
	return string(b)
}

// isDuplicate 判断 MySQL 错误是否唯一键冲突（uq_docs_path）。
func isDuplicate(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

// Create 插入节点（path 冲突 → ErrDuplicatePath）。
func (r *DocRepository) Create(doc *Doc) error {
	res, err := r.db.Exec(
		`INSERT INTO docs (parent_id, slug, path, kind, title, summary, content, tags, sort_order, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		doc.ParentID, doc.Slug, doc.Path, doc.Kind, doc.Title, doc.Summary, doc.Content,
		encodeTags(doc.Tags), doc.SortOrder, doc.Status, doc.CreatedAt, doc.UpdatedAt)
	if isDuplicate(err) {
		return ErrDuplicatePath
	}
	if err != nil {
		return fmt.Errorf("docs: create: %w", err)
	}
	doc.ID, err = res.LastInsertId()
	if err != nil {
		return err
	}
	return nil
}

// GetByPath 按全路径取节点。
func (r *DocRepository) GetByPath(path string) (*Doc, error) {
	d, err := scanDoc(r.db.QueryRow(docSelect+" WHERE path = ?", strings.Trim(path, "/")))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("docs: get by path: %w", err)
	}
	return d, nil
}

// GetByID 按主键取节点。
func (r *DocRepository) GetByID(id int64) (*Doc, error) {
	d, err := scanDoc(r.db.QueryRow(docSelect+" WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("docs: get by id: %w", err)
	}
	return d, nil
}

// ListChildren 直接子节点（sort_order 升序 → slug 字典序）。
func (r *DocRepository) ListChildren(parentID int64) ([]*Doc, error) {
	rows, err := r.db.Query(docSelect+" WHERE parent_id = ? ORDER BY sort_order ASC, slug ASC", parentID)
	if err != nil {
		return nil, fmt.Errorf("docs: list children: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return collectDocs(rows)
}

// ListAll 全量节点（sort_order → path 排序）。
func (r *DocRepository) ListAll() ([]*Doc, error) {
	rows, err := r.db.Query(docSelect + " ORDER BY sort_order ASC, path ASC")
	if err != nil {
		return nil, fmt.Errorf("docs: list all: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return collectDocs(rows)
}

// collectDocs 汇集行集。
func collectDocs(rows *sql.Rows) ([]*Doc, error) {
	docs := make([]*Doc, 0)
	for rows.Next() {
		d, err := scanDoc(rows)
		if err != nil {
			return nil, fmt.Errorf("docs: scan: %w", err)
		}
		docs = append(docs, d)
	}
	return docs, rows.Err()
}

// Update 全量更新（按 ID）。
func (r *DocRepository) Update(doc *Doc) error {
	_, err := r.db.Exec(
		`UPDATE docs SET parent_id=?, slug=?, path=?, kind=?, title=?, summary=?, content=?, tags=?, sort_order=?, status=?, updated_at=? WHERE id=?`,
		doc.ParentID, doc.Slug, doc.Path, doc.Kind, doc.Title, doc.Summary, doc.Content,
		encodeTags(doc.Tags), doc.SortOrder, doc.Status, doc.UpdatedAt, doc.ID)
	if isDuplicate(err) {
		return ErrDuplicatePath
	}
	if err != nil {
		return fmt.Errorf("docs: update: %w", err)
	}
	return nil
}

// Delete 删除单节点。
func (r *DocRepository) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM docs WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("docs: delete: %w", err)
	}
	return nil
}

// ListUpdatedSince 增量拉取（updated_at >= since，升序）。
func (r *DocRepository) ListUpdatedSince(since time.Time) ([]*Doc, error) {
	rows, err := r.db.Query(docSelect+" WHERE updated_at >= ? ORDER BY updated_at ASC", since)
	if err != nil {
		return nil, fmt.Errorf("docs: list since: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return collectDocs(rows)
}

// UpdatePath 更新节点自身定位。
func (r *DocRepository) UpdatePath(id int64, parentID int64, slug, path string, updatedAt time.Time) error {
	_, err := r.db.Exec("UPDATE docs SET parent_id=?, slug=?, path=?, updated_at=? WHERE id=?",
		parentID, slug, path, updatedAt, id)
	if isDuplicate(err) {
		return ErrDuplicatePath
	}
	if err != nil {
		return fmt.Errorf("docs: update path: %w", err)
	}
	return nil
}

// RenameDescendants 级联改写后代 path 前缀（REPLACE 前缀；单条 UPDATE 原子完成）。
func (r *DocRepository) RenameDescendants(oldPrefix, newPrefix string, updatedAt time.Time) error {
	_, err := r.db.Exec(
		"UPDATE docs SET path = CONCAT(?, SUBSTRING(path, ?)), updated_at=? WHERE path LIKE ?",
		newPrefix, len(oldPrefix)+1, updatedAt, oldPrefix+"%")
	if isDuplicate(err) {
		return ErrDuplicatePath
	}
	if err != nil {
		return fmt.Errorf("docs: rename descendants: %w", err)
	}
	return nil
}

// DeleteSubtree 事务删除子树（自身 + 全部前缀后代，单事务原子完成）。
func (r *DocRepository) DeleteSubtree(rootPath string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("docs: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec("DELETE FROM docs WHERE path = ?", rootPath); err != nil {
		return fmt.Errorf("docs: delete root: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM docs WHERE path LIKE CONCAT(?, '/%')", rootPath); err != nil {
		return fmt.Errorf("docs: delete descendants: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("docs: commit: %w", err)
	}
	return nil
}

// CountChildren 直接子节点数。
func (r *DocRepository) CountChildren(id int64) (int, error) {
	var n int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM docs WHERE parent_id = ?", id).Scan(&n); err != nil {
		return 0, fmt.Errorf("docs: count children: %w", err)
	}
	return n, nil
}
