-- 011：访问统计——(date, path) 维度逐日累计 PV
-- 双埋点（Go 网关中间件 + SPA 导航上报）统一落这张表；
-- 复合主键 (date, path) 天然幂等，同日内同路径重复访问走 upsert 累加。
-- CREATE TABLE IF NOT EXISTS 幂等，对齐 001 风格。

CREATE TABLE IF NOT EXISTS visits (
    date  DATE          NOT NULL,
    path  VARCHAR(255)  NOT NULL,
    count INT UNSIGNED  NOT NULL DEFAULT 0,
    PRIMARY KEY (date, path)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
