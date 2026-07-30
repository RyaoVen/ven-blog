package persistence

import (
	"database/sql"
	"errors"
	"fmt"

	"ven_hybird/build/domain/image"
)

// ImageRepository 是 image.Repository 的 MySQL 实现（BLOB 后端：data 列存二进制）。
type ImageRepository struct {
	db *sql.DB
}

// NewImageRepository 构造图片仓储。
func NewImageRepository(db *sql.DB) *ImageRepository {
	return &ImageRepository{db: db}
}

// Create 新建图片并回填 ID 与时间戳。
func (r *ImageRepository) Create(img *image.Image) error {
	res, err := r.db.Exec(
		"INSERT INTO images (uploader_id, filename, mime, data) VALUES (?, ?, ?, ?)",
		img.UploaderID, img.Filename, img.Mime, img.Data,
	)
	if err != nil {
		return fmt.Errorf("create image: %w", err)
	}
	img.ID, err = res.LastInsertId()
	if err != nil {
		return err
	}
	created, err := r.Get(img.ID)
	if err != nil {
		return err
	}
	*img = *created
	return nil
}

// Get 按 ID 取图片（含二进制数据），不存在返回 image.ErrNotFound。
func (r *ImageRepository) Get(id int64) (*image.Image, error) {
	img := &image.Image{}
	err := r.db.QueryRow(
		"SELECT id, uploader_id, filename, mime, data, created_at FROM images WHERE id = ?", id,
	).Scan(&img.ID, &img.UploaderID, &img.Filename, &img.Mime, &img.Data, &img.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, image.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get image %d: %w", id, err)
	}
	return img, nil
}
