-- 008：API 访问密钥（程序化鉴权凭据，agent 调用网关用）
-- 服务端只存 sha256 哈希，明文仅在创建响应中出现一次；吊销即终态，不可恢复

CREATE TABLE IF NOT EXISTS api_keys (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id      BIGINT UNSIGNED NOT NULL,
    name         VARCHAR(64)     NOT NULL COMMENT '用途备注，如 zcode-agent',
    key_hash     CHAR(64)        NOT NULL COMMENT 'sha256(明文) 十六进制，唯一检索键',
    prefix       VARCHAR(16)     NOT NULL COMMENT '明文前 8 位（如 ven_ab12），列表展示用，不可还原',
    created_at   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at DATETIME        NULL COMMENT '最近一次鉴权成功时间，NULL=从未使用',
    revoked_at   DATETIME        NULL COMMENT '吊销时间，非 NULL=终态失效',
    PRIMARY KEY (id),
    UNIQUE KEY uk_api_keys_hash (key_hash),
    KEY idx_api_keys_user (user_id),
    CONSTRAINT fk_api_keys_user FOREIGN KEY (user_id) REFERENCES users (id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
