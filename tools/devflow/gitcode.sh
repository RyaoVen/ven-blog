#!/usr/bin/env bash
# =============================================================================
# VenDevFlow · GitCode OpenAPI 封装
# 文档: https://docs.gitcode.com/docs/apis/  (Base: /api/v5, Gitee 风格)
# 依赖: curl + jq；配置: 本目录 devflow.env 或同名环境变量
# 用法: gitcode.sh <子命令> [参数...]   详细帮助: gitcode.sh help
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ---------- 配置加载（env 文件优先级低于已导出的环境变量） ----------
if [[ -f "$SCRIPT_DIR/devflow.env" ]]; then
  while IFS= read -r _line || [[ -n "$_line" ]]; do
    [[ -z "$_line" || "$_line" == \#* ]] && continue
    [[ "$_line" =~ ^[A-Za-z_][A-Za-z0-9_]*= ]] && export "$_line"
  done < "$SCRIPT_DIR/devflow.env"
  unset _line
fi

GITCODE_API="${GITCODE_API:-https://api.gitcode.com/api/v5}"
GITCODE_MAIN="${GITCODE_MAIN:-main}"

die() { echo "[gitcode.sh] ERROR: $*" >&2; exit 1; }
usage() { sed -n '2,6p' "$0"; echo "子命令:"; grep -E '^  [a-z-]+\)' "$0" | sed -E 's/\).*//;s/^ +//' | column -t 2>/dev/null || cat; }
require_env() {  # 仅在真正调 API 时校验，help 无需配置
  [[ -n "${GITCODE_OWNER:-}" ]] || die "缺少 GITCODE_OWNER（组织或个人 path）"
  [[ -n "${GITCODE_REPO:-}" ]] || die "缺少 GITCODE_REPO（仓库路径）"
  [[ -n "${GITCODE_TOKEN:-}" ]] || die "缺少 GITCODE_TOKEN（经典 PAT: gitcode.com/setting/token-classic）"
}

# ---------- 通用请求 ----------
# gc_req <METHOD> <PATH> [JSON_BODY]
gc_req() {
  require_env
  local method="$1" path="$2" body="${3:-}" resp http_code
  local args=(-sS -w '\n%{http_code}' -X "$method" \
    "$GITCODE_API$path" \
    -H "Authorization: Bearer $GITCODE_TOKEN" \
    -H "Content-Type: application/json")
  [[ -n "$body" ]] && args+=(-d "$body")
  resp="$(curl --max-time 30 "${args[@]}")" || die "网络请求失败: $method $path"
  http_code="$(tail -n1 <<<"$resp")"
  if [[ "$http_code" -lt 200 || "$http_code" -ge 300 ]]; then
    die "API $method $path -> HTTP $http_code: $(sed '$d' <<<"$resp" | head -c 500)"
  fi
  sed '$d' <<<"$resp"
}

# 从 stdin 或 "@"文件 读取正文：body_arg "-" 表示读 stdin，"@path" 表示读文件
read_body() {
  local arg="$1"
  if [[ "$arg" == "-" ]]; then cat
  elif [[ "$arg" == @* ]]; then cat "${arg#@}"
  else printf '%s' "$arg"
  fi
}

# ---------- 子命令 ----------

cmd_doctor() {  # 验证令牌与仓库可达
  echo "== 身份 =="
  gc_req GET /user | jq -r '"login: \(.login // .login_name)  name: \(.name)"'
  echo "== 仓库 $GITCODE_OWNER/$GITCODE_REPO =="
  gc_req GET "/repos/$GITCODE_OWNER/$GITCODE_REPO" | jq -r '"path: \(.path)  默认分支: \(.default_branch // "?")  私有: \(.private)"'
  echo "OK"
}

cmd_issue_create() {  # issue-create <title> <body(-|@file)> [labels] [assignee]
  local title="$1" body labels="${3:-}" assignee="${4:-}"
  body="$(read_body "$2")"
  local json; json="$(jq -n \
    --arg repo "$GITCODE_REPO" --arg title "$title" --arg body "$body" \
    --arg labels "$labels" --arg assignee "$assignee" \
    '{repo:$repo, title:$title, body:$body}
     + (if $labels   != "" then {labels:$labels}   else {} end)
     + (if $assignee != "" then {assignee:$assignee} else {} end)')"
  gc_req POST "/repos/$GITCODE_OWNER/issues" "$json" \
    | jq '{number, html_url, state, title}'
}

cmd_issue_get() {  # issue-get <number>
  gc_req GET "/repos/$GITCODE_OWNER/$GITCODE_REPO/issues/$1" \
    | jq '{number, state, title, labels: [.labels[].name], body}'
}

cmd_issue_list() {  # issue-list [state=open|closed|all] [labels]
  local state="${1:-open}" labels="${2:-}" q=""
  [[ "$state" != "open" ]] && q="?state=$state"
  [[ -n "$labels" ]] && q="${q:+$q&}?labels=$labels"
  gc_req GET "/repos/$GITCODE_OWNER/$GITCODE_REPO/issues$q" \
    | jq -r '.[] | "#\(.number) [\(.state)] \(.title)  {\([.labels[].name] | join(","))}"'
}

cmd_issue_comment() {  # issue-comment <number> <body(-|@file)>
  local body; body="$(read_body "$2")"
  local json; json="$(jq -n --arg body "$body" '{body:$body}')"
  gc_req POST "/repos/$GITCODE_OWNER/$GITCODE_REPO/issues/$1/comments" "$json" \
    | jq '{id, html_url, created_at}'
}

cmd_issue_close() {  # issue-close <number>  JSON body 且 repo/title 必填，state=close
  local n="$1"
  local title; title="$(gc_req GET "/repos/$GITCODE_OWNER/$GITCODE_REPO/issues/$n" | jq -r '.title')"
  [[ -n "$title" && "$title" != "null" ]] || die "无法读取 Issue #$n 标题"
  local json; json="$(jq -n --arg repo "$GITCODE_REPO" --arg title "$title" \
    '{repo:$repo, title:$title, state:"close"}')"
  gc_req PATCH "/repos/$GITCODE_OWNER/issues/$n" "$json" \
    | jq '{number, state, html_url}'
}

cmd_pr_create() {  # pr-create <title> <head> <base> <body(-|@file)>
  local title="$1" head="$2" base="$3" body
  body="$(read_body "$4")"
  local json; json="$(jq -n --arg t "$title" --arg h "$head" --arg b "$base" --arg body "$body" \
    '{title:$t, head:$h, base:$b, body:$body}')"
  gc_req POST "/repos/$GITCODE_OWNER/$GITCODE_REPO/pulls" "$json" \
    | jq '{number, html_url, state, title}'
}

cmd_pr_get() {  # pr-get <number>
  gc_req GET "/repos/$GITCODE_OWNER/$GITCODE_REPO/pulls/$1" \
    | jq '{number, state, title, head: .head.ref, base: .base.ref, mergeable, html_url}'
}

cmd_pr_list() {  # pr-list [state]
  local q="?state=${1:-open}"
  gc_req GET "/repos/$GITCODE_OWNER/$GITCODE_REPO/pulls$q" \
    | jq -r '.[] | "#\(.number) [\(.state)] \(.title)  \(.head.ref -> \(.base.ref))"'
}

cmd_pr_comment() {  # pr-comment <number> <body(-|@file)>
  local body; body="$(read_body "$2")"
  local json; json="$(jq -n --arg body "$body" '{body:$body}')"
  gc_req POST "/repos/$GITCODE_OWNER/$GITCODE_REPO/pulls/$1/comments" "$json" \
    | jq '{id, html_url, created_at}'
}

cmd_pr_link() {  # pr-link <pr_number> <issue_numbers 逗号分隔>  body 为裸整型数组
  local body; body="$(jq -cn --arg n "$2" '$n | split(",") | map(tonumber)')"
  gc_req POST "/repos/$GITCODE_OWNER/$GITCODE_REPO/pulls/$1/issues" "$body" \
    | jq 'if type=="array" then [.[] | {number, state}] else . end'
}

cmd_pr_merge() {  # pr-merge <number> [merge_method=merge|rebase|squash]
  local method="${2:-squash}"
  local json; json="$(jq -n --arg m "$method" '{merge_method:$m}')"
  echo "合并 PR #$1 (method=$method) ..."
  gc_req PUT "/repos/$GITCODE_OWNER/$GITCODE_REPO/pulls/$1/merge" "$json"
  echo ""
}

cmd_branch_create() {  # branch-create <branch_name> [ref=main]
  local ref="${2:-$GITCODE_MAIN}"
  local json; json="$(jq -n --arg b "$1" --arg r "$ref" \
    '{branch_name:$b, refs:"heads/\($r)"}')"
  gc_req POST "/repos/$GITCODE_OWNER/$GITCODE_REPO/branches" "$json" \
    | jq '{name, commit: .commit.sha[0:8]}'
}

cmd_milestone_list() {  # milestone-list [state=open|closed|all]
  gc_req GET "/repos/$GITCODE_OWNER/$GITCODE_REPO/milestones?state=${1:-open}" \
    | jq -r '.[] | "#\(.number) [\(.state // "?")] \(.title)  due: \(.due_on // "-")"'
}

cmd_milestone_get() {  # milestone-get <number>
  gc_req GET "/repos/$GITCODE_OWNER/$GITCODE_REPO/milestones/$1"
}

cmd_milestone_create() {  # milestone-create <title> <description(-|@file)> [due YYYY-MM-DD 或 ISO]
  local desc; desc="$(read_body "$2")"
  local due="${3:-}"
  [[ "$due" == *T* ]] && due="${due%%T*}"   # ISO8601 → 纯日期（GitCode 仅接受 YYYY-MM-DD）
  local json; json="$(jq -n --arg t "$1" --arg d "$desc" --arg due "$due" \
    '{title:$t, description:$d, due_on:$due}')"
  gc_req POST "/repos/$GITCODE_OWNER/$GITCODE_REPO/milestones" "$json" \
    | jq '{number, title, due: (.due_date // .due_on), html_url}'
}

# ---------- 入口 ----------
cmd="${1:-help}"; shift || true
case "$cmd" in
  doctor)        cmd_doctor "$@" ;;
  issue-create)  cmd_issue_create "$@" ;;
  issue-get)     cmd_issue_get "$@" ;;
  issue-list)    cmd_issue_list "$@" ;;
  issue-comment) cmd_issue_comment "$@" ;;
  issue-close)   cmd_issue_close "$@" ;;
  pr-create)     cmd_pr_create "$@" ;;
  pr-get)        cmd_pr_get "$@" ;;
  pr-list)       cmd_pr_list "$@" ;;
  pr-comment)    cmd_pr_comment "$@" ;;
  pr-link)       cmd_pr_link "$@" ;;
  pr-merge)      cmd_pr_merge "$@" ;;
  branch-create) cmd_branch_create "$@" ;;
  milestone-list)   cmd_milestone_list "$@" ;;
  milestone-get)    cmd_milestone_get "$@" ;;
  milestone-create) cmd_milestone_create "$@" ;;
  help|-h|--help) usage ;;
  *) die "未知子命令: $cmd（运行 gitcode.sh help 查看）" ;;
esac
