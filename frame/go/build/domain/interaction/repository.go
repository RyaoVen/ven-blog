package interaction

// Repository 互动仓储接口（领域层定义，基础设施层实现）。
type Repository interface {
	// AddLike 点赞（INSERT IGNORE 幂等）。返回是否新插入：
	// true = 本次插入成功（原本未点赞）；false = 已存在（重复键被忽略）。
	AddLike(userID int64, targetType TargetType, targetID int64) (bool, error)
	// RemoveLike 取消点赞。
	RemoveLike(userID int64, targetType TargetType, targetID int64) error
	// IsLiked 查询用户是否已点赞。
	IsLiked(userID int64, targetType TargetType, targetID int64) (bool, error)
	// LikeCount 目标点赞总数。
	LikeCount(targetType TargetType, targetID int64) (int, error)
	// CountLikes 全站点赞总数（后台统计）。
	CountLikes() (int, error)
	// PostLikeCounts 各文章点赞数分组统计（后台文章列表用，一次查询出全表 map）。
	PostLikeCounts() (map[int64]int, error)
	// MomentLikeCounts 各动态点赞数分组统计（target_type='moment'）。
	MomentLikeCounts() (map[int64]int, error)
	// LikedTargetIDs 某用户点赞过的目标 ID 列表（viewer 状态下发用）。
	LikedTargetIDs(userID int64, targetType TargetType) ([]int64, error)

	// AddFavorite 收藏（INSERT IGNORE 幂等）。返回是否新插入。
	AddFavorite(userID, postID int64) (bool, error)
	// RemoveFavorite 取消收藏。
	RemoveFavorite(userID, postID int64) error
	// IsFavorited 查询用户是否已收藏。
	IsFavorited(userID, postID int64) (bool, error)
	// FavoriteCount 文章收藏总数。
	FavoriteCount(postID int64) (int, error)
	// CountFavorites 全站收藏总数（后台统计）。
	CountFavorites() (int, error)
	// PostFavoriteCounts 各文章收藏数分组统计（后台文章列表用，一次查询出全表 map）。
	PostFavoriteCounts() (map[int64]int, error)
}
