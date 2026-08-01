-- 007：用户邮箱与邮箱验证码

-- users.email（可空，唯一索引——MySQL 唯一索引允许多个 NULL）
SET @col := (
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users' AND COLUMN_NAME = 'email'
);
SET @sql := IF(
    @col = 0,
    'ALTER TABLE users ADD COLUMN email VARCHAR(254) NULL AFTER avatar_url',
    'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx := (
    SELECT COUNT(*)
    FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users' AND INDEX_NAME = 'uk_users_email'
);
SET @sql := IF(
    @idx = 0,
    'ALTER TABLE users ADD UNIQUE KEY uk_users_email (email)',
    'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 邮箱验证码（哈希存储，10 分钟有效期，最多尝试 5 次）
CREATE TABLE IF NOT EXISTS email_codes (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    email      VARCHAR(254) NOT NULL,
    code_hash  VARCHAR(128) NOT NULL,
    attempts   INT          NOT NULL DEFAULT 0,
    expires_at DATETIME     NOT NULL,
    created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_email_codes_email (email)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
