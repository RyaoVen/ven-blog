-- 005：设置存储 + 评论审核状态

CREATE TABLE IF NOT EXISTS settings (
    `key` VARCHAR(64) NOT NULL,
    value  TEXT NOT NULL,
    PRIMARY KEY (`key`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;

-- comments.status（approved | pending），幂等条件 ALTER
SET @col := (
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'comments' AND COLUMN_NAME = 'status'
);
SET @sql := IF(
    @col = 0,
    'ALTER TABLE comments ADD COLUMN status VARCHAR(16) NOT NULL DEFAULT ''approved'' AFTER reply_to',
    'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx := (
    SELECT COUNT(*)
    FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'comments' AND INDEX_NAME = 'idx_comments_status'
);
SET @sql := IF(
    @idx = 0,
    'ALTER TABLE comments ADD INDEX idx_comments_status (status)',
    'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
