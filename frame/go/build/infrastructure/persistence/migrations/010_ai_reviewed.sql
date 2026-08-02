-- 010：AI 审核队列拆分——comments/guestbook 增加 ai_reviewed_at（NULL = AI 未判）。
-- worker 只拉未判 pending；AI 判 uncertain 后打标（留 pending 交人工），不再重复提交 LLM；
-- 判定失败不打标，下轮自动重试。幂等条件 ALTER，对齐 009 风格。

-- 1) comments.ai_reviewed_at
SET @col := (
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'comments' AND COLUMN_NAME = 'ai_reviewed_at'
);
SET @sql := IF(
    @col = 0,
    'ALTER TABLE comments ADD COLUMN ai_reviewed_at DATETIME NULL AFTER rejected_reason',
    'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 2) guestbook.ai_reviewed_at
SET @col := (
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'guestbook' AND COLUMN_NAME = 'ai_reviewed_at'
);
SET @sql := IF(
    @col = 0,
    'ALTER TABLE guestbook ADD COLUMN ai_reviewed_at DATETIME NULL AFTER rejected_reason',
    'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
