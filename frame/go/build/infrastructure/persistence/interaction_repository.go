package persistence

import (
	"database/sql"
	"fmt"

	"ven_hybird/build/domain/interaction"
)

// InteractionRepository 是 interaction.Repository 的 MySQL 实现。
type InteractionRepository struct {
	db *sql.DB
}

// NewInteractionRepository 构造互动仓储。
func NewInteractionRepository(db *sql.DB) *InteractionRepository {
	return &InteractionRepository{db: db}
}

// AddLike 点赞（INSERT IGNORE 幂等）。
func (r *InteractionRepository) AddLike(userID int64, targetType interaction.TargetType, targetID int64) error {
	_, err := r.db.Exec(
		"INSERT IGNORE INTO likes (user_id, target_type, target_id) VALUES (?, ?, ?)",
		userID, string(targetType), targetID,
	)
	return err
}

// RemoveLike 取消点赞。
func (r *InteractionRepository) RemoveLike(userID int64, targetType interaction.TargetType, targetID int64) error {
	_, err := r.db.Exec(
		"DELETE FROM likes WHERE user_id = ? AND target_type = ? AND target_id = ?",
		userID, string(targetType), targetID,
	)
	return err
}

// IsLiked 查询用户是否已点赞。
func (r *InteractionRepository) IsLiked(userID int64, targetType interaction.TargetType, targetID int64) (bool, error) {
	var n int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM likes WHERE user_id = ? AND target_type = ? AND target_id = ?",
		userID, string(targetType), targetID,
	).Scan(&n)
	return n > 0, err
}

// LikeCount 目标点赞总数。
func (r *InteractionRepository) LikeCount(targetType interaction.TargetType, targetID int64) (int, error) {
	var n int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM likes WHERE target_type = ? AND target_id = ?",
		string(targetType), targetID,
	).Scan(&n)
	return n, err
}

// AddFavorite 收藏（INSERT IGNORE 幂等）。
func (r *InteractionRepository) AddFavorite(userID, postID int64) error {
	_, err := r.db.Exec("INSERT IGNORE INTO favorites (user_id, post_id) VALUES (?, ?)", userID, postID)
	return err
}

// RemoveFavorite 取消收藏。
func (r *InteractionRepository) RemoveFavorite(userID, postID int64) error {
	_, err := r.db.Exec("DELETE FROM favorites WHERE user_id = ? AND post_id = ?", userID, postID)
	return err
}

// IsFavorited 查询用户是否已收藏。
func (r *InteractionRepository) IsFavorited(userID, postID int64) (bool, error) {
	var n int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM favorites WHERE user_id = ? AND post_id = ?",
		userID, postID,
	).Scan(&n)
	return n > 0, err
}

// FavoriteCount 文章收藏总数。
func (r *InteractionRepository) FavoriteCount(postID int64) (int, error) {
	var n int
	err := r.db.QueryRow("SELECT COUNT(*) FROM favorites WHERE post_id = ?", postID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("favorite count for post %d: %w", postID, err)
	}
	return n, nil
}

// CountLikes 全站点赞总数（后台统计）。
func (r *InteractionRepository) CountLikes() (int, error) {
	var n int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM likes").Scan(&n); err != nil {
		return 0, fmt.Errorf("count likes: %w", err)
	}
	return n, nil
}

// CountFavorites 全站收藏总数（后台统计）。
func (r *InteractionRepository) CountFavorites() (int, error) {
	var n int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM favorites").Scan(&n); err != nil {
		return 0, fmt.Errorf("count favorites: %w", err)
	}
	return n, nil
}
