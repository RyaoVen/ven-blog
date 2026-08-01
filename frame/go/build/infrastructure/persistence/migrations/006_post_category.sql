-- 006：文章分类（编辑器必选，默认「随笔」），幂等条件 ALTER

SET @col := (
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'posts' AND COLUMN_NAME = 'category'
);
SET @sql := IF(
    @col = 0,
    'ALTER TABLE posts ADD COLUMN category VARCHAR(64) NOT NULL DEFAULT ''随笔'' AFTER title',
    'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
