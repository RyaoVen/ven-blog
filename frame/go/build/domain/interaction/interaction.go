// Package interaction 互动聚合：点赞与收藏。
package interaction

import "time"

// TargetType 互动目标类型（点赞支持文章/动态，收藏仅文章）。
type TargetType string

const (
	TargetPost   TargetType = "post"
	TargetMoment TargetType = "moment"
)

// Like 点赞记录。
type Like struct {
	UserID     int64
	TargetType TargetType
	TargetID   int64
	CreatedAt  time.Time
}

// Favorite 收藏记录。
type Favorite struct {
	UserID    int64
	PostID    int64
	CreatedAt time.Time
}

// Status 某用户对某文章的互动状态。
type Status struct {
	Liked         bool
	Favorited     bool
	LikeCount     int
	FavoriteCount int
}
