package persistence

import (
	"errors"
	"testing"

	"ven_hybird/build/domain/interaction"

	"github.com/DATA-DOG/go-sqlmock"
)

// TestInteractionRepository_AddLikeRowsAffected INSERT IGNORE 的 RowsAffected
// 决定返回值：1 = 新插入（原本未点赞），0 = 重复键被忽略（原本已点赞）。
func TestInteractionRepository_AddLikeRowsAffected(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()
	repo := NewInteractionRepository(db)

	for _, tc := range []struct {
		name     string
		affected int64
		want     bool
	}{
		{name: "inserted", affected: 1, want: true},
		{name: "duplicate ignored", affected: 0, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mock.ExpectExec("INSERT IGNORE INTO likes").WillReturnResult(sqlmock.NewResult(1, tc.affected))
			got, err := repo.AddLike(1, interaction.TargetPost, 2)
			if err != nil {
				t.Fatalf("AddLike: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet expectations: %v", err)
			}
		})
	}
}

// TestInteractionRepository_AddLikeExecError 执行失败：返回错误且不误报"未插入"。
func TestInteractionRepository_AddLikeExecError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()
	repo := NewInteractionRepository(db)

	execErr := errors.New("connection lost")
	mock.ExpectExec("INSERT IGNORE INTO likes").WillReturnError(execErr)
	got, err := repo.AddLike(1, interaction.TargetPost, 2)
	if !errors.Is(err, execErr) {
		t.Fatalf("expected exec error, got %v", err)
	}
	if got {
		t.Fatal("must not report inserted on exec error")
	}
}

// TestInteractionRepository_AddFavoriteRowsAffected 收藏同构：RowsAffected 决定是否新插入。
func TestInteractionRepository_AddFavoriteRowsAffected(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()
	repo := NewInteractionRepository(db)

	for _, tc := range []struct {
		name     string
		affected int64
		want     bool
	}{
		{name: "inserted", affected: 1, want: true},
		{name: "duplicate ignored", affected: 0, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mock.ExpectExec("INSERT IGNORE INTO favorites").WillReturnResult(sqlmock.NewResult(1, tc.affected))
			got, err := repo.AddFavorite(1, 2)
			if err != nil {
				t.Fatalf("AddFavorite: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet expectations: %v", err)
			}
		})
	}
}

// TestInteractionRepository_RemoveLikeIdempotent DELETE 幂等：行不存在不算错误。
func TestInteractionRepository_RemoveLikeIdempotent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()
	repo := NewInteractionRepository(db)

	// 模拟 DELETE 命中 0 行（用户本就没点赞）：必须返回 nil
	mock.ExpectExec("DELETE FROM likes").WillReturnResult(sqlmock.NewResult(0, 0))
	if err := repo.RemoveLike(1, interaction.TargetPost, 2); err != nil {
		t.Fatalf("RemoveLike on missing row must be nil, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
