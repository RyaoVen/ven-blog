# 线上部署操作手册（基于 2026-09-16 技术测评报告修复方案）

> 服务器：101.35.48.19（nginx/1.18.0 Ubuntu，当前纯 HTTP）
> 范围：nginx 补全（gzip/安全头）+ 部署对齐（最新 master + BLOG_SITE_URL）+ HTTPS 后置清单（域名就绪后执行）
> 约定：`/etc/nginx/sites-available/ven-blog.conf` 为站点配置文件名（按实际调整）

## 第 1 步：备份与检查

```bash
cp /etc/nginx/sites-available/ven-blog.conf ~/ven-blog.conf.bak.$(date +%F)
nginx -t && systemctl reload nginx   # 先确认当前配置可正常重载
```

## 第 2 步：nginx 补全（gzip + 安全响应头）

在站点 `server { }` 块内追加：

```nginx
# ---- gzip（报告 #5：默认只压 text/html，730KB JS 未压缩）----
gzip on;
gzip_comp_level 5;
gzip_min_length 1024;
gzip_vary on;
gzip_types application/javascript text/css application/json image/svg+xml application/manifest+json;

# ---- 安全响应头（报告 #12）----
add_header X-Content-Type-Options nosniff always;
add_header X-Frame-Options SAMEORIGIN always;
add_header Referrer-Policy strict-origin-when-cross-origin always;
add_header Permissions-Policy "camera=(), microphone=(), geolocation=()" always;
# HTTPS 就绪后追加（第 5 步）：
# add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
```

重载并验证：

```bash
nginx -t && systemctl reload nginx
curl -sI http://101.35.48.19/assets/entry-client.js | grep -iE "content-encoding|x-content-type"
# 期望：content-encoding: gzip
```

## 第 3 步：部署对齐（报告「补充观察」：线上落后于仓库）

```bash
# 服务器上（或本地构建后上传）
cd /opt/ven-blog && git pull origin master   # 按实际部署目录调整

# Node 侧
cd frame/node && npm ci && npm run build

# Go 侧
cd ../go && go build -o /opt/ven-blog/bin/ven-blog .

# env 关键项（.env.local 或 systemd Environment=）
BLOG_SITE_URL=https://blog.example.com     # 或 http://101.35.48.19（HTTPS 前过渡）
BLOG_MYSQL_DSN=...                         # 既有
VEN_INTERNAL_TOKEN=...                     # 既有
BLOG_AUTHOR_PASSWORD=...                   # 既有

# 重启（tools/deploy 亦可：./deploy restart）
systemctl restart ven-blog-node ven-blog-go
```

**验证清单（对表执行）**：

| 项 | 期望 |
|---|---|
| `curl -s http://127.0.0.1:8080/rss.xml \| grep -c 127.0.0.1` | 0（RSS 域名修复生效） |
| `curl -sI http://.../favicon.ico` | 200 |
| `curl -s http://.../robots.txt` | 含 Sitemap 行 |
| `curl -s "http://.../api/posts?size=2"` | JSON 含 total/page/size，响应 ≤30KB |
| `curl -s "http://.../api/posts?page=abc"` | 400 |
| `POST /auth/login`（错误凭证） | 401（不是 500） |
| `curl -s http://.../healthz` | 200 |
| `/guestbook`、`/admin` | 200（模块随部署对齐上线） |

## 第 4 步（后置）：HTTPS + HTTP/2（域名就绪后）

```bash
apt install certbot python3-certbot-nginx
certbot --nginx -d blog.example.com        # 自动改 nginx + 续期定时器
# nginx server 块追加 HSTS（确认 HTTPS 稳定运行一周后）：
# add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
# listen 443 ssl http2; 已由 certbot 写入，确认 http2 关键字存在
```

## 回滚

```bash
cp ~/ven-blog.conf.bak.<date> /etc/nginx/sites-available/ven-blog.conf && nginx -t && systemctl reload nginx
# 应用回滚：切回上一构建产物目录 / 上一 git tag 重新 build
```
