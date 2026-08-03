// Package interactionapp 互动用例服务测试：toggle 写优先幂等与并发收敛。
package interactionapp

import (
	"sync"
	"testing"

	"ven_hybird/build/domain/interaction"
)

// fakeRepo 内存实现 interaction.Repository，模拟 MySQL 语句级原子语义：
// AddLike/AddFavorite = INSERT IGNORE（已存在返回"未插入"），Remove = DELETE，
// 每次调用在锁内完成（等价于单条 SQL 的原子性）。
type fakeRepo struct {
	mu        sync.Mutex
	likes     map[likeKey]bool // user 是否已赞某目标
	favorites map[favKey]bool  // user 是否已收藏某文章
}

type likeKey struct {
	targetType interaction.TargetType
	userID     int64
	targetID   int64
}

type favKey struct {
	userID int64
	postID int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		likes:     map[likeKey]bool{},
		favorites: map[favKey]bool{},
	}
}

func (f *fakeRepo) AddLike(userID int64, targetType interaction.TargetType, targetID int64) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := likeKey{targetType, userID, targetID}
	if f.likes[k] {
		return false, nil // INSERT IGNORE 命中重复键
	}
	f.likes[k] = true
	return true, nil
}

func (f *fakeRepo) RemoveLike(userID int64, targetType interaction.TargetType, targetID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.likes, likeKey{targetType, userID, targetID})
	return nil
}

func (f *fakeRepo) IsLiked(userID int64, targetType interaction.TargetType, targetID int64) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.likes[likeKey{targetType, userID, targetID}], nil
}

func (f *fakeRepo) LikeCount(targetType interaction.TargetType, targetID int64) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for k := range f.likes {
		if k.targetType == targetType && k.targetID == targetID {
			n++
		}
	}
	return n, nil
}

func (f *fakeRepo) CountLikes() (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.likes), nil
}

func (f *fakeRepo) PostLikeCounts() (map[int64]int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[int64]int{}
	for k := range f.likes {
		if k.targetType == interaction.TargetPost {
			out[k.targetID]++
		}
	}
	return out, nil
}

func (f *fakeRepo) MomentLikeCounts() (map[int64]int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[int64]int{}
	for k := range f.likes {
		if k.targetType == interaction.TargetMoment {
			out[k.targetID]++
		}
	}
	return out, nil
}

func (f *fakeRepo) LikedTargetIDs(userID int64, targetType interaction.TargetType) ([]int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ids := make([]int64, 0)
	for k := range f.likes {
		if k.userID == userID && k.targetType == targetType {
			ids = append(ids, k.targetID)
		}
	}
	return ids, nil
}

func (f *fakeRepo) AddFavorite(userID, postID int64) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := favKey{userID, postID}
	if f.favorites[k] {
		return false, nil
	}
	f.favorites[k] = true
	return true, nil
}

func (f *fakeRepo) RemoveFavorite(userID, postID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.favorites, favKey{userID, postID})
	return nil
}

func (f *fakeRepo) IsFavorited(userID, postID int64) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.favorites[favKey{userID, postID}], nil
}

func (f *fakeRepo) FavoriteCount(postID int64) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for k := range f.favorites {
		if k.postID == postID {
			n++
		}
	}
	return n, nil
}

func (f *fakeRepo) CountFavorites() (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.favorites), nil
}

func (f *fakeRepo) PostFavoriteCounts() (map[int64]int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[int64]int{}
	for k := range f.favorites {
		out[k.postID]++
	}
	return out, nil
}

// TestToggleLike_ToggleOnOff 串行切换：开→关→开，状态与计数逐次正确。
func TestToggleLike_ToggleOnOff(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)

	liked, count, err := svc.Toggle(1, interaction.TargetPost, 100)
	if err != nil || !liked || count != 1 {
		t.Fatalf("first toggle: liked=%v count=%d err=%v, want liked=true count=1", liked, count, err)
	}
	liked, count, err = svc.Toggle(1, interaction.TargetPost, 100)
	if err != nil || liked || count != 0 {
		t.Fatalf("second toggle: liked=%v count=%d err=%v, want liked=false count=0", liked, count, err)
	}
	liked, count, err = svc.Toggle(1, interaction.TargetPost, 100)
	if err != nil || !liked || count != 1 {
		t.Fatalf("third toggle: liked=%v count=%d err=%v, want liked=true count=1", liked, count, err)
	}

	// moment 与 post 互不干扰
	liked, count, err = svc.Toggle(1, interaction.TargetMoment, 100)
	if err != nil || !liked || count != 1 {
		t.Fatalf("moment toggle: liked=%v count=%d err=%v", liked, count, err)
	}
	postLiked, err := repo.IsLiked(1, interaction.TargetPost, 100)
	if err != nil || !postLiked {
		t.Fatalf("post like should survive moment toggle: liked=%v err=%v", postLiked, err)
	}
}

