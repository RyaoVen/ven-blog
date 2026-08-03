// Package interactionapp 互动用例服务：点赞/收藏的切换与状态查询。
package interactionapp

import (
	"ven_hybird/build/domain/interaction"
)

// Service 互动用例服务。
type Service struct {
	repo interaction.Repository
}

// NewService 构造互动用例服务。
func NewService(repo interaction.Repository) *Service {
	return &Service{repo: repo}
}

// ToggleLike 切换文章点赞状态，返回最新状态与总数。
func (s *Service) ToggleLike(userID, postID int64) (liked bool, count int, err error) {
	return s.Toggle(userID, interaction.TargetPost, postID)
}

// Toggle 切换点赞状态（post/moment 通用），返回最新状态与总数。
// 写优先幂等：INSERT IGNORE 后看 RowsAffected——1 行=本次插入=liked；
// 0 行=已存在→DELETE（幂等）=unliked。无"读-改-写"窗口，并发连点按串行翻转收敛。
func (s *Service) Toggle(userID int64, targetType interaction.TargetType, targetID int64) (liked bool, count int, err error) {
	inserted, err := s.repo.AddLike(userID, targetType, targetID)
	if err != nil {
		return false, 0, err
	}
	liked = inserted
	if !inserted {
		// 原本已点赞：取消（DELETE 幂等）
		if err := s.repo.RemoveLike(userID, targetType, targetID); err != nil {
			return false, 0, err
		}
	}
	count, err = s.repo.LikeCount(targetType, targetID)
	if err != nil {
		return false, 0, err
	}
	return liked, count, nil
}

// MomentLikeCounts 各动态点赞数分组统计（/moments 页展示用）。
func (s *Service) MomentLikeCounts() (map[int64]int, error) {
	return s.repo.MomentLikeCounts()
}

// PostLikeCounts 各文章点赞数分组统计（后台文章列表用，一次查询避免 N+1）。
func (s *Service) PostLikeCounts() (map[int64]int, error) {
	return s.repo.PostLikeCounts()
}

// ViewerMomentLikes 用户点赞过的动态 ID（viewer 状态下发用）。
func (s *Service) ViewerMomentLikes(userID int64) ([]int64, error) {
	return s.repo.LikedTargetIDs(userID, interaction.TargetMoment)
}

// ToggleFavorite 切换文章收藏状态，返回最新状态与总数。
// 写优先幂等：与 Toggle 同构（INSERT IGNORE 看 RowsAffected，未插入则 DELETE）。
func (s *Service) ToggleFavorite(userID, postID int64) (favorited bool, count int, err error) {
	inserted, err := s.repo.AddFavorite(userID, postID)
	if err != nil {
		return false, 0, err
	}
	favorited = inserted
	if !inserted {
		// 原本已收藏：取消（DELETE 幂等）
		if err := s.repo.RemoveFavorite(userID, postID); err != nil {
			return false, 0, err
		}
	}
	count, err = s.repo.FavoriteCount(postID)
	if err != nil {
		return false, 0, err
	}
	return favorited, count, nil
}

// Counts 文章的公开计数（ISR 页面数据用，不含 viewer 状态）。
func (s *Service) Counts(postID int64) (likeCount, favoriteCount int, err error) {
	likeCount, err = s.repo.LikeCount(interaction.TargetPost, postID)
	if err != nil {
		return 0, 0, err
	}
	favoriteCount, err = s.repo.FavoriteCount(postID)
	if err != nil {
		return 0, 0, err
	}
	return likeCount, favoriteCount, nil
}

// ViewerStatus 某用户对文章的互动状态（登录用户 data-only 查询用）。
func (s *Service) ViewerStatus(userID, postID int64) (*interaction.Status, error) {
	liked, err := s.repo.IsLiked(userID, interaction.TargetPost, postID)
	if err != nil {
		return nil, err
	}
	favorited, err := s.repo.IsFavorited(userID, postID)
	if err != nil {
		return nil, err
	}
	likeCount, favoriteCount, err := s.Counts(postID)
	if err != nil {
		return nil, err
	}
	return &interaction.Status{
		Liked:         liked,
		Favorited:     favorited,
		LikeCount:     likeCount,
		FavoriteCount: favoriteCount,
	}, nil
}

// Totals 全站点赞/收藏总数（后台统计）。
func (s *Service) Totals() (likes int, favorites int, err error) {
	likes, err = s.repo.CountLikes()
	if err != nil {
		return 0, 0, err
	}
	favorites, err = s.repo.CountFavorites()
	if err != nil {
		return 0, 0, err
	}
	return likes, favorites, nil
}

// PostFavoriteCounts 各文章收藏数分组统计（后台文章列表用，一次查询避免 N+1）。
func (s *Service) PostFavoriteCounts() (map[int64]int, error) {
	return s.repo.PostFavoriteCounts()
}
