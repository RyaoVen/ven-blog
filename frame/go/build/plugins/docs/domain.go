// Package docs 插件：树形文档模块（笔记、项目文档），unit-7 设计。
// 本文件为领域层：实体、业务规则与仓储接口（不依赖外层）。
package docs

import (
	"errors"
	"strings"
	"time"
)

// 领域错误（应用层映射 MCP 协议错误）。
var (
	ErrNotFound      = errors.New("doc not found")
	ErrDuplicatePath = errors.New("doc path already exists")
	ErrHasChildren   = errors.New("doc has children")
	ErrInvalidSlug   = errors.New("invalid slug")
	ErrTooDeep       = errors.New("doc tree too deep")
	ErrInvalidKind   = errors.New("invalid kind")
	ErrInvalidStatus = errors.New("invalid status")
	ErrTooManyTags   = errors.New("too many tags")
	ErrTagTooLong    = errors.New("tag too long")
)

// Kind 节点类型：doc = 文档；section = 目录节点（可带 index 正文）。
type Kind string

const (
	KindDoc     Kind = "doc"
	KindSection Kind = "section"
)

// Status 发布状态。
type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
)

// 约束常量（unit-7 §2）。
const (
	MaxSlugLen   = 64
	MaxPathLen   = 512
	MaxTitleLen  = 128
	MaxSummaryLe = 200
	MaxTags      = 8
	MaxTagLen    = 24
	MaxDepth     = 8 // 树深软上限（防御性）
)

// Doc 树形文档节点（文档与目录同表，path 为物化全路径主标识，如 "ai/attention"）。
type Doc struct {
	ID        int64
	ParentID  int64 // 0 = 根级
	Slug      string
	Path      string // 相对根的全路径（不含前导 /），如 "ai/attention"
	Kind      Kind
	Title     string
	Summary   string
	Content   string
	Tags      []string
	SortOrder int
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsSection 是否目录节点。
func (d *Doc) IsSection() bool { return d.Kind == KindSection }

// ValidateSlug 校验 slug：[a-z0-9-]，禁首尾连字符与连续连字符（unit-7 §2）。
func ValidateSlug(slug string) error {
	if slug == "" || len(slug) > MaxSlugLen {
		return ErrInvalidSlug
	}
	prevHyphen := true
	for _, r := range slug {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			prevHyphen = false
		case r == '-':
			if prevHyphen {
				return ErrInvalidSlug
			}
			prevHyphen = true
		default:
			return ErrInvalidSlug
		}
	}
	if prevHyphen {
		return ErrInvalidSlug
	}
	return nil
}

// ValidateTags 校验标签：≤8 个、各 ≤24 字符。
func ValidateTags(tags []string) error {
	if len(tags) > MaxTags {
		return ErrTooManyTags
	}
	for _, t := range tags {
		if len(t) > MaxTagLen {
			return ErrTagTooLong
		}
	}
	return nil
}

// ValidateKind/ValidateStatus 枚举校验。
func ValidateKind(k Kind) error {
	if k != KindDoc && k != KindSection {
		return ErrInvalidKind
	}
	return nil
}

func ValidateStatus(s Status) error {
	if s != StatusDraft && s != StatusPublished {
		return ErrInvalidStatus
	}
	return nil
}

// SplitPath 把全路径拆为段（"ai/attention" → ["ai","attention"]）；空路径返回空段。
func SplitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

// JoinPath 由父路径与 slug 拼全路径（parentID 为根时 parentPath 为空）。
func JoinPath(parentPath, slug string) string {
	if parentPath == "" {
		return slug
	}
	return parentPath + "/" + slug
}

// Depth 计算路径深度（根级 = 1）。
func Depth(path string) int {
	return len(SplitPath(path))
}

// Repository 仓储接口（repo.go 实现 MySQL 版；测试用内存假实现）。
type Repository interface {
	// Create 插入节点（ID/时间戳回填）；path 冲突返回 ErrDuplicatePath。
	Create(doc *Doc) error
	// GetByPath 按全路径取节点，不存在返回 ErrNotFound。
	GetByPath(path string) (*Doc, error)
	// GetByID 按主键取节点，不存在返回 ErrNotFound。
	GetByID(id int64) (*Doc, error)
	// ListChildren 返回直接子节点（sort_order 升序、slug 字典序次之）。
	ListChildren(parentID int64) ([]*Doc, error)
	// ListAll 返回全量节点（tree 构建；sort_order + path 排序）。
	ListAll() ([]*Doc, error)
	// Update 全量更新节点（按 ID），path 冲突返回 ErrDuplicatePath。
	Update(doc *Doc) error
	// Delete 删除单节点。
	Delete(id int64) error
	// CountChildren 直接子节点数。
	CountChildren(id int64) (int, error)
	// ListUpdatedSince 返回 updated_at >= since 的节点（增量拉取，M5）。
	ListUpdatedSince(since time.Time) ([]*Doc, error)
	// UpdatePath 更新节点自身定位（path/parent_id/slug）。
	UpdatePath(id int64, parentID int64, slug, path string, updatedAt time.Time) error
	// RenameDescendants 级联改写后代 path 前缀（oldPrefix → newPrefix；按前缀匹配）。
	RenameDescendants(oldPrefix, newPrefix string, updatedAt time.Time) error
	// DeleteSubtree 事务删除子树（含自身：path 精确 + 前缀匹配一并原子完成）。
	DeleteSubtree(rootPath string) error
}
