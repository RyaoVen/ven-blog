// Package persistence 基础设施层：MySQL 连接、自动迁移与种子数据。
package persistence

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"

	"github.com/go-sql-driver/mysql"
)

// defaultDSN 是开发默认 DSN（生产/本地开发一律用 BLOG_MYSQL_DSN 覆盖，不要把真实密码写进代码）。
const defaultDSN = "root:root@tcp(127.0.0.1:3306)/ven_blog?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci"

//go:embed migrations/001_init.sql
var migration001 string

//go:embed migrations/002_subscribers.sql
var migration002 string

// migrations 按序执行的迁移脚本（文件名即顺序）。
var migrations = []string{migration001, migration002}

// DSNFromEnv 读取 BLOG_MYSQL_DSN，未设置回退开发默认值。
func DSNFromEnv() string {
	if dsn := os.Getenv("BLOG_MYSQL_DSN"); dsn != "" {
		return dsn
	}
	return defaultDSN
}

// Open 打开 MySQL 连接：数据库不存在时自动创建，随后执行幂等迁移。
func Open(dsn string) (*sql.DB, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse BLOG_MYSQL_DSN: %w", err)
	}
	if cfg.DBName == "" {
		return nil, fmt.Errorf("BLOG_MYSQL_DSN must include database name (e.g. /ven_blog)")
	}
	if err := ensureDatabase(cfg); err != nil {
		return nil, err
	}
	// 迁移需要多语句执行
	cfg.MultiStatements = true
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("run migration: %w", err)
		}
	}
	return db, nil
}

// ensureDatabase 以无库名连接执行 CREATE DATABASE IF NOT EXISTS。
func ensureDatabase(cfg *mysql.Config) error {
	bare := *cfg
	bare.DBName = ""
	db, err := sql.Open("mysql", bare.FormatDSN())
	if err != nil {
		return fmt.Errorf("open mysql (server): %w", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_unicode_ci", cfg.DBName)); err != nil {
		return fmt.Errorf("create database %q: %w", cfg.DBName, err)
	}
	return nil
}
