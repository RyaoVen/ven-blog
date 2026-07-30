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
	liked, err = s.repo.IsLiked(userID, interaction.TargetPost, postID)
	if err != nil {
		return false, 0, err
	}
	if liked {
		if err := s.repo.RemoveLike(userID, interaction.TargetPost, postID); err != nil {
			return false, 0, err
		}
		liked = false
	} else {
		if err := s.repo.AddLike(userID, interaction.TargetPost, postID); err != nil {
			return false, 0, err
		}
		liked = true
	}
	count, err = s.repo.LikeCount(interaction.TargetPost, postID)
	if err != nil {
		return false, 0, err
	}
	return liked, count, nil
}

// ToggleFavorite 切换文章收藏状态，返回最新状态与总数。
func (s *Service) ToggleFavorite(userID, postID int64) (favorited bool, count int, err error) {
	favorited, err = s.repo.IsFavorited(userID, postID)
	if err != nil {
		return false, 0, err
	}
	if favorited {
		if err := s.repo.RemoveFavorite(userID, postID); err != nil {
			return false, 0, err
		}
		favorited = false
	} else {
		if err := s.repo.AddFavorite(userID, postID); err != nil {
			return false, 0, err
		}
		favorited = true
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