// TestToggleLike_Concurrent 并发 N 次 toggle 收敛：行集恒 ∈ {0,1}（单用户点赞永不重复），
// 最终状态与计数自洽（IsLiked == count>0）。旧"读-改-写"实现会把单用户计数放大到 N。
// 注：写优先设计不承诺与串行执行同奇偶（并发下个别 DELETE 可能落空——它决策时依赖的行
// 已被别的 DELETE 移除），但绝不产生重复行、状态永不撕裂，这正是本修复的目标。
func TestToggleLike_Concurrent(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)

	const n = 16
	var wg sync.WaitGroup
	errs := make(chan error, n)
	counts := make(chan int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, count, err := svc.Toggle(1, interaction.TargetPost, 100)
			if err != nil {
				errs <- err
				return
			}
			counts <- count
		}()
	}
	wg.Wait()
	close(errs)
	close(counts)
	for err := range errs {
		t.Fatalf("toggle failed: %v", err)
	}
	for count := range counts {
		if count != 0 && count != 1 {
			t.Fatalf("count must stay in {0,1} under concurrency, got %d", count)
		}
	}

	// 收敛断言：行集不膨胀（≤1），状态与计数自洽
	liked, err := repo.IsLiked(1, interaction.TargetPost, 100)
	if err != nil {
		t.Fatalf("IsLiked: %v", err)
	}
	count, err := repo.LikeCount(interaction.TargetPost, 100)
	if err != nil {
		t.Fatalf("LikeCount: %v", err)
	}
	if count != 0 && count != 1 {
		t.Fatalf("final count=%d, want 0 or 1 (no duplicate rows)", count)
	}
	if liked != (count == 1) {
		t.Fatalf("state inconsistent: liked=%v count=%d", liked, count)
	}
}

// TestToggleLike_DoubleClick 双击（2 次并发 toggle）确定收敛为未点赞：
// 恰有一个 INSERT 成功，另一个必走 DELETE 且必然命中（N=2 无双删竞态），与串行双击语义一致。
func TestToggleLike_DoubleClick(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, err := svc.Toggle(1, interaction.TargetPost, 100); err != nil {
				t.Errorf("toggle failed: %v", err)
			}
		}()
	}
	wg.Wait()

	count, err := repo.LikeCount(interaction.TargetPost, 100)
	if err != nil {
		t.Fatalf("LikeCount: %v", err)
	}
	liked, err := repo.IsLiked(1, interaction.TargetPost, 100)
	if err != nil {
		t.Fatalf("IsLiked: %v", err)
	}
	if count != 0 || liked {
		t.Fatalf("double click must converge to unliked, got count=%d liked=%v", count, liked)
	}
}

// TestToggleFavorite_Concurrent 收藏 toggle 与点赞同构：并发下计数恒 ∈ {0,1}，
// 最终状态与计数自洽。
func TestToggleFavorite_Concurrent(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)

	const n = 12
	var wg sync.WaitGroup
	errs := make(chan error, n)
	counts := make(chan int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, count, err := svc.ToggleFavorite(1, 200)
			if err != nil {
				errs <- err
				return
			}
			counts <- count
		}()
	}
	wg.Wait()
	close(errs)
	close(counts)
	for err := range errs {
		t.Fatalf("toggle favorite failed: %v", err)
	}
	for count := range counts {
		if count != 0 && count != 1 {
			t.Fatalf("favorite count must stay in {0,1} under concurrency, got %d", count)
		}
	}

	favorited, err := repo.IsFavorited(1, 200)
	if err != nil {
		t.Fatalf("IsFavorited: %v", err)
	}
	count, err := repo.FavoriteCount(200)
	if err != nil {
		t.Fatalf("FavoriteCount: %v", err)
	}
	if count != 0 && count != 1 {
		t.Fatalf("final count=%d, want 0 or 1 (no duplicate favorites)", count)
	}
	if favorited != (count == 1) {
		t.Fatalf("state inconsistent: favorited=%v count=%d", favorited, count)
	}
}

// TestToggleLike_AlreadyLikedStart 从"已点赞"初始态并发 toggle：行集不膨胀、状态自洽。
func TestToggleLike_AlreadyLikedStart(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	if _, err := repo.AddLike(1, interaction.TargetPost, 100); err != nil {
		t.Fatalf("seed like: %v", err)
	}

	const n = 7
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, err := svc.Toggle(1, interaction.TargetPost, 100); err != nil {
				t.Errorf("toggle failed: %v", err)
			}
		}()
	}
	wg.Wait()

	liked, err := repo.IsLiked(1, interaction.TargetPost, 100)
	if err != nil {
		t.Fatalf("IsLiked: %v", err)
	}
	count, err := repo.LikeCount(interaction.TargetPost, 100)
	if err != nil {
		t.Fatalf("LikeCount: %v", err)
	}
	if count != 0 && count != 1 {
		t.Fatalf("final count=%d, want 0 or 1 (no duplicate rows)", count)
	}
	if liked != (count == 1) {
		t.Fatalf("state inconsistent: liked=%v count=%d", liked, count)
	}
}

// TestFakeRepo_InterfaceSatisfaction 编译期断言：fakeRepo 完整实现 Repository。
var _ interaction.Repository = (*fakeRepo)(nil)
