-- 012：文章与动态置顶——posts/moments 增加 pinned（0 普通 / 1 置顶）。
-- 置顶是加分排序（pinned DESC, created_at DESC），不影响既有数据语义；
-- 默认 0，老数据迁移后全部为普通。幂等条件 ALTER，对齐 009/010 风格。

-- 1) posts.pinned
SET @col := (
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'posts' AND COLUMN_NAME = 'pinned'
);
SET @sql := IF(
    @col = 0,
    'ALTER TABLE posts ADD COLUMN pinned TINYINT(1) NOT NULL DEFAULT 0 AFTER cover_url',
    'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 2) moments.pinned
SET @col := (
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'moments' AND COLUMN_NAME = 'pinned'
);
SET @sql := IF(
    @col = 0,
    'ALTER TABLE moments ADD COLUMN pinned TINYINT(1) NOT NULL DEFAULT 0 AFTER content',
    'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
