-- 009：UGC 审核三态（approved | pending | rejected）+ 驳回原因（评论与留言板）
-- 幂等条件 ALTER，对齐 003/005/006/007 风格；
-- comments.status 已由 005 提供（VARCHAR(16) 无枚举约束），rejected 值无需改列。

-- 1) guestbook.status：存量行经 DEFAULT 'approved' 自动回填为已通过
SET @col := (
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'guestbook' AND COLUMN_NAME = 'status'
);
SET @sql := IF(
    @col = 0,
    'ALTER TABLE guestbook ADD COLUMN status VARCHAR(16) NOT NULL DEFAULT ''approved'' AFTER content',
    'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 2) guestbook.rejected_reason
SET @col := (
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'guestbook' AND COLUMN_NAME = 'rejected_reason'
);
SET @sql := IF(
    @col = 0,
    'ALTER TABLE guestbook ADD COLUMN rejected_reason VARCHAR(200) NOT NULL DEFAULT '''' AFTER status',
    'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 3) guestbook.status 索引（公开列表与面板按状态过滤）
SET @idx := (
    SELECT COUNT(*)
    FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'guestbook' AND INDEX_NAME = 'idx_guestbook_status'
);
SET @sql := IF(
    @idx = 0,
    'ALTER TABLE guestbook ADD INDEX idx_guestbook_status (status)',
    'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 4) comments.rejected_reason（rejected 状态值写入现有 status 列即可）
SET @col := (
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'comments' AND COLUMN_NAME = 'rejected_reason'
);
SET @sql := IF(
    @col = 0,
    'ALTER TABLE comments ADD COLUMN rejected_reason VARCHAR(200) NOT NULL DEFAULT '''' AFTER status',
    'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
