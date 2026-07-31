-- 003：评论回复目标（@ 形式平铺回复，不嵌套）
-- MySQL 的 ALTER ADD COLUMN 没有 IF NOT EXISTS，用 information_schema 条件执行保证幂等

SET @col := (
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'comments' AND COLUMN_NAME = 'reply_to'
);
SET @sql := IF(
    @col = 0,
    'ALTER TABLE comments ADD COLUMN reply_to VARCHAR(64) NOT NULL DEFAULT '''' AFTER content',
    'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
